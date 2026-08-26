// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

// Package cdp implements a Chrome DevTools Protocol bridge for ldin.
// It connects to a running Chrome instance via CDP WebSocket and executes
// JavaScript fetch() calls inside the real browser context, completely
// bypassing LinkedIn's TLS fingerprinting (JA3/BotManager) that blocks
// Go's net/http client.
//
// Architecture:
//
//	ldin CLI → CDP WebSocket → Chrome (real TLS) → LinkedIn Voyager API
package cdp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	DefaultCDPPort = 9222
	DefaultCDPHost = "localhost"
)

// CDPMessage represents a Chrome DevTools Protocol message
type CDPMessage struct {
	ID     int64                  `json:"id"`
	Method string                 `json:"method,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
	Result json.RawMessage        `json:"result,omitempty"`
	Error  *CDPError              `json:"error,omitempty"`
}

// CDPError is a CDP error response
type CDPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *CDPError) Error() string {
	return fmt.Sprintf("CDP error %d: %s", e.Code, e.Message)
}

// TabInfo describes a Chrome tab from the CDP /json endpoint
type TabInfo struct {
	ID                   string `json:"id"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	Type                 string `json:"type"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// Bridge is a live connection to Chrome's DevTools Protocol
type Bridge struct {
	conn     *websocket.Conn
	mu       sync.Mutex
	msgID    atomic.Int64
	pending  map[int64]chan *CDPMessage
	pendingM sync.Mutex
	done     chan struct{}
}

// ListTabs returns all open tabs from Chrome's CDP HTTP endpoint
func ListTabs(host string, port int) ([]*TabInfo, error) {
	url := fmt.Sprintf("http://%s:%d/json", host, port)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to Chrome CDP on port %d: %w\n\nStart Chrome with: open -a 'Google Chrome' --args --remote-debugging-port=%d", port, err, port)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tabs []*TabInfo
	if err := json.Unmarshal(body, &tabs); err != nil {
		return nil, fmt.Errorf("failed to parse CDP tab list: %w", err)
	}
	return tabs, nil
}

// FindLinkedInTab returns the best tab for LinkedIn requests
func FindLinkedInTab(tabs []*TabInfo) *TabInfo {
	// Prefer a LinkedIn tab already open
	for _, t := range tabs {
		if t.Type == "page" && strings.Contains(t.URL, "linkedin.com") {
			return t
		}
	}
	// Fall back to any page tab
	for _, t := range tabs {
		if t.Type == "page" && t.WebSocketDebuggerURL != "" {
			return t
		}
	}
	return nil
}

// Connect opens a CDP WebSocket connection to a specific tab
func Connect(ctx context.Context, tab *TabInfo) (*Bridge, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, tab.WebSocketDebuggerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("CDP WebSocket dial failed: %w", err)
	}

	b := &Bridge{
		conn:    conn,
		pending: make(map[int64]chan *CDPMessage),
		done:    make(chan struct{}),
	}

	go b.readLoop()
	return b, nil
}

func (b *Bridge) readLoop() {
	defer close(b.done)
	for {
		_, raw, err := b.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg CDPMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		if msg.ID > 0 {
			b.pendingM.Lock()
			ch, ok := b.pending[msg.ID]
			if ok {
				delete(b.pending, msg.ID)
			}
			b.pendingM.Unlock()
			if ok {
				ch <- &msg
			}
		}
	}
}

// Send dispatches a CDP command and waits for its response
func (b *Bridge) Send(ctx context.Context, method string, params map[string]interface{}) (*CDPMessage, error) {
	id := b.msgID.Add(1)

	msg := CDPMessage{
		ID:     id,
		Method: method,
		Params: params,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	ch := make(chan *CDPMessage, 1)
	b.pendingM.Lock()
	b.pending[id] = ch
	b.pendingM.Unlock()

	b.mu.Lock()
	err = b.conn.WriteMessage(websocket.TextMessage, data)
	b.mu.Unlock()
	if err != nil {
		b.pendingM.Lock()
		delete(b.pending, id)
		b.pendingM.Unlock()
		return nil, fmt.Errorf("CDP write failed: %w", err)
	}

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp, nil
	case <-ctx.Done():
		b.pendingM.Lock()
		delete(b.pending, id)
		b.pendingM.Unlock()
		return nil, ctx.Err()
	case <-b.done:
		return nil, fmt.Errorf("CDP connection closed")
	}
}

// Eval executes a JavaScript expression in the browser context and returns the string result
func (b *Bridge) Eval(ctx context.Context, expression string) (string, error) {
	resp, err := b.Send(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":            expression,
		"awaitPromise":          true,
		"returnByValue":         true,
		"timeout":               15000,
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Result struct {
			Type        string      `json:"type"`
			Value       interface{} `json:"value"`
			Description string      `json:"description"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("failed to parse CDP eval result: %w", err)
	}
	if result.ExceptionDetails != nil {
		return "", fmt.Errorf("JS exception: %s", result.ExceptionDetails.Text)
	}

	switch v := result.Result.Value.(type) {
	case string:
		return v, nil
	case nil:
		return "", nil
	default:
		raw, _ := json.Marshal(v)
		return string(raw), nil
	}
}

// FetchURL executes a fetch() inside Chrome, returning the raw response body.
// This request uses Chrome's real TLS stack and existing session cookies — 
// LinkedIn's bot detection sees it as a legitimate browser request.
func (b *Bridge) FetchURL(ctx context.Context, url string, extraHeaders map[string]string) (string, error) {
	headersJS := "{"
	for k, v := range extraHeaders {
		headersJS += fmt.Sprintf(`%q: %q,`, k, v)
	}
	headersJS += "}"

	js := fmt.Sprintf(`
(async () => {
  try {
    const resp = await fetch(%q, {
      method: 'GET',
      credentials: 'include',
      headers: Object.assign({
        'Accept': 'application/vnd.linkedin.normalized+json+2.1',
        'Accept-Language': 'en-US,en;q=0.9',
        'X-Li-Lang': 'en_US',
        'X-Li-Track': JSON.stringify({clientVersion: window.__ossBuildVersion || '1.0', osName: 'web'}),
        'X-Requested-With': 'XMLHttpRequest',
        'Csrf-Token': (document.cookie.match(/JSESSIONID="?([^";]+)"?/) || [])[1] || '',
      }, %s)
    });
    if (!resp.ok) { return JSON.stringify({__ldin_error: resp.status, __ldin_url: resp.url}); }
    const text = await resp.text();
    return text;
  } catch(e) {
    return JSON.stringify({__ldin_error: e.message});
  }
})()
`, url, headersJS)

	result, err := b.Eval(ctx, js)
	if err != nil {
		return "", err
	}

	// Check for error sentinel
	if strings.Contains(result, `"__ldin_error"`) {
		var errObj map[string]interface{}
		if json.Unmarshal([]byte(result), &errObj) == nil {
			if code, ok := errObj["__ldin_error"]; ok {
				return "", fmt.Errorf("fetch returned error: %v (url: %v)", code, errObj["__ldin_url"])
			}
		}
	}

	return result, nil
}

// NavigateTo opens a URL in the current tab (waits for page load)
func (b *Bridge) NavigateTo(ctx context.Context, url string) error {
	_, err := b.Send(ctx, "Page.navigate", map[string]interface{}{
		"url": url,
	})
	return err
}

// Close shuts down the CDP WebSocket connection
func (b *Bridge) Close() error {
	return b.conn.Close()
}

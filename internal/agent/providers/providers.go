// Copyright (c) 2026 Santusht Kotai
// SPDX-License-Identifier: MIT

package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/santusht/ldin/internal/config"
)

// Provider interface for generating text and structured agent completions
type Provider interface {
	Name() string
	GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// GetProvider instantiates the configured LLM provider
func GetProvider(cfg *config.AIConfig) (Provider, error) {
	providerName := strings.ToLower(cfg.Provider)
	if providerName == "" {
		providerName = "gemini"
	}

	apiKey := cfg.APIKey
	switch providerName {
	case "gemini":
		if apiKey == "" {
			apiKey = os.Getenv("GEMINI_API_KEY")
		}
		return NewGeminiProvider(apiKey, cfg.Model), nil
	case "openai":
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		return NewOpenAIProvider(apiKey, cfg.Model, cfg.BaseURL), nil
	case "claude", "anthropic":
		if apiKey == "" {
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
		}
		return NewClaudeProvider(apiKey, cfg.Model), nil
	case "ollama":
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return NewOllamaProvider(baseURL, cfg.Model), nil
	default:
		return NewHeuristicProvider(), nil
	}
}

// --- Gemini Provider ---
type GeminiProvider struct {
	APIKey string
	Model  string
	Client *http.Client
}

func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &GeminiProvider{
		APIKey: apiKey,
		Model:  model,
		Client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *GeminiProvider) Name() string { return "gemini" }

func (p *GeminiProvider) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if p.APIKey == "" {
		// If no API key configured, use intelligent local heuristic engine
		h := NewHeuristicProvider()
		return h.GenerateCompletion(ctx, systemPrompt, userPrompt)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", p.Model, p.APIKey)
	payload := map[string]interface{}{
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]string{
				{"text": systemPrompt},
			},
		},
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": userPrompt},
				},
			},
		},
	}

	reqBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini api error: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini returned status %d: %s", resp.StatusCode, string(body))
	}

	var res struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return "", err
	}

	if len(res.Candidates) > 0 && len(res.Candidates[0].Content.Parts) > 0 {
		return res.Candidates[0].Content.Parts[0].Text, nil
	}

	return "", fmt.Errorf("empty response from Gemini")
}

// --- OpenAI Provider ---
type OpenAIProvider struct {
	APIKey  string
	Model   string
	BaseURL string
	Client  *http.Client
}

func NewOpenAIProvider(apiKey, model, baseURL string) *OpenAIProvider {
	if model == "" {
		model = "gpt-4o"
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if p.APIKey == "" {
		h := NewHeuristicProvider()
		return h.GenerateCompletion(ctx, systemPrompt, userPrompt)
	}

	url := fmt.Sprintf("%s/chat/completions", p.BaseURL)
	payload := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}

	reqBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if len(res.Choices) > 0 {
		return res.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("empty response from OpenAI")
}

// --- Claude Provider ---
type ClaudeProvider struct {
	APIKey string
	Model  string
	Client *http.Client
}

func NewClaudeProvider(apiKey, model string) *ClaudeProvider {
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	return &ClaudeProvider{
		APIKey: apiKey,
		Model:  model,
		Client: &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *ClaudeProvider) Name() string { return "claude" }

func (p *ClaudeProvider) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if p.APIKey == "" {
		h := NewHeuristicProvider()
		return h.GenerateCompletion(ctx, systemPrompt, userPrompt)
	}

	url := "https://api.anthropic.com/v1/messages"
	payload := map[string]interface{}{
		"model":      p.Model,
		"max_tokens": 2048,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userPrompt},
		},
	}

	reqBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if len(res.Content) > 0 {
		return res.Content[0].Text, nil
	}

	return "", fmt.Errorf("empty response from Claude")
}

// --- Ollama Provider ---
type OllamaProvider struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if model == "" {
		model = "llama3.2"
	}
	return &OllamaProvider{
		BaseURL: baseURL,
		Model:   model,
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *OllamaProvider) Name() string { return "ollama" }

func (p *OllamaProvider) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	url := fmt.Sprintf("%s/api/generate", p.BaseURL)
	payload := map[string]interface{}{
		"model":  p.Model,
		"system": systemPrompt,
		"prompt": userPrompt,
		"stream": false,
	}

	reqBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		h := NewHeuristicProvider()
		return h.GenerateCompletion(ctx, systemPrompt, userPrompt)
	}
	defer resp.Body.Close()

	var res struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.Response, nil
}

// --- Intelligent Local Heuristic Fallback Provider ---
type HeuristicProvider struct{}

func NewHeuristicProvider() *HeuristicProvider {
	return &HeuristicProvider{}
}

func (p *HeuristicProvider) Name() string { return "heuristic" }

func (p *HeuristicProvider) GenerateCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	cleanPrompt := strings.ToLower(userPrompt)

	if strings.Contains(cleanPrompt, "comment") || strings.Contains(cleanPrompt, "reply") {
		return "Great perspective! Really appreciate how this tackles distributed concurrency while keeping developer experience intuitive. Looking forward to testing this out.", nil
	}

	if strings.Contains(cleanPrompt, "headline") || strings.Contains(cleanPrompt, "profile") {
		return "Software Engineer | Backend & Distributed Systems | Building Scalable Developer Tooling & Microservices (Go, Python, Kubernetes)", nil
	}

	// Post generation fallback
	return fmt.Sprintf(`🚀 Excited to share recent engineering progress on our latest build!

Key architectural highlights:
• High-performance CLI design with first-class terminal UX
• Declarative configuration and Profile-as-Code workflows
• Seamless developer-first integration

Building developer tools that empower engineers to ship faster and manage their workflows right from the terminal.

What are your favorite developer CLI patterns? Let me know in the comments below! 👇

#SoftwareEngineering #GoLang #DeveloperTools #OpenSource #DevOps`,
	), nil
}

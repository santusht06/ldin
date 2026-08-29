# `ldin` — The Developer-First LinkedIn CLI & AI Agent Platform

<div align="center">

```
  _       _ _       
 | |   __| (_)_ __  
 | |  / _` | | '_ \ 
 | | | (_| | | | | |
 |_|  \__,_|_|_| |_|
```

**Manage your professional identity, Profile-as-Code, content publishing, and LinkedIn workflows from the terminal.**

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![LinkedIn API](https://img.shields.io/badge/LinkedIn_API-v202608-0A66C2?style=flat&logo=linkedin)](https://learn.microsoft.com/en-us/linkedin/)

</div>

---

## 📦 Installation

### Prerequisites
- Go 1.25 or later
- GNU Make

### Build from source
```bash
git clone https://github.com/santusht/ldin.git
cd ldin
make build
```

### Install binary to your PATH
```bash
make install
```

The `make install` target places the `ldin` binary in `/usr/local/bin` (or `$GOPATH/bin` if `$GOPATH` is set).

---

## 🌟 Philosophy

> **"Simple things should be simple. Complex things should be possible."**
>
> `gh` manages your code identity. `ldin` manages your professional identity.

`ldin` is not a scraper or a spam bot. It is a developer‑first command‑line workspace built around **official LinkedIn REST APIs**, **Profile-as-Code**, and an **autonomous AI agent layer**.

---

## 🏛️ Architecture

```
                         ┌───────────────────────┐
                         │        ldin           │
                         │ LinkedIn CLI Platform │
                         └───────────┬───────────┘
                                     │
            ┌────────────────────────┼────────────────────────┐
            │                        │                        │
       Human CLI                AI Agent                 Raw API
            │                        │                        │
      ldin post...             ldin ai ...              ldin api ...

### 1. Build & Install

```bash
# Clone and build
git clone https://github.com/santusht/ldin.git
cd ldin
make build

# Or install directly to PATH
make install
```

### 2. Authenticate

```bash
# Interactive OAuth 2.0 PKCE in browser
ldin auth login

# Or direct access token for CI/CD
ldin auth login --token <your_token>

# Check authentication status & granted scopes
ldin auth status
```

### 3. Check Live API Capabilities

```bash
ldin capabilities
```

Surfaces which endpoints are open for self-service vs those requiring LinkedIn Community Management approval.

---

## 📑 Feature Showcase

### 1. LinkedIn Profile-as-Code (`ldin profile`)

Treat your career profile just like infrastructure as code:

```bash
# Export active profile to declarative YAML
ldin profile export -o profile.yaml

# Audit & lint your profile for SEO, character limits, and section strength
ldin profile validate --file profile.yaml

# View colored terminal diff between local YAML and live profile
ldin profile diff --file profile.yaml

# Ask AI to optimize your headline and experience for backend/distributed systems
ldin profile optimize --file profile.yaml

# Edit in your configured $EDITOR
ldin profile edit
```

Example `profile.yaml`:
```yaml
name: Santusht Kotai
headline: Software Engineer | Backend Engineering | Distributed Systems
location: Indore, India
about: |
  Backend-focused Software Engineer passionate about scalable distributed systems
  and developer tooling.
skills:
  - Go
  - Python
  - FastAPI
  - PostgreSQL
  - Redis
  - Docker
  - Kubernetes
experience:
  - company: ShareXpress Systems
    role: Software Engineer
    start_date: 2024-01
    end_date: Present
    description: Architecting high throughput microservices and open source developer platforms.
```

---

### 2. Post Lifecycle & Terminal Previews (`ldin post`)

```bash
# Render a rich terminal preview before publishing
ldin post preview "Building high performance developer tools in Go! 🚀 #golang"

# Publish instantly
ldin post create "Excited to open source ldin!"

# Attach image / PDF document
ldin post create --file ./post.md --image ./architecture.png

# Interactive Poll
ldin post create --poll "Primary backend runtime?" --options "Go,Rust,Python,Java"

# Offline drafts
ldin post draft "Work in progress draft..."
ldin post list
ldin post publish draft-1787751791
```

---

### 3. AI Agent Layer & Context Engine (`ldin ai` / `ldin agent`)

Instead of copying text between browser tabs, `ldin` knows your engineering context:

```bash
# Sync local Git or GitHub repository context
ldin repo sync .
ldin repo sync santusht06/interleet

# Turn code contributions into a LinkedIn post
ldin ai "Write a LinkedIn post about my latest commits and architecture trade-offs"

# Craft technical replies to comments
ldin ai reply urn:li:comment:123 "Thank them for feedback on distributed caching"

# Autonomous agent task execution
ldin agent run "Audit my profile skills and suggest a technical post about Go concurrency"
```

#### Agent Safety Sandbox
```bash
# Inspect agent permissions
ldin agent permissions

# Explicitly grant or revoke live publish authority
ldin agent allow publish
ldin agent deny publish
```

---

### 4. Raw API Escape Hatch (`ldin api`)

When LinkedIn releases a new endpoint or API version, you don't need to wait for CLI updates:

```bash
# Query any REST endpoint
ldin api GET /v2/userinfo
ldin api GET /rest/posts/urn%3Ali%3Ashare%3A71982341234

# POST raw payload with custom version header
ldin api POST /rest/posts --body @post.json -H "Linkedin-Version: 202608"
```

---

### 5. Multi-Identity & Unix Scripting (`--json`, `--profile`)

```bash
# Manage multiple identities (e.g. personal vs company)
ldin auth login --name company
ldin --profile company post create "Company update..."

# Seamless piping into jq
ldin capabilities --json | jq '.[] | select(.available == true)'
ldin post list --json | jq '.published[].id'
```

---

## 📚 Complete Command Reference

| Command Group | Subcommands | Description |
| :--- | :--- | :--- |
| `ldin auth` | `login`, `logout`, `status`, `refresh`, `scopes`, `whoami`, `switch` | Multi-identity authentication & tokens |
| `ldin capabilities` | `capabilities` | Live LinkedIn API capability & scope matrix |
| `ldin profile` | `get`, `show`, `export`, `import`, `diff`, `validate`, `optimize`, `sync`, `edit` | LinkedIn Profile-as-Code engine |
| `ldin post` | `create`, `draft`, `publish`, `preview`, `list`, `get`, `delete`, `text`, `image`, `video`, `document`, `poll` | Multi-format post publishing & drafts |
| `ldin comment` | `create`, `reply`, `list`, `delete` | Comments and nested thread conversations |
| `ldin reaction` | `like`, `react`, `unlike` | Send and manage reactions (`LIKE`, `PRAISE`, `EMPATHY`, etc.) |
| `ldin social` | `summary` | Aggregated engagement counts and current user status |
| `ldin media` | `upload` | 3-step media asset upload protocol |
| `ldin analytics` | `profile`, `post`, `posts` | Impressions, reach, and engagement time-series |
| `ldin org` | `list`, `post` | Manage LinkedIn Company Pages |
| `ldin event` | `list`, `create` | LinkedIn live audio/video events |
| `ldin ads` | `accounts`, `campaigns` | Marketing Developer Platform campaigns |
| `ldin repo` | `sync` | Git & GitHub context extraction |
| `ldin ai` | `post`, `profile`, `reply` | Intelligent content generation & copilot |
| `ldin agent` | `run`, `permissions`, `allow`, `deny`, `tools` | Autonomous ReAct agent loop |
| `ldin config` | `get`, `set`, `list`, `path` | Configuration management (`~/.ldin/config.yaml`) |
| `ldin api` | `GET`, `POST`, `PUT`, `PATCH`, `DELETE` | Raw authenticated REST API client |

---

## 🛠️ Configuration

Stored in `~/.ldin/config.yaml`:
```yaml
version: "1.0.0"
active_profile: "default"
output_format: "human"
linkedin_api_version: "202608"
editor: "nano"
ai:
  provider: "gemini"        # gemini, openai, claude, ollama
  model: "gemini-2.5-flash"
agent:
  auto_publish: false
  allowed_scopes:
    - read
    - draft
    - ai
```

---

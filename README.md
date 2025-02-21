# Context-Keep

Context-Keep is an open-source **hierarchical memory system** for LLM-based applications. It efficiently manages structured context in **Redis**, allowing for dynamic retrieval of conversation events and summaries. The system is designed to optimize **long-term memory** and **context-aware responses** while keeping computation lightweight.

## Features

- **Hierarchical Memory in Redis**
  - Stores events as a structured tree (e.g., `event -> summary -> raw content`).
  - Enables efficient retrieval of contextual information for LLM queries.
- **Summarization & Context Generation**
  - Dynamically summarizes stored events to optimize context for LLMs.
  - Supports multiple levels of abstraction (e.g., conversation-wide summaries, topic-based summaries).
- **Go-based Service**
  - High-performance, concurrent memory manager.
  - REST/gRPC API for interacting with stored context.
- **CLI for Testing & Debugging**
  - Manages memory, queries events, and triggers summarization.
- **Pluggable LLM Backend**
  - Works with local or remote LLM APIs (e.g., DeepSeek, LM Studio, OpenAI).

## Repository Structure

```
context-keep/
├── go-service/            # Go-based service for hierarchical memory + API
│   ├── cmd/               # CLI commands for interacting with memory
│   ├── internal/          # Internal service logic (memory, event handling, etc.)
│   ├── main.go            # Entry point for Go service
│   ├── go.mod             # Go dependencies
│   ├── go.sum             # Go dependencies lockfile
│   └── README.md
├── redis/                 # Redis schemas, setup, and tooling
│   ├── redis-schema.txt   # Hierarchical Redis key structure
│   ├── docker-compose.yml # Redis setup
│   ├── README.md
├── docs/                  # Documentation for the project
├── cli/                   # CLI tool for interacting with context
├── .gitignore
├── README.md
└── LICENSE
```

## Getting Started

### Prerequisites
- **Go** (>=1.21)
- **Redis** (>=6.0, for hierarchical key storage)
- **LLM API** (DeepSeek, LM Studio, OpenAI, etc.)

### Installation
```sh
# Clone the repository
$ git clone https://github.com/your-username/context-keep.git
$ cd context-keep
```

### Running Redis (Docker Compose)
```sh
$ docker-compose up -d
```

### Running the Go Service
```sh
$ cd go-service
$ go run main.go
```

## Usage

### API Endpoints
- `POST /events` - Store an event in memory
- `GET /events/{id}` - Retrieve event data
- `POST /summarize` - Trigger hierarchical summarization

### CLI Commands
```sh
$ go run main.go store --event "User asked about Redis"
$ go run main.go retrieve --id 123
$ go run main.go summarize --id 123
```

## Roadmap
- [ ] Implement Redis-backed hierarchical memory storage
- [ ] Expose REST API for storing and retrieving context
- [ ] CLI for manual testing and debugging
- [ ] Summarization pipeline for LLM context compression

## Contributing
PRs and contributions are welcome! Open an issue or submit a PR if you’d like to help.


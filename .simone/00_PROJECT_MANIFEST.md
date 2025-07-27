# PROJECT MANIFEST

## Project Name
CodeDoc MCP Server

## Project Description
An intelligent codebase documentation system that automatically generates, maintains, and verifies documentation using Zettelkasten methodology. Built as a Model Context Protocol (MCP) server in Go, it enables AI assistants to create hierarchical documentation that evolves with the codebase while respecting token limitations.

## Project Status
- Created: 2025-07-27
- Status: Active (Implementation Phase)
- Version: 0.0.1
- Current Milestone: M01 - Core Services
- Current Sprint: S01 - Foundation and Orchestrator Core
- Current Task: T03 - Workflow State Machine (in_progress)
- Highest Milestone: M01
- Last Updated: 2025-07-27 22:46

## Technical Overview
- **Language**: Go 1.24+
- **Framework**: MCP (Model Context Protocol) via mark3labs/mcp-go
- **Architecture**: Clean Architecture with DI
- **Storage**: PostgreSQL 15+, ChromaDB, Local filesystem
- **AI Integration**: OpenAI GPT-4, Google Gemini Pro

## Key Features
- Automated hierarchical documentation generation
- Zettelkasten memory system for knowledge management
- Multi-model consensus validation
- Token-aware architecture (25K limit compliance)
- Documentation verification against code

## Project Structure
```
codedoc-mcp-server/
├── .simone/              # Simone project management
├── cmd/server/           # Application entry point
├── internal/             # Private application code
├── pkg/                  # Public packages
├── configs/              # Configuration files
├── migrations/           # Database migrations
├── docs/                 # Project documentation (ADRs)
└── doc/                  # Generated documentation output
```

## Development Phases
1. **Foundation Setup** (Complete) - Infrastructure and base implementation
2. **Core Services** (Current - M01) - Orchestrator and file services
3. **AI Integration** - OpenAI and Gemini clients
4. **Zettelkasten System** - Memory and evolution services
5. **Documentation Generation** - Writer and hierarchy
6. **Verification & Consensus** - Quality assurance
7. **Production Readiness** - Security and deployment

## Milestones
- [x] milestone_001_foundation - Foundation Setup
- [ ] [M01: Core Services Implementation](02_REQUIREMENTS/M01_Core_Services/M01_milestone_meta.md) - Status: Planning

## Team
- AI Assistant: Claude (Opus 4)
- Human: nixlim
- Framework: Simone Project Management

## Repository
- Location: /Users/nixlim/Documents/codedoc
- VCS: Git (branch: master)

---
*Managed by Simone Project Management System*
*Last Updated: 2025-07-27*

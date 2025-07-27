# CodeDoc MCP Server Architecture

## Project Overview

**CodeDoc MCP Server** is an intelligent codebase documentation system that automatically generates, maintains, and verifies documentation using Zettelkasten methodology. It operates within the Model Context Protocol (MCP) framework, providing AI assistants with tools to create hierarchical documentation that evolves with the codebase.

## Core Purpose

The system addresses the critical challenge of keeping documentation synchronized with code changes by:
- Automatically traversing codebases to generate comprehensive documentation
- Verifying existing documentation against actual implementation
- Building interconnected knowledge networks using Zettelkasten principles
- Leveraging multi-model AI consensus for quality assurance

## Architecture Principles

### 1. Token-Aware Design
- Respects MCP's 25,000 token limit per exchange
- Passes only file paths between server and AI agents
- Each component accesses files directly

### 2. Clean Architecture
- Separation of concerns with clear boundaries
- Dependency injection
- Domain-driven design with rich models

### 3. Hierarchical Documentation
- **File-Level**: Individual source file documentation
- **Component-Level**: Aggregated documentation for related files
- **System-Level**: High-level architecture documentation

### 4. Multi-Model Intelligence
- Software Architect persona for design validation
- Technical Writer persona for clarity and completeness
- Staff Engineer persona for implementation accuracy

## System Components

### MCP Protocol Handler
- Routes incoming MCP requests to appropriate handlers
- Enforces token limits and validates requests
- Provides detailed error responses with recovery hints

### Documentation Orchestrator
- Manages complex documentation workflows
- Creates and tracks TODO lists for AI agents
- Coordinates between different services

### Zettelkasten Memory System
- Stores atomic documentation units (memories)
- Maintains bidirectional relationships between memories
- Enables organic evolution of knowledge network
- Implements semantic search using ChromaDB

### Verification Service
- Compares documentation against actual code
- Identifies discrepancies and outdated sections
- Generates actionable reports for updates

### Consensus Engine
- Orchestrates multi-model reviews
- Implements majority voting logic
- Tracks quality scores and feedback

### Documentation Writer
- Generates markdown documentation files
- Maintains proper directory structure
- Handles cross-references and linking

## Technology Stack

### Core Technologies
- **Language**: Go 1.24+
- **MCP Framework**: mark3labs/mcp-go 
- **Logging**: zerolog (structured JSON)

### Storage Layer
- **Vector Database**: ChromaDB for embeddings and semantic search
- **Relational DB**: PostgreSQL 15+ for metadata
- **File Storage**: Local filesystem with validation

### AI Integration
- **OpenAI GPT-4**: Primary analysis and memory evolution
- **Google Gemini Pro**: Large context summarization (1M tokens)

### Development Tools
- **Testing**: Go native + testify
- **DI Framework**: Google Wire
- **Hot Reload**: air
- **Monitoring**: Prometheus

## Data Flow

### Full Documentation Flow
1. AI requests project structure
2. Server analyzes and groups files thematically
3. AI processes each theme/module
4. Documentation memories are created and evolved
5. Consensus review validates quality
6. Final documentation is generated

### Verification Flow
1. Load existing documentation
2. Compare with current code implementation
3. Identify discrepancies
4. Generate update recommendations

## Key Design Patterns

### Repository Pattern
- Abstracts data access for all storage systems
- Enables easy testing and swapping of implementations

### Strategy Pattern
- Flexible LLM provider switching
- Configurable consensus strategies

### Observer Pattern
- Tracks memory evolution events
- Enables reactive documentation updates

### Chain of Responsibility
- Request validation pipeline
- Error handling cascade

## Security Considerations

### Path Validation
- Strict validation of file paths
- Workspace isolation
- Prevention of directory traversal

### Rate Limiting
- Per-session limits
- Token budget enforcement
- Graceful degradation

### Audit Logging
- All operations logged with context
- Structured logs for analysis
- Security event tracking

## Deployment Architecture

### Containerization
- Multi-stage Docker builds
- Minimal runtime images
- Health check endpoints

### Configuration
- Environment-based configuration
- Secrets management
- Feature flags for gradual rollout

### Monitoring
- Prometheus metrics for all operations
- Custom dashboards for documentation quality
- Alert rules for system health

## Error Handling Strategy

### Structured Error Responses
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable message",
    "details": {},
    "hint": "Recovery suggestion"
  }
}
```

### Error Categories
- `TOKEN_LIMIT_EXCEEDED`: MCP token constraints
- `INVALID_PATH`: File system errors
- `LLM_ERROR`: AI service failures
- `CONSENSUS_FAILED`: Quality validation issues

## Future Considerations

### Scalability
- Distributed processing for large codebases
- Caching strategies for common queries
- Incremental documentation updates

### Extensibility
- Plugin system for custom documentation formats
- Additional LLM provider support
- Custom consensus strategies

### Integration
- IDE plugins for real-time documentation
- CI/CD pipeline integration
- Git hooks for documentation validation
# Project Brief: CodeDoc MCP Server

## Project Overview
CodeDoc MCP Server is an intelligent codebase documentation system that leverages the Model Context Protocol (MCP) to generate comprehensive, token-aware documentation. The system uses a Zettelkasten-inspired memory network for knowledge management and semantic search capabilities.

## Core Requirements

### Functional Requirements
1. **Automated Documentation Generation**
   - Process codebases to generate documentation within 25,000 token MCP limit
   - Support multiple programming languages with language-specific parsers
   - Generate structured documentation (functions, classes, modules)
   - Create intelligent summaries and cross-references

2. **Memory Management System**
   - Implement Zettelkasten methodology for knowledge organization
   - Vector storage using ChromaDB for semantic search
   - Memory evolution and pattern recognition
   - Workspace isolation for multi-tenant support

3. **MCP Protocol Integration**
   - Full MCP v2024.11 compliance
   - Support for tools: docAnalyze, docGenerate, docSearch, memoryStore, memoryRetrieve
   - Handle 25,000 token context limit intelligently
   - Proper error handling with MCP format

### Non-Functional Requirements
1. **Performance**
   - Process large codebases efficiently
   - Sub-second response for search queries
   - Concurrent file processing

2. **Security**
   - Rate limiting (100 req/min per workspace)
   - Path traversal prevention
   - Comprehensive audit logging
   - Workspace isolation

3. **Scalability**
   - PostgreSQL for persistent storage
   - ChromaDB for vector operations
   - Clean architecture for maintainability

## Project Goals
1. Create a production-ready MCP server for intelligent code documentation
2. Implement advanced memory management with semantic search
3. Ensure robust security and performance
4. Provide excellent developer experience

## Success Criteria
- All MCP tools implemented and functional
- Documentation generation within token limits
- Semantic search with >90% relevance
- Security measures preventing all common attacks
- Comprehensive test coverage (>70%)
- Clear documentation and deployment guides

## Constraints
- Must use Go 1.24+ with mark3labs/mcp-go library
- PostgreSQL 15+ for data persistence
- ChromaDB for vector storage
- Must respect 25,000 token MCP limit
- Follow clean architecture principles

## Timeline
- Milestone M01: Core Services (Current)
  - Sprint S01: Foundation & Orchestrator
  - Sprint S02: MCP Protocol Implementation
  - Sprint S03: FileSystem & Integration
- Future milestones: Zettelkasten, AI Integration, UI
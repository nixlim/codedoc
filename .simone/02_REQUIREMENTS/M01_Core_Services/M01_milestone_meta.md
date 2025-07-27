---
milestone_id: M01
title: Core Services Implementation
status: planning
created: 2025-07-27
last_updated: 2025-07-27 03:05
---

# M01: Core Services Implementation

## Overview
Build the foundational service layer that orchestrates documentation workflows, handles MCP protocol routing, and manages file system operations with proper validation and security. This milestone transforms the basic MCP server from the foundation phase into a fully functional service layer capable of managing complex documentation workflows.

## Goals
1. **Implement Documentation Orchestrator** - Create the central service that manages documentation workflows, tracks sessions, and coordinates between different components
2. **Enhance MCP Protocol Handler** - Upgrade from basic placeholders to full routing implementation for all 8 MCP tools with proper token management
3. **Build File System Service** - Develop secure file operations with path validation, workspace isolation, and efficient traversal
4. **Establish Service Integration** - Ensure all services work together seamlessly with proper error handling and state management

## Key Documents
- PRD_Core_Services.md - Product requirements
- SPECS_Orchestrator.md - Technical specifications for orchestrator
- SPECS_MCP_Handler.md - Protocol implementation details
- SPECS_FileSystem.md - File service specifications

## Definition of Done
- [ ] All 8 MCP tool endpoints return valid, functional responses (not placeholders)
- [ ] Documentation Orchestrator successfully manages a complete workflow from start to finish
- [ ] File system operations are secure with 100% path validation test coverage
- [ ] Integration tests pass for all service-to-service interactions
- [ ] Error responses follow the structured format with actionable recovery hints
- [ ] API documentation exists for all public interfaces
- [ ] Performance benchmarks show <100ms response time for file operations
- [ ] Token counting is accurate within 1% margin of actual OpenAI tokenization
- [ ] All services use dependency injection and follow Clean Architecture principles
- [ ] Logging captures all significant operations with appropriate context

## Technical Requirements
### Documentation Orchestrator
- Session management with UUID tracking
- Workflow state machine (idle → processing → complete/failed)
- TODO list creation and tracking for AI agents
- Inter-service communication via interfaces
- Concurrent session support with isolation

### MCP Protocol Handler
- Complete routing implementation for:
  - `full_documentation` - Full codebase analysis
  - `provide_thematic_groupings` - AI callback for file organization
  - `provide_dependency_files` - AI callback for dependencies
  - `create_documentation` - Module documentation entry point
  - `analyze_file_callback` - AI file analysis results
  - `get_project_structure` - Project tree without contents
  - `verify_documentation` - Doc vs code comparison
  - `get_documentation_status` - Progress tracking
- Token counting with OpenAI tiktoken library
- Request validation middleware
- Structured error responses

### File System Service
- Path validation using filepath.Clean and whitelist approach
- Workspace isolation (no access outside project root)
- Efficient recursive traversal with .gitignore support
- File type detection and filtering
- Concurrent-safe operations

## Dependencies
- Foundation milestone completed:
  - Basic MCP server running
  - PostgreSQL database with migrations
  - Project structure established
  - Logging infrastructure (zerolog)
  - Docker environment configured
- Go modules installed and building successfully
- Health check endpoints operational

## Risks & Mitigations
- **MCP Protocol Complexity**: Limited documentation → Study mark3labs/mcp-go examples, create comprehensive test harness
- **Token Counting Accuracy**: Critical for MCP limits → Implement multiple strategies, validate against OpenAI's tokenizer
- **File System Security**: Path traversal vulnerabilities → Use Go's filepath.Clean, implement strict whitelist validation
- **Service Coordination**: Complex state management → Use context propagation, implement circuit breakers for resilience
- **Performance at Scale**: Large codebases → Implement streaming for file operations, use goroutines for parallel processing

## Success Metrics
- All unit tests passing with >80% coverage
- Integration test suite covers all workflows
- Zero security vulnerabilities in path validation
- Documentation workflow completes in <30s for medium projects
- Error rate <0.1% in normal operations

## Notes/Context
This milestone establishes the core service infrastructure that all future features will build upon. The focus is on robustness, security, and clean architecture rather than advanced features. The orchestrator pattern will enable complex multi-step workflows required for documentation generation while maintaining clean separation of concerns.

The implementation should prioritize:
1. Security (especially path validation)
2. Clean interfaces for future extensibility
3. Comprehensive error handling
4. Performance for large codebases
5. Clear documentation for other developers

Integration with AI services (OpenAI, Gemini) will come in the next milestone - this milestone focuses on the service infrastructure.

## Sprint Breakdown
The milestone is divided into 3 sprints:

1. **[S01: Foundation and Orchestrator Core](../../03_SPRINTS/M01_Core_Services/S01_Foundation_Orchestrator/S01_sprint_plan.md)** (1 week)
   - Establishes orchestrator infrastructure
   - Implements session management and state machine
   - Creates TODO list system and database schema

2. **[S02: MCP Protocol Implementation](../../03_SPRINTS/M01_Core_Services/S02_MCP_Protocol/S02_sprint_plan.md)** (1 week)
   - Implements all 8 MCP tool handlers
   - Adds token counting and validation
   - Integrates with orchestrator

3. **[S03: File System Service and Integration](../../03_SPRINTS/M01_Core_Services/S03_FileSystem_Integration/S03_sprint_plan.md)** (1.5 weeks)
   - Builds secure file system operations
   - Completes service integration
   - Ensures end-to-end workflow functionality

Total Duration: 3.5 weeks
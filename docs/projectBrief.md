# CodeDoc MCP Server Specification
## Intelligent Codebase Documentation System with Zettelkasten Memory

Version 1.0.1  
Date: July 2025

**Changelog:**
- v1.0.1: Updated to use zerolog for structured logging and Go 1.24+
- v1.0.0: Initial specification

---

## Executive Summary

### Overview
The CodeDoc MCP Server enables AI assistants to intelligently traverse codebases, generate comprehensive documentation, and verify existing documentation accuracy using Zettelkasten methodology. The system addresses the critical challenge of maintaining accurate, hierarchical documentation that evolves with the codebase while respecting MCP protocol token limitations.

### Key Concepts

**Zettelkasten Memory System**: A knowledge management approach where atomic notes (memories) are interconnected, enabling organic growth of understanding through relationships rather than rigid hierarchies.

**Token-Aware Architecture**: Due to MCP's 25,000 token limit per exchange, the system passes only file paths between server and AI agent, with each component accessing files directly.

**Consensus-Based Validation**: Multiple AI models with distinct personas (Software Architect, Technical Writer, Staff Engineer) review and validate documentation quality through majority voting.

**Hierarchical Documentation Generation**:
1. **File-Level**: Documentation for individual source files
2. **Component-Level**: Aggregated documentation for related files
3. **System-Level**: High-level architecture documentation

### Example Interaction Flow

```
User: "Document the authentication module in our codebase"

1. AI requests codebase structure via `get_project_structure`
2. Server returns file paths and directory tree
3. AI identifies authentication-related files
4. For each file:
   - AI requests `analyze_file` with file path
   - Server returns analysis metadata (not content)
   - AI reads file directly and generates documentation
   - AI stores documentation via `store_documentation`
5. AI requests `aggregate_documentation` for component-level docs
6. Server orchestrates consensus review with multiple models
7. Final documentation is stored and indexed
```

### Design Principles

1. **Token Efficiency**: Never pass file contents through MCP protocol
2. **Incremental Understanding**: Build knowledge bottom-up from files to systems
3. **Verification First**: Validate existing docs before generating new ones
4. **Multi-Model Intelligence**: Leverage diverse AI perspectives for quality
5. **Evolutionary Memory**: Documentation improves through memory network evolution

---
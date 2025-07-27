## Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        AI Assistant (Claude)                     │
│  ┌─────────────────┐  ┌──────────────────┐  ┌──────────────┐  │
│  │ Code Discovery  │  │ File Analysis    │  │ File Reader  │  │
│  │    Engine       │  │    Engine        │  │  Component   │  │
│  └────────┬────────┘  └────────┬─────────┘  └──────┬───────┘  │
│           │                    │                     │          │
└───────────┼────────────────────┼─────────────────────┼──────────┘
            │                    │                     │
            │      MCP Protocol (JSON-RPC)            │
            │      Max 25k tokens per exchange        │
            ▼                    ▼                     ▼
┌─────────────────────────────────────────────────────────────────┐
│                    CodeDoc MCP Server                            │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   MCP Protocol Handler                    │   │
│  │                  (mark3labs/mcp-go)                      │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Documentation Orchestrator                   │   │
│  │  ┌──────────────┐  ┌───────────────┐  ┌─────────────┐  │   │
│  │  │ TODO List    │  │ AI Prompter   │  │  Progress   │  │   │
│  │  │ Manager      │  │               │  │  Tracker    │  │   │
│  │  └──────────────┘  └───────────────┘  └─────────────┘  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────┐  ┌───────────────┐  ┌───────────────────┐   │
│  │ Verification  │  │   Consensus   │  │  Documentation    │   │
│  │   Service     │  │    Engine     │  │    Writer         │   │
│  └──────┬───────┘  └───────┬───────┘  └─────────┬─────────┘   │
│         │                  │                     │              │
│  ┌──────▼───────────────────▼──────────────────▼──────────┐   │
│  │              Zettelkasten Memory System                  │   │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────┐   │   │
│  │  │   Memory    │  │   Memory     │  │   Memory     │   │   │
│  │  │   Store     │  │  Evolution   │  │  Retrieval   │   │   │
│  │  └─────────────┘  └──────────────┘  └──────────────┘   │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌────────────────┐  ┌─────────────────┐  ┌───────────────┐   │
│  │  File System   │  │   Vector DB     │  │  LLM Clients  │   │
│  │   Service      │  │   (ChromaDB)    │  │ OpenAI/Gemini │   │
│  └────────────────┘  └─────────────────┘  └───────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Core Components

#### 1. MCP Protocol Handler
- Implements MCP specification using mark3labs/mcp-go
- Manages tool registration and request routing
- **MUST provide detailed error responses for every invalid request**
- Handles error responses with detailed recovery hints
- Enforces token limits on responses

#### 2. Documentation Orchestrator
- **TODO List Manager**: Creates and manages documentation workflow from file list
- **AI Prompter**: Sends file paths to AI for analysis one by one
- **Progress Tracker**: Monitors documentation progress and handles failures
- Coordinates the entire documentation pipeline

#### 3. Documentation Writer
- Aggregates individual file analyses into comprehensive documentation
- Generates markdown files with proper structure
- Creates hierarchical documentation (file → component → system)
- Manages documentation versioning
- **Outputs final documentation to `doc/` directory as `.md` files**
- Creates index files for navigation
- Generates cross-references between related components

#### 4. Verification Service
- Compares existing documentation against code
- Identifies outdated or incorrect statements
- Detects mocked functionality claims
- Generates discrepancy reports with line references

#### 5. Consensus Engine
- Manages multiple AI model personas
- Orchestrates review rounds
- Implements majority voting logic
- Handles disagreement resolution

#### 6. Zettelkasten Memory System
- Stores atomic documentation units
- Manages bidirectional links between memories
- Implements memory evolution with LLMs
- Provides semantic search capabilities

#### 7. File System Service
- Provides secure file access
- Generates file metadata without content
- Manages workspace boundaries
- Implements path validation

### Data Flow

```
1. Full Documentation Flow:
   AI → full_documentation(workspace_id) → Server
   Server → request_thematic_groupings → AI
   AI → provide_thematic_groupings([grouped_paths]) → Server
   Server → creates multiple TODO lists by theme
   For each theme:
      For each file in theme:
         Server → request_file_analysis(file_path) → AI
         AI → analyze_file_callback(analysis) → Server
         Server → creates Zettelkasten note
         If dependencies found:
            Server → request_dependency_files(file) → AI
            AI → provide_dependency_files(deps) → Server
            Server → adds to TODO list
   Server → refine notes → evolve memory network
   Server → generate comprehensive documentation by theme
   Server → create system-wide documentation
   Server → [full documentation complete] → AI

2. Module Documentation Flow:
   AI → create_documentation([file_paths]) → Server
   Server → creates TODO list from file paths
   For each file in TODO list:
      Server → request_file_analysis(file_path) → AI
      AI → [reads file directly] → analyzes
      AI → analyze_file_callback(analysis) → Server
      Server → creates Zettelkasten note → Vector DB
   Server → refine notes → evolve memory network
   Server → generate comprehensive documentation
   Server → [documentation complete] → AI

3. Documentation Verification Flow:
   AI → verify_documentation(doc_path, code_paths) → Server
   Server → [loads documentation] → Server
   Server → [compares via LLM] → Server
   Server → [discrepancy report] → AI

4. Consensus Review Flow (Server-Initiated):
   Server → [doc to review] → Gemini Pro
   Gemini → [personas evaluate] → votes
   Server → [aggregates votes] → result
   Server → [stores consensus] → Memory

5. Documentation Storage Flow:
   Server → [final documentation] → File System
   Server → [creates .md files] → doc/ directory
   Server → [indexes content] → Vector DB
```

---
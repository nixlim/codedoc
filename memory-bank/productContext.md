# Product Context: CodeDoc MCP Server

## Why This Project Exists

### The Problem
Developers struggle with maintaining up-to-date documentation for large codebases. Traditional documentation tools:
- Require manual updates that often lag behind code changes
- Generate overwhelming amounts of text that exceed AI context limits
- Lack semantic understanding of code relationships
- Don't evolve with the codebase over time

### The Solution
CodeDoc MCP Server provides intelligent, context-aware documentation that:
- Automatically analyzes and documents code changes
- Respects MCP's 25,000 token limit through smart summarization
- Uses vector storage for semantic search across documentation
- Implements Zettelkasten methodology for knowledge evolution

## How It Should Work

### User Experience Flow
1. **Developer connects CodeDoc to their IDE** via MCP protocol
2. **Requests documentation** for a file, module, or entire codebase
3. **CodeDoc analyzes** the code structure and relationships
4. **Generates concise documentation** within token limits
5. **Stores insights** in the memory network for future reference
6. **Enables semantic search** across all documentation

### Key User Interactions
- `docAnalyze`: Analyze code structure and complexity
- `docGenerate`: Create documentation for specified scope
- `docSearch`: Find relevant documentation semantically
- `memoryStore`: Save important insights
- `memoryRetrieve`: Access historical knowledge

## User Experience Goals

### Primary Goals
1. **Effortless Documentation**: One command to document any code
2. **Always Current**: Documentation updates with code changes
3. **Context-Aware**: Understands relationships between components
4. **Searchable Knowledge**: Find information semantically, not just by keywords
5. **Learning System**: Improves documentation quality over time

### Quality Attributes
- **Speed**: Sub-second responses for most operations
- **Accuracy**: Correctly identifies code patterns and relationships
- **Conciseness**: Fits within MCP token limits without losing essential information
- **Intelligence**: Provides insights, not just descriptions
- **Security**: Respects workspace boundaries and access controls

## Target Users

### Primary Users
- **Software Development Teams**: Need living documentation
- **Open Source Maintainers**: Document projects for contributors
- **Enterprise Developers**: Comply with documentation requirements
- **DevOps Engineers**: Understand system architecture

### Use Cases
1. **Onboarding**: New developers quickly understand codebases
2. **Code Reviews**: Reviewers get context about changes
3. **Architecture Decisions**: Document and track design choices
4. **API Documentation**: Auto-generate from code
5. **Knowledge Preservation**: Capture institutional knowledge

## Product Principles
1. **Documentation as Code**: Treat docs as first-class citizens
2. **Intelligent Summarization**: Quality over quantity
3. **Semantic Understanding**: Beyond syntax to meaning
4. **Continuous Evolution**: Learn and improve from usage
5. **Developer-First**: Integrate seamlessly into workflows
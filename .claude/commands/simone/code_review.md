# Code Review - Execute top to bottom

Use the following instructions from top to bottom to execute a Code Review.

USE `code-review-expert` AGENT FOR THIS TASK
USE `zen:codereview` for this task

## Create a TODO with EXACTLY these 7 Items

1. Analyze the Scope given
2. Find code changes within Scope
3. Run automated quality checks (linting/type-checking)
4. Find relevant Specification and Documentation
5. Compare code changes against Documentation and Requirements
6. Analyze possible differences
7. Provide PASS/FAIL verdict with details

Follow step by step and adhere closely to the following instructions for each step.

## DETAILS on every TODO item

### 1. Analyze the Scope given

check: <$ARGUMENTS>

If empty, use default, otherwise interpret <$ARGUMENTS> to identify the scope of the Review. Only continue if you can find meaningful changes to review.

**CONTEXT:** Before reviewing code changes:

- Read `.simone/00_PROJECT_MANIFEST.md` to understand current sprint and milestone context
- Use the manifest to identify which sprint is active and what work is in scope
- Only evaluate against requirements appropriate for the current sprint's deliverables

### 2. Find code changes within Scope

With the identified Scope use `git diff` (on default: `git diff HEAD~1`) to find code changes.

### 3. Run automated quality checks (linting/type-checking)

**Detect and run project's quality tools:**

1. **Go projects:**
   - If `go.mod` exists: Run the following quality checks:
     - `gofmt -s -d .` - Check formatting (non-zero exit if unformatted)
     - `go vet ./...` - Run Go's static analyzer
     - `go test -race ./...` - Run tests with race detector
   - Review against `go_best_practices.md` standards

2. **JavaScript/TypeScript projects:**
   - If `package.json` exists: Check "scripts" for "lint", "type-check", "format"
   - If `.eslintrc*` exists: Run appropriate eslint command
   - If `tsconfig.json` exists: Check for type-check script or run `tsc --noEmit`
   - If `.prettierrc*` exists: Check formatting

3. **HTML/CSS projects:**
   - If HTML files changed: Check for valid HTML structure
   - If CSS files changed: Check for CSS validation and naming conventions
   - If using a CSS preprocessor (SASS/LESS): Run appropriate linter
   - Check for accessibility best practices in HTML

**Execute detected tools:**

- RUN each detected tool
- For auto-fixable issues: Apply fixes if safe (formatting only)
- For non-fixable issues: Note them in the findings

**If no linting tools found:** Skip this step (not all projects use linters)

**Critical Issues (that should influence FAIL verdict):**

- Go compilation errors or race conditions
- Type errors that would cause runtime failures
- Syntax errors in any language
- Critical security issues (SQL injection, XSS vulnerabilities)
- Violations of Go best practices (error handling, goroutine leaks)

### 4. Find relevant Specifications and Documentation

- FIND the Task, Sprint and Milestone involved in the work that was done and output your findings
- Navigate to `.simone/03_SPRINTS/` to find the current sprint directory
- READ the sprint meta file to understand sprint objectives and deliverables
- If a specific task is in scope, find and READ the task file in the sprint directory
- IDENTIFY related requirements in `.simone/02_REQUIREMENTS/` for the current milestone
- READ involved Documents especially in `.simone/01_PROJECT_DOCS/` and `.simone/02_REQUIREMENTS/`
- **IMPORTANT:** Focus on current sprint deliverables, not future milestone features

### 5. Compare code changes against Documentation and Requirements

- Use DEEP THINKING to compare changes against found Requirements and Specs.
- Compare especially these things:
  - **Data models / schemas** — fields, types, constraints, relationships.
  - **APIs / interfaces** — endpoints, params, return shapes, status codes, errors.
  - **Config / environment** — keys, defaults, required/optional.
  - **Behaviour** — business rules, side-effects, error handling.
  - **Quality** — naming, formatting, tests, linter status.
- Use `zen:precommit` to review the changes

**IMPORTANT**:

- Deviations from the Specs is not allowed. Not even small ones. Be very picky here!
- If in doubt call a **FAIL** and ask the User.
- Zero tolerance on not following the Specs and Documentation.

### 6. Analyze the differences

- Analyze any difference found
- Give every issue a Severity Score
- Severity ranges from 1 (low) to 10 (high)
- Remember List of issues and Scores for output

### 7. Provide PASS/FAIL verdict with details

- Call a **FAIL** on any differences found.
  - Zero Tolerance - even on well meant additions.
  - Leave it on the user to decide if small changes are allowed.
- Only **PASS** if no discrepancy appeared.

#### IMPORTANT: Output Format

- Output the results of your review to the task's **## Output Log** section in the task file
- Find the task file in `.simone/03_SPRINTS/` or `.simone/04_GENERAL_TASKS/` based on the scope
- Append the review results to the existing Output Log with timestamp
- Output Format:

  ```
  [YYYY-MM-DD HH:MM]: Code Review - PASS/FAIL
  Result: **FAIL/PASS** Your final decision on if it's a PASS or a FAIL.
  **Scope:** Inform the user about the review scope.
  **Findings:** Detailed list with all Issues found and Severity Score.
  **Summary:** Short summary on what is wrong or not.
  **Recommendation:** Your personal recommendation on further steps.
  ```

- Also output a brief result summary to the console for immediate feedback

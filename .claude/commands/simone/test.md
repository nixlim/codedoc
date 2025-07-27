# Run tests and fix common issues

Follow these instructions from top to bottom.

## Create a TODO with EXACTLY these 4 items

1. Execute test suite
2. Analyze results and identify issues
3. Fix common problems if found
4. Provide test summary

---

## 1 · Execute test suite

### First, detect the project's test runner

1. **Go projects:**
   - If `go.mod` exists: Try `go test ./...` (tests all packages)
   - For verbose output: `go test -v ./...`
   - For race detection: `go test -race ./...`
   - For coverage: `go test -cover ./...`
   - If custom test script exists (e.g., `test.sh`, `Makefile` with test target): Use that

2. **JavaScript/TypeScript projects:**
   - If `package.json` exists: Check "scripts" section for "test" command
   - Common: `npm test`, `npm run test`, `yarn test`, `pnpm test`
   - Framework specific: 
     - Unit testing: `jest`, `vitest`, `mocha`, `jasmine`
     - E2E testing: `cypress`, `playwright`, `puppeteer`
   - Check test configuration files:
     - `jest.config.js`, `vitest.config.js`, `.mocharc.json`
     - `cypress.config.js`, `playwright.config.js`

3. **Frontend (HTML/CSS) testing:**
   - Visual regression tests: `npm run test:visual` (if configured)
   - E2E tests: `npm run test:e2e` or `npm run cypress`
   - Linting: `npm run lint:css` or `stylelint`

### Execute the detected test command

- RUN the appropriate test command
- CAPTURE full output including any errors
- NOTE execution time and test counts

**If no test runner is found:** Report this to the user and ask for the correct test command.

## 2 · Analyze results and identify issues

Check for common issues in this order:

### Language-specific issues

**Go:**

- Test files not ending with `_test.go`
- Missing test packages
- Table-driven test syntax errors
- Race conditions (detectable with -race flag)
- Import cycle errors
- Build tag issues

**JavaScript/TypeScript:**

- Module resolution errors
- Missing dependencies in node_modules
- Jest/Vitest/Mocha configuration issues
- TypeScript compilation errors
- Test file pattern mismatches (*.test.js, *.spec.js)
- Missing test setup files

**Python:**

- Missing **init**.py files (import errors, tests not discovered)
- Import path problems
- Fixture issues (pytest)
- Virtual environment problems

**Common across languages:**

- Environment variable issues (missing config)
- Database/external service connection errors
- File path problems (absolute vs relative)
- Permission issues

## 3 · Fix common problems if found

**ONLY** fix these specific issues automatically:**

### Go-specific fixes

- RENAME test files to end with `_test.go` if needed
- FIX package declaration in test files
- ENSURE test functions start with `Test` prefix
- ADD missing import statements for testing package

### JavaScript/TypeScript fixes

- RUN `npm install` if node_modules missing
- FIX simple module resolution in jest.config.js
- CREATE missing test setup files
- ENSURE test files match configured patterns

### Python-specific fixes

- CREATE empty `__init__.py` files where needed
- FIX simple import path issues
- ADD missing test directory to Python path if needed

### General fixes

- CREATE missing test directories
- FIX file permissions if possible
- IDENTIFY missing env vars and inform user

**DO NOT** fix:

- Actual test logic failures
- Business logic bugs
- Complex configuration issues
- Database schema problems
- External service dependencies

## 4 · Provide test summary

Create a brief summary:

```
Test Results:
- Total: X tests
- Passed: Y (Z%)
- Failed: A
- Skipped: B
- Time: C seconds

Issues Fixed:
- [List any fixes applied]

Issues Found (requires manual fix):
- [List problems that need attention]

Status: PASSING | FAILING | BLOCKED
```

**IMPORTANT:** Keep it concise. This command should be quick and focused on running tests, not detailed analysis.
EOF < /dev/null

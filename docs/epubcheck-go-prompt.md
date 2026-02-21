# Project: epubcheck-go — Native EPUB Validator in Go

## Goal

Implement a fast, native Go EPUB validator. This is an implementation of the checks defined in the `epubcheck-spec` test suite. The test suite is the source of truth — our job is to make those tests pass.

## Environment Setup (do this first!)

```bash
# 1. Install Go
curl -fsSL https://go.dev/dl/go1.23.6.linux-amd64.tar.gz -o /tmp/go.tar.gz
sudo tar -C /usr/local -xzf /tmp/go.tar.gz
export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc

# 2. Clone the shared test suite
git clone https://github.com/adammathes/epubcheck-spec.git ~/epubcheck-spec

# 3. Install reference epubcheck (needed for comparison testing)
mkdir -p ~/tools
curl -fsSL https://github.com/w3c/epubcheck/releases/download/v5.3.0/epubcheck-5.3.0.zip -o /tmp/epubcheck.zip
unzip -qo /tmp/epubcheck.zip -d ~/tools/
export EPUBCHECK_JAR="$HOME/tools/epubcheck-5.3.0/epubcheck.jar"

# 4. Build the spec's epub fixtures
cd ~/epubcheck-spec
chmod +x scripts/*.sh
./scripts/build-fixtures.sh
cd -

# 5. Verify reference epubcheck works on the fixtures
java -jar $EPUBCHECK_JAR ~/epubcheck-spec/fixtures/epub/valid/minimal-epub3.epub

# 6. Set up our project
mkdir -p ~/epubcheck-go
cd ~/epubcheck-go
go mod init github.com/adammathes/epubcheck-go
```

## Relationship to epubcheck-spec

```
~/epubcheck-spec/                ← Shared test suite (cloned from GitHub)
├── fixtures/epub/               ← Test epub files (built from source)
├── expected/                    ← What correct validation output looks like
├── checks.json                  ← Registry of all checks to implement
└── scripts/
    └── compare-implementation.sh

~/epubcheck-go/                  ← This project
├── cmd/epubcheck/main.go
├── pkg/...
└── Makefile
```

Our workflow:
1. Read checks from `~/epubcheck-spec/checks.json`
2. Run our tool on each fixture epub
3. Compare against `~/epubcheck-spec/expected/`
4. Fix until it matches
5. Run the full comparison suite periodically

## Module Structure

```
epubcheck-go/
├── cmd/
│   └── epubcheck/
│       └── main.go
├── pkg/
│   ├── epub/
│   │   ├── reader.go          # Unzip + parse epub structure
│   │   ├── reader_test.go
│   │   └── types.go
│   ├── validate/
│   │   ├── validator.go       # Orchestration: run all checks, collect messages
│   │   ├── validator_test.go
│   │   ├── ocf.go             # OCF container checks
│   │   ├── ocf_test.go
│   │   ├── opf.go             # Package document checks
│   │   ├── opf_test.go
│   │   ├── nav.go             # Navigation document checks
│   │   ├── nav_test.go
│   │   ├── content.go         # Content document checks
│   │   ├── content_test.go
│   │   ├── references.go      # Cross-reference checks
│   │   ├── references_test.go
│   │   └── media.go           # Media type checks
│   └── report/
│       ├── message.go         # Message types, severity levels
│       ├── json.go            # JSON output
│       └── text.go            # Human-readable output
├── test/
│   └── spec_test.go           # Runs all epubcheck-spec fixtures against our validator
├── Makefile
├── go.mod
└── go.sum
```

## The Spec Test Runner

This is the most important file. It reads checks.json and runs every fixture:

```go
// test/spec_test.go
//
// Reads ~/epubcheck-spec/checks.json (path from EPUBCHECK_SPEC_DIR env var)
// For each check:
//   - runs our validator on the fixture epub
//   - loads expected output from expected/
//   - compares: did we detect the right error with the right severity?
//
// go test ./test/ -v                          → shows per-check pass/fail
// go test ./test/ -run TestSpec/OCF           → just OCF checks  
// go test ./test/ -run TestSpec/OCF-001       → single check
```

Each check becomes a subtest: `TestSpec/OCF-001_mimetype-file-present`

Unimplemented checks should be `t.Skip("not yet implemented")` so test output cleanly shows progress.

## CLI Interface

```
epubcheck book.epub                    # text output to stderr
epubcheck book.epub --json out.json    # JSON output to file
epubcheck book.epub --json -           # JSON to stdout
epubcheck --version
```

Exit codes: 0 = valid, 1 = errors, 2 = fatal/could not process

## Implementation Order

Follow the phases from checks.json. Each check has a `level` field. Start with Level 1:

**Level 1 — OCF container checks:**
- mimetype file present, first, uncompressed, correct content, no extra fields
- container.xml exists, parses, has rootfile, rootfile target exists

**Level 1 — OPF/metadata checks:**
- Package document parses
- Required metadata: dc:identifier, dc:title, dc:language, dcterms:modified
- unique-identifier resolves
- Manifest items have required attributes (id, href, media-type)
- No duplicate manifest IDs
- Spine itemrefs reference valid manifest IDs
- Spine not empty

**Level 1 — Cross-reference checks:**
- Every manifest href exists in the zip
- Every OEBPS/ zip entry is in the manifest
- Navigation document exists with properties="nav"
- Nav doc has epub:type="toc"

**Level 1 — Content checks:**
- XHTML content documents are well-formed XML
- Internal links resolve
- Image references resolve

After implementing each group:
```bash
go test ./test/ -v
cd ~/epubcheck-spec && ./scripts/compare-implementation.sh ~/epubcheck-go/epubcheck
```

## Technical Decisions

- `archive/zip` for epub reading
- `encoding/xml` for XML parsing (yes, namespace handling is painful — push through it)
- No external deps without justification
- Validation checks accumulate messages into a slice, never bail early
- Every check ID from checks.json maps to a Go function

## Development Loop

```
1. Pick next check from checks.json (start with Level 1, OCF first)
2. Look at the fixture: cat ~/epubcheck-spec/fixtures/src/invalid/<name>/...
3. Look at expected: cat ~/epubcheck-spec/expected/invalid/<name>.json
4. Run reference on it to see what it reports:
   java -jar $EPUBCHECK_JAR ~/epubcheck-spec/fixtures/epub/invalid/<name>.epub
5. Verify spec_test.go shows SKIP for this check
6. Implement the check
7. Run: go test ./test/ -run TestSpec/<CHECK-ID> -v
8. Compare: does our output match expected?
9. Run full suite: go test ./test/ -v
10. Commit
```

## Makefile

```makefile
SPEC_DIR ?= $(HOME)/epubcheck-spec
EPUBCHECK_JAR ?= $(HOME)/tools/epubcheck-5.3.0/epubcheck.jar

.PHONY: build test spec-test compare bench clean

build:                       ## Build the binary
	go build -o epubcheck ./cmd/epubcheck/

test:                        ## Run unit tests
	go test ./pkg/...

spec-test:                   ## Run spec compliance tests
	EPUBCHECK_SPEC_DIR=$(SPEC_DIR) go test ./test/ -v

compare: build               ## Run full comparison via spec scripts
	cd $(SPEC_DIR) && ./scripts/compare-implementation.sh $(CURDIR)/epubcheck

bench: build                 ## Benchmark vs reference epubcheck
	@echo "=== epubcheck-go ===" && time ./epubcheck $(SPEC_DIR)/fixtures/epub/valid/minimal-epub3.epub --json /dev/null 2>/dev/null
	@echo "=== reference java ===" && time java -jar $(EPUBCHECK_JAR) $(SPEC_DIR)/fixtures/epub/valid/minimal-epub3.epub --json /dev/null 2>/dev/null

clean:
	rm -f epubcheck

help:                        ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
```

## Style

- Go 1.22+, modern idioms
- Prefer stdlib, justify any external dep
- Table-driven tests in pkg/ for unit tests
- spec_test.go for integration/compliance against the shared suite
- `go vet` and `go test ./...` clean at all times
- No frameworks, no DI containers, keep it simple
- `errors.New` / `fmt.Errorf` with `%w`

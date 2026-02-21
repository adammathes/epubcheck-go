# ⚠️ WARNING: Vibecoded Experiment

**This is an experimental project created as a vibecoded experiment. It is not production-ready and is not used anywhere yet. Use at your own risk.**

---

# epubverify

A Go-based EPUB validator that checks EPUB files for compliance with standards.

## Installation

### Building from source

```bash
git clone https://github.com/adammathes/epubverify-go.git
cd epubverify-go
go build -o epubverify .
```

The compiled binary will be created as `epubverify` in the current directory.

### Verify installation

```bash
./epubverify --version
```

## Usage

### Basic validation

```bash
./epubverify path/to/book.epub
```

This will validate the EPUB file and display the results.

## Testing

### Run all tests

```bash
go test ./pkg/...
```

### Run tests with verbose output

```bash
go test -v ./pkg/...
```

### Run specific package tests

```bash
# Test the epub package
go test ./pkg/epub

# Test the validate package
go test ./pkg/validate
```

## Project Structure

- `pkg/epub/` - EPUB file parsing and handling
- `pkg/validate/` - EPUB validation logic
- `pkg/report/` - Report generation and formatting
- `main.go` - CLI entry point

## License

See LICENSE file for details.

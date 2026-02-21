package doctor

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adammathes/epubverify/pkg/epub"
	"github.com/adammathes/epubverify/pkg/validate"
)

// createTestEPUB builds a minimal EPUB 3 in a temp file and returns its path.
// The options allow injecting specific problems for doctor to fix.
type epubOpts struct {
	mimetypeContent  string // empty = correct
	mimetypeMethod   uint16 // 0 = Store
	mimetypeFirst    bool   // true = mimetype is first entry
	version          string // "3.0" or "2.0"
	includeDCModified bool
	doctype          string // empty = HTML5, "xhtml" = XHTML 1.1 doctype
	includeScript    bool   // add <script> to content but not property
	wrongMediaType   string // if non-empty, set this as media-type for the cover image
}

func defaultOpts() epubOpts {
	return epubOpts{
		mimetypeContent:  "application/epub+zip",
		mimetypeMethod:   zip.Store,
		mimetypeFirst:    true,
		version:          "3.0",
		includeDCModified: true,
		doctype:          "",
	}
}

func createTestEPUB(t *testing.T, opts epubOpts) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.epub")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	w := zip.NewWriter(f)

	mimetypeContent := opts.mimetypeContent
	if mimetypeContent == "" {
		mimetypeContent = "application/epub+zip"
	}

	// Write mimetype
	writeMimetype := func() {
		header := &zip.FileHeader{
			Name:   "mimetype",
			Method: opts.mimetypeMethod,
		}
		mw, err := w.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		mw.Write([]byte(mimetypeContent))
	}

	writeContainer := func() {
		cw, _ := w.Create("META-INF/container.xml")
		cw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`))
	}

	writeOPF := func() {
		modified := ""
		if opts.includeDCModified {
			modified = `    <meta property="dcterms:modified">2024-01-01T00:00:00Z</meta>` + "\n"
		}

		coverItem := ""
		if opts.wrongMediaType != "" {
			coverItem = `    <item id="cover" href="cover.jpg" media-type="` + opts.wrongMediaType + `"/>` + "\n"
		}

		scriptProp := ""
		if opts.includeScript {
			// Deliberately omit "scripted" property
			scriptProp = ""
		}
		_ = scriptProp

		cw, _ := w.Create("OEBPS/content.opf")
		cw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="` + opts.version + `" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="uid">urn:uuid:12345678-1234-1234-1234-123456789012</dc:identifier>
    <dc:title>Test Book</dc:title>
    <dc:language>en</dc:language>
` + modified + `  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
` + coverItem + `  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`))
	}

	writeNav := func() {
		cw, _ := w.Create("OEBPS/nav.xhtml")
		cw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head><title>Navigation</title></head>
<body>
<nav epub:type="toc"><ol><li><a href="chapter1.xhtml">Chapter 1</a></li></ol></nav>
</body>
</html>`))
	}

	writeChapter := func() {
		doctype := "<!DOCTYPE html>"
		if opts.doctype == "xhtml" {
			doctype = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">`
		}

		scriptTag := ""
		if opts.includeScript {
			scriptTag = `<script>console.log("test");</script>` + "\n"
		}

		cw, _ := w.Create("OEBPS/chapter1.xhtml")
		cw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
` + doctype + `
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>
` + scriptTag + `<p>Hello world</p>
</body>
</html>`))
	}

	writeCover := func() {
		// Write a valid JPEG file (just the magic bytes + minimal data)
		cw, _ := w.Create("OEBPS/cover.jpg")
		cw.Write([]byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00})
	}

	if opts.mimetypeFirst {
		writeMimetype()
		writeContainer()
	} else {
		writeContainer()
		writeMimetype()
	}

	writeOPF()
	writeNav()
	writeChapter()

	if opts.wrongMediaType != "" {
		writeCover()
	}

	w.Close()
	f.Close()

	return path
}

func TestDoctorFixesMimetypeContent(t *testing.T) {
	opts := defaultOpts()
	opts.mimetypeContent = "wrong/type"
	input := createTestEPUB(t, opts)
	output := filepath.Join(t.TempDir(), "fixed.epub")

	result, err := Repair(input, output)
	if err != nil {
		t.Fatalf("Repair failed: %v", err)
	}

	if len(result.Fixes) == 0 {
		t.Fatal("Expected fixes but got none")
	}

	foundMimeFix := false
	for _, fix := range result.Fixes {
		if fix.CheckID == "OCF-003" {
			foundMimeFix = true
			break
		}
	}
	if !foundMimeFix {
		t.Error("Expected OCF-003 fix for mimetype content")
	}

	// Verify the output EPUB has correct mimetype
	zr, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	if zr.File[0].Name != "mimetype" {
		t.Errorf("mimetype is not first entry, got %s", zr.File[0].Name)
	}
	if zr.File[0].Method != zip.Store {
		t.Errorf("mimetype should be stored, got method %d", zr.File[0].Method)
	}
}

func TestDoctorFixesMissingDCModified(t *testing.T) {
	opts := defaultOpts()
	opts.includeDCModified = false
	input := createTestEPUB(t, opts)
	output := filepath.Join(t.TempDir(), "fixed.epub")

	result, err := Repair(input, output)
	if err != nil {
		t.Fatalf("Repair failed: %v", err)
	}

	foundModFix := false
	for _, fix := range result.Fixes {
		if fix.CheckID == "OPF-004" {
			foundModFix = true
			break
		}
	}
	if !foundModFix {
		t.Error("Expected OPF-004 fix for missing dcterms:modified")
	}

	// Verify the output no longer has the OPF-004 error
	for _, msg := range result.AfterReport.Messages {
		if msg.CheckID == "OPF-004" {
			t.Errorf("OPF-004 still present after fix: %s", msg.Message)
		}
	}
}

func TestDoctorFixesDoctype(t *testing.T) {
	opts := defaultOpts()
	opts.doctype = "xhtml"
	input := createTestEPUB(t, opts)
	output := filepath.Join(t.TempDir(), "fixed.epub")

	result, err := Repair(input, output)
	if err != nil {
		t.Fatalf("Repair failed: %v", err)
	}

	foundDTFix := false
	for _, fix := range result.Fixes {
		if fix.CheckID == "HTM-010" {
			foundDTFix = true
			break
		}
	}
	if !foundDTFix {
		t.Error("Expected HTM-010 fix for XHTML DOCTYPE")
	}

	// Verify the output content has HTML5 DOCTYPE
	ep, err := epub.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer ep.Close()

	data, err := ep.ReadFile("OEBPS/chapter1.xhtml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("<!DOCTYPE html>")) {
		t.Error("Expected HTML5 DOCTYPE in output")
	}
	if bytes.Contains(data, []byte("XHTML")) {
		t.Error("XHTML DOCTYPE should have been removed")
	}
}

func TestDoctorFixesMissingScriptedProperty(t *testing.T) {
	opts := defaultOpts()
	opts.includeScript = true
	input := createTestEPUB(t, opts)
	output := filepath.Join(t.TempDir(), "fixed.epub")

	result, err := Repair(input, output)
	if err != nil {
		t.Fatalf("Repair failed: %v", err)
	}

	foundScriptFix := false
	for _, fix := range result.Fixes {
		if fix.CheckID == "HTM-005" {
			foundScriptFix = true
			break
		}
	}
	if !foundScriptFix {
		t.Error("Expected HTM-005 fix for missing 'scripted' property")
	}

	// Verify the property was added in the output
	for _, msg := range result.AfterReport.Messages {
		if msg.CheckID == "HTM-005" {
			t.Errorf("HTM-005 still present after fix: %s", msg.Message)
		}
	}
}

func TestDoctorFixesMediaTypeMismatch(t *testing.T) {
	opts := defaultOpts()
	opts.wrongMediaType = "image/png" // file is actually JPEG
	input := createTestEPUB(t, opts)
	output := filepath.Join(t.TempDir(), "fixed.epub")

	result, err := Repair(input, output)
	if err != nil {
		t.Fatalf("Repair failed: %v", err)
	}

	foundMediaFix := false
	for _, fix := range result.Fixes {
		if fix.CheckID == "OPF-024" {
			foundMediaFix = true
			if !strings.Contains(fix.Description, "image/jpeg") {
				t.Errorf("Expected fix to correct to image/jpeg, got: %s", fix.Description)
			}
			break
		}
	}
	if !foundMediaFix {
		t.Error("Expected OPF-024 fix for media-type mismatch")
	}
}

func TestDoctorNoFixesOnValidEPUB(t *testing.T) {
	opts := defaultOpts()
	input := createTestEPUB(t, opts)
	output := filepath.Join(t.TempDir(), "fixed.epub")

	result, err := Repair(input, output)
	if err != nil {
		t.Fatalf("Repair failed: %v", err)
	}

	// A valid EPUB should have no fixes to apply
	if len(result.Fixes) > 0 {
		for _, fix := range result.Fixes {
			t.Logf("Unexpected fix: [%s] %s", fix.CheckID, fix.Description)
		}
		t.Errorf("Expected no fixes on valid EPUB, got %d", len(result.Fixes))
	}
}

func TestDoctorOutputPassesValidation(t *testing.T) {
	// Create an EPUB with multiple problems
	opts := defaultOpts()
	opts.mimetypeContent = "wrong"
	opts.includeDCModified = false
	opts.doctype = "xhtml"
	input := createTestEPUB(t, opts)
	output := filepath.Join(t.TempDir(), "fixed.epub")

	_, err := Repair(input, output)
	if err != nil {
		t.Fatalf("Repair failed: %v", err)
	}

	// Re-validate independently
	report, err := validate.Validate(output)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	// Should have fewer errors than before
	if report.FatalCount() > 0 {
		t.Errorf("Output has %d fatal errors", report.FatalCount())
		for _, msg := range report.Messages {
			t.Logf("  %s", msg)
		}
	}
}

func TestDoctorMimetypeNotFirst(t *testing.T) {
	opts := defaultOpts()
	opts.mimetypeFirst = false
	input := createTestEPUB(t, opts)
	output := filepath.Join(t.TempDir(), "fixed.epub")

	_, err := Repair(input, output)
	if err != nil {
		t.Fatalf("Repair failed: %v", err)
	}

	// Verify output has mimetype as first entry
	zr, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	if len(zr.File) == 0 {
		t.Fatal("Output ZIP has no files")
	}
	if zr.File[0].Name != "mimetype" {
		t.Errorf("Expected mimetype as first entry, got '%s'", zr.File[0].Name)
	}
}

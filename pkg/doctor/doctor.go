// Package doctor implements an EPUB repair mode ("doctor") that applies
// safe, mechanical fixes for common validation errors.
//
// The approach:
//  1. Open the EPUB and read all files into memory
//  2. Run the standard validator to identify problems
//  3. Apply Tier 1 fixes (safe, deterministic, content-preserving)
//  4. Write a new EPUB with all fixes applied
//  5. Re-validate the output to confirm fixes worked
//
// Tier 1 fixes (implemented):
//   - OCF-001/002/003/004/005: mimetype file issues (missing, wrong content,
//     not first, extra field, compressed) — all handled by correct ZIP writing
//   - OPF-004: missing dcterms:modified — adds current timestamp
//   - OPF-024/MED-001: media-type mismatch — corrects based on file magic bytes
//   - HTM-005/006/007: missing manifest properties — adds scripted/svg/mathml
//   - HTM-010/011: wrong DOCTYPE — replaces with <!DOCTYPE html>
package doctor

import (
	"fmt"
	"io"

	"github.com/adammathes/epubverify/pkg/epub"
	"github.com/adammathes/epubverify/pkg/report"
	"github.com/adammathes/epubverify/pkg/validate"
)

// Result holds the outcome of a doctor run.
type Result struct {
	Fixes       []Fix
	BeforeReport *report.Report
	AfterReport  *report.Report
}

// Repair opens an EPUB, applies fixes, and writes the repaired version.
// If outputPath is empty, it writes to inputPath with a ".fixed.epub" suffix.
func Repair(inputPath, outputPath string) (*Result, error) {
	if outputPath == "" {
		outputPath = inputPath + ".fixed.epub"
	}

	// Step 1: Open and validate original
	ep, err := epub.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("opening epub: %w", err)
	}

	beforeReport, err := validate.Validate(inputPath)
	if err != nil {
		ep.Close()
		return nil, fmt.Errorf("validating: %w", err)
	}

	// If already valid, nothing to do
	if beforeReport.IsValid() && beforeReport.WarningCount() == 0 {
		ep.Close()
		return &Result{
			BeforeReport: beforeReport,
			AfterReport:  beforeReport,
		}, nil
	}

	// Step 2: Read all files into memory
	files := make(map[string][]byte)
	for name, f := range ep.Files {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		files[name] = data
	}

	// Need to parse container and OPF for fix functions
	// (the ep already has these parsed from Open + validate)
	ep.ParseContainer()
	ep.ParseOPF()

	// Step 3: Apply fixes
	var allFixes []Fix

	// ZIP-level: ensure correct mimetype (also fixes OCF-001 if missing)
	allFixes = append(allFixes, fixMimetype(files)...)

	// Detect ZIP-structural issues fixed by construction (the writer always
	// writes mimetype first, stored, with no extra field).
	allFixes = append(allFixes, detectZipFixes(beforeReport)...)

	// OPF-level: add missing dcterms:modified
	allFixes = append(allFixes, fixDCTermsModified(files, ep)...)

	// OPF-level: correct media-type mismatches
	allFixes = append(allFixes, fixMediaTypes(files, ep)...)

	// OPF-level: add missing manifest properties (scripted/svg/mathml)
	allFixes = append(allFixes, fixManifestProperties(files, ep)...)

	// Content-level: fix DOCTYPE declarations
	allFixes = append(allFixes, fixDoctype(files, ep)...)

	if len(allFixes) == 0 {
		ep.Close()
		return &Result{
			BeforeReport: beforeReport,
			AfterReport:  beforeReport,
		}, nil
	}

	// Step 4: Write repaired EPUB
	// The writer handles OCF-002 (mimetype first), OCF-004 (no extra field),
	// and OCF-005 (stored not compressed) by construction.
	if err := writeEPUB(outputPath, files, ep.ZipFile); err != nil {
		ep.Close()
		return nil, fmt.Errorf("writing repaired epub: %w", err)
	}

	ep.Close()

	// Step 5: Re-validate to confirm
	afterReport, err := validate.Validate(outputPath)
	if err != nil {
		return nil, fmt.Errorf("validating repaired epub: %w", err)
	}

	return &Result{
		Fixes:        allFixes,
		BeforeReport: beforeReport,
		AfterReport:  afterReport,
	}, nil
}

// Note on OCF-002/004/005:
// These are "fixed by construction" — the writeEPUB function always writes
// mimetype as the first entry, stored (not compressed), with no extra field.
// So any EPUB that passes through doctor mode gets these fixed automatically,
// even though we don't emit explicit Fix entries for them unless the original
// had different issues.

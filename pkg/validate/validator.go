package validate

import (
	"github.com/adammathes/epubverify/pkg/epub"
	"github.com/adammathes/epubverify/pkg/report"
)

// Options configures validation behavior.
type Options struct {
	// Strict enables checks that follow the EPUB spec more closely,
	// even when the reference epubcheck tool doesn't flag them.
	// This includes OCF-005 (compressed mimetype) and RSC-002 (file not in manifest).
	Strict bool

	// Accessibility enables accessibility metadata and best-practice checks (ACC-*).
	// These are not flagged by epubcheck without --profile and are off by default.
	Accessibility bool
}

// Validate runs all validation checks on an EPUB file and returns a report.
func Validate(path string) (*report.Report, error) {
	return ValidateWithOptions(path, Options{})
}

// ValidateBytes runs all validation checks on raw EPUB bytes (e.g. from a browser upload).
func ValidateBytes(data []byte) (*report.Report, error) {
	return ValidateBytesWithOptions(data, Options{})
}

// ValidateBytesWithOptions runs validation on raw EPUB bytes with the given options.
func ValidateBytesWithOptions(data []byte, opts Options) (*report.Report, error) {
	r := report.NewReport()

	ep, err := epub.OpenFromBytes(data)
	if err != nil {
		r.Add(report.Fatal, "PKG-000", "Could not open EPUB: "+err.Error())
		return r, nil
	}
	defer ep.Close()

	// Phase 1: OCF container checks
	if fatal := checkOCF(ep, r, opts); fatal {
		return r, nil
	}

	// Phase 2: Parse and check OPF
	if fatal := checkOPF(ep, r); fatal {
		return r, nil
	}

	// Phase 3: Cross-reference checks
	checkReferences(ep, r, opts)

	// Phase 4: Navigation document checks
	checkNavigation(ep, r)

	// Phase 5: Encoding checks (before content to identify bad files)
	badEncoding := checkEncoding(ep, r)

	// Phase 6: Content document checks
	checkContentWithSkips(ep, r, badEncoding)

	// Phase 7: CSS checks
	checkCSS(ep, r)

	// Phase 8: Fixed-layout checks
	checkFXL(ep, r)

	// Phase 9: Media checks
	checkMedia(ep, r)

	// Phase 10: EPUB 2 specific checks
	checkEPUB2(ep, r)

	// Phase 11: Accessibility checks (opt-in)
	if opts.Accessibility {
		checkAccessibility(ep, r)
	}

	return r, nil
}

// ValidateWithOptions runs validation with the given options.
func ValidateWithOptions(path string, opts Options) (*report.Report, error) {
	r := report.NewReport()

	ep, err := epub.Open(path)
	if err != nil {
		r.Add(report.Fatal, "PKG-000", "Could not open EPUB: "+err.Error())
		return r, nil
	}
	defer ep.Close()

	// Phase 1: OCF container checks
	if fatal := checkOCF(ep, r, opts); fatal {
		return r, nil
	}

	// Phase 2: Parse and check OPF
	if fatal := checkOPF(ep, r); fatal {
		return r, nil
	}

	// Phase 3: Cross-reference checks
	checkReferences(ep, r, opts)

	// Phase 4: Navigation document checks
	checkNavigation(ep, r)

	// Phase 5: Encoding checks (before content to identify bad files)
	badEncoding := checkEncoding(ep, r)

	// Phase 6: Content document checks
	checkContentWithSkips(ep, r, badEncoding)

	// Phase 7: CSS checks
	checkCSS(ep, r)

	// Phase 8: Fixed-layout checks
	checkFXL(ep, r)

	// Phase 9: Media checks
	checkMedia(ep, r)

	// Phase 10: EPUB 2 specific checks
	checkEPUB2(ep, r)

	// Phase 11: Accessibility checks (opt-in, not flagged by epubcheck without --profile)
	if opts.Accessibility {
		checkAccessibility(ep, r)
	}

	return r, nil
}

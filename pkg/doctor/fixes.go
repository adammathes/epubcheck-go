package doctor

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/adammathes/epubverify/pkg/epub"
	"github.com/adammathes/epubverify/pkg/report"
)

// Fix represents a single applied fix.
type Fix struct {
	CheckID     string
	Description string
	File        string // which file was modified (empty for zip-level fixes)
}

// fixMimetype ensures the mimetype file has the correct content.
// Fixes OCF-003. OCF-002/004/005 are handled by the writer.
func fixMimetype(files map[string][]byte) []Fix {
	var fixes []Fix
	expected := []byte("application/epub+zip")

	current, exists := files["mimetype"]
	if !exists {
		files["mimetype"] = expected
		fixes = append(fixes, Fix{
			CheckID:     "OCF-001",
			Description: "Added missing mimetype file",
		})
		return fixes
	}

	if !bytes.Equal(current, expected) {
		files["mimetype"] = expected
		fixes = append(fixes, Fix{
			CheckID:     "OCF-003",
			Description: fmt.Sprintf("Fixed mimetype content from '%s' to 'application/epub+zip'", strings.TrimSpace(string(current))),
		})
	}

	return fixes
}

// fixDCTermsModified adds a dcterms:modified element if missing in EPUB 3.
// Fixes OPF-004.
func fixDCTermsModified(files map[string][]byte, ep *epub.EPUB) []Fix {
	if ep.Package == nil || ep.Package.Version < "3.0" {
		return nil
	}
	if ep.Package.Metadata.Modified != "" {
		return nil
	}

	opfData, ok := files[ep.RootfilePath]
	if !ok {
		return nil
	}

	content := string(opfData)
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	// Insert before </metadata>
	metaClose := strings.Index(content, "</metadata>")
	if metaClose == -1 {
		// Try with namespace prefix
		metaClose = findClosingTag(content, "metadata")
	}
	if metaClose == -1 {
		return nil
	}

	insertion := fmt.Sprintf("    <meta property=\"dcterms:modified\">%s</meta>\n  ", now)
	newContent := content[:metaClose] + insertion + content[metaClose:]
	files[ep.RootfilePath] = []byte(newContent)

	return []Fix{{
		CheckID:     "OPF-004",
		Description: fmt.Sprintf("Added dcterms:modified with value '%s'", now),
		File:        ep.RootfilePath,
	}}
}

// fixMediaTypes corrects manifest media-type attributes that don't match actual content.
// Fixes OPF-024 and MED-001.
func fixMediaTypes(files map[string][]byte, ep *epub.EPUB) []Fix {
	if ep.Package == nil {
		return nil
	}

	var fixes []Fix

	// Image magic bytes
	pngMagic := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	jpegMagic := []byte{0xff, 0xd8, 0xff}
	gifMagic := []byte{0x47, 0x49, 0x46, 0x38}

	for _, item := range ep.Package.Manifest {
		if item.Href == "\x00MISSING" || item.MediaType == "\x00MISSING" {
			continue
		}

		fullPath := ep.ResolveHref(item.Href)

		// Check extension-based mismatch
		ext := strings.ToLower(path.Ext(item.Href))
		expectedByExt := extensionToMediaType(ext)

		// Check magic-byte-based mismatch for images
		var detectedByMagic string
		if data, ok := files[fullPath]; ok && strings.HasPrefix(item.MediaType, "image/") && item.MediaType != "image/svg+xml" {
			if len(data) >= 8 {
				if bytes.HasPrefix(data, pngMagic) {
					detectedByMagic = "image/png"
				} else if bytes.HasPrefix(data, jpegMagic) {
					detectedByMagic = "image/jpeg"
				} else if bytes.HasPrefix(data, gifMagic) {
					detectedByMagic = "image/gif"
				}
			}
		}

		// Determine the correct type — prefer magic bytes for images, fall back to extension
		correctType := ""
		if detectedByMagic != "" && detectedByMagic != item.MediaType {
			correctType = detectedByMagic
		} else if expectedByExt != "" && expectedByExt != item.MediaType {
			// Only fix extension-based mismatches for non-images (images use magic bytes)
			if !strings.HasPrefix(item.MediaType, "image/") || !strings.HasPrefix(expectedByExt, "image/") {
				correctType = expectedByExt
			}
		}

		if correctType != "" {
			fixes = append(fixes, Fix{
				CheckID:     "OPF-024",
				Description: fmt.Sprintf("Fixed media-type for '%s' from '%s' to '%s'", item.Href, item.MediaType, correctType),
				File:        ep.RootfilePath,
			})
			// Apply fix in OPF
			opfData := files[ep.RootfilePath]
			opfStr := string(opfData)
			// Replace the specific media-type for this item's href
			// Match: href="<href>" ... media-type="<old>"  or  media-type="<old>" ... href="<href>"
			opfStr = fixManifestItemMediaType(opfStr, item.Href, item.MediaType, correctType)
			files[ep.RootfilePath] = []byte(opfStr)
		}
	}

	return fixes
}

// fixManifestProperties adds missing scripted/svg/mathml properties to manifest items.
// Fixes HTM-005, HTM-006, HTM-007.
func fixManifestProperties(files map[string][]byte, ep *epub.EPUB) []Fix {
	if ep.Package == nil || ep.Package.Version < "3.0" {
		return nil
	}

	var fixes []Fix

	for _, item := range ep.Package.Manifest {
		if item.Href == "\x00MISSING" || item.MediaType != "application/xhtml+xml" {
			continue
		}

		fullPath := ep.ResolveHref(item.Href)
		data, ok := files[fullPath]
		if !ok {
			continue
		}

		// Skip nav documents
		if hasProperty(item.Properties, "nav") {
			continue
		}

		hasScript, hasSVG, hasMathML := detectContentFeatures(data)
		var missing []string

		if hasScript && !hasProperty(item.Properties, "scripted") {
			missing = append(missing, "scripted")
		}
		if hasSVG && !hasProperty(item.Properties, "svg") {
			missing = append(missing, "svg")
		}
		if hasMathML && !hasProperty(item.Properties, "mathml") {
			missing = append(missing, "mathml")
		}

		if len(missing) == 0 {
			continue
		}

		newProps := item.Properties
		for _, m := range missing {
			if newProps == "" {
				newProps = m
			} else {
				newProps = newProps + " " + m
			}
		}

		opfData := files[ep.RootfilePath]
		opfStr := string(opfData)
		opfStr = fixManifestItemProperties(opfStr, item.ID, item.Properties, newProps)
		files[ep.RootfilePath] = []byte(opfStr)

		for _, m := range missing {
			checkID := "HTM-005"
			if m == "svg" {
				checkID = "HTM-006"
			} else if m == "mathml" {
				checkID = "HTM-007"
			}
			fixes = append(fixes, Fix{
				CheckID:     checkID,
				Description: fmt.Sprintf("Added '%s' property to manifest item '%s'", m, item.ID),
				File:        ep.RootfilePath,
			})
		}
	}

	return fixes
}

// fixDoctype replaces XHTML/DTD doctypes with HTML5 DOCTYPE in EPUB 3 content docs.
// Fixes HTM-010 and HTM-011.
func fixDoctype(files map[string][]byte, ep *epub.EPUB) []Fix {
	if ep.Package == nil || ep.Package.Version < "3.0" {
		return nil
	}

	var fixes []Fix
	doctypeRe := regexp.MustCompile(`(?i)<!DOCTYPE[^>]*>`)

	for _, item := range ep.Package.Manifest {
		if item.MediaType != "application/xhtml+xml" || item.Href == "\x00MISSING" {
			continue
		}

		fullPath := ep.ResolveHref(item.Href)
		data, ok := files[fullPath]
		if !ok {
			continue
		}

		content := string(data)
		match := doctypeRe.FindString(content)
		if match == "" {
			continue
		}

		upper := strings.ToUpper(match)
		if strings.Contains(upper, "XHTML") || strings.Contains(upper, "DTD") {
			newContent := doctypeRe.ReplaceAllString(content, "<!DOCTYPE html>")
			files[fullPath] = []byte(newContent)
			fixes = append(fixes, Fix{
				CheckID:     "HTM-010",
				Description: fmt.Sprintf("Replaced non-HTML5 DOCTYPE with <!DOCTYPE html>"),
				File:        fullPath,
			})
		}
	}

	return fixes
}

// detectZipFixes checks the before-report for OCF issues that are fixed
// by construction when the writer rewrites the ZIP (mimetype ordering,
// compression, extra field). These don't modify the in-memory files but
// the writer's output will fix them.
func detectZipFixes(r *report.Report) []Fix {
	var fixes []Fix
	for _, msg := range r.Messages {
		switch msg.CheckID {
		case "OCF-002":
			fixes = append(fixes, Fix{
				CheckID:     "OCF-002",
				Description: "Reordered mimetype as first ZIP entry",
			})
		case "OCF-004":
			fixes = append(fixes, Fix{
				CheckID:     "OCF-004",
				Description: "Removed extra field from mimetype ZIP entry",
			})
		case "OCF-005":
			fixes = append(fixes, Fix{
				CheckID:     "OCF-005",
				Description: "Changed mimetype from compressed to stored",
			})
		}
	}
	return fixes
}

// --- Helper functions ---

func extensionToMediaType(ext string) string {
	switch ext {
	case ".xhtml", ".html", ".htm":
		return "application/xhtml+xml"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".ncx":
		return "application/x-dtbncx+xml"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".otf":
		return "font/otf"
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "video/mp4"
	case ".smil":
		return "application/smil+xml"
	default:
		return ""
	}
}

func hasProperty(properties, prop string) bool {
	for _, p := range strings.Fields(properties) {
		if p == prop {
			return true
		}
	}
	return false
}

func detectContentFeatures(data []byte) (hasScript, hasSVG, hasMathML bool) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if se.Name.Local == "script" {
			hasScript = true
		}
		if se.Name.Local == "svg" || se.Name.Space == "http://www.w3.org/2000/svg" {
			hasSVG = true
		}
		if se.Name.Local == "math" || se.Name.Space == "http://www.w3.org/1998/Math/MathML" {
			hasMathML = true
		}
	}
	return
}

// fixManifestItemMediaType replaces the media-type attribute for a manifest item matching href.
func fixManifestItemMediaType(opf, href, oldType, newType string) string {
	// Strategy: find the <item> element that contains this href and replace its media-type.
	// We look for the item element containing href="<href>" and replace media-type="<old>" with media-type="<new>".
	// This is done carefully to avoid false matches.

	// Escape for regex
	escapedHref := regexp.QuoteMeta(href)
	escapedOld := regexp.QuoteMeta(oldType)

	// Pattern: <item ... href="HREF" ... media-type="OLD" ...> (attributes in any order)
	// We'll find the <item ...> that contains this href
	itemRe := regexp.MustCompile(`<item\s[^>]*href="` + escapedHref + `"[^>]*>`)
	match := itemRe.FindString(opf)
	if match == "" {
		// Try single quotes
		itemRe = regexp.MustCompile(`<item\s[^>]*href='` + escapedHref + `'[^>]*>`)
		match = itemRe.FindString(opf)
	}
	if match == "" {
		return opf
	}

	// Replace media-type within this specific match
	oldAttr := regexp.MustCompile(`media-type=["']` + escapedOld + `["']`)
	newMatch := oldAttr.ReplaceAllString(match, `media-type="`+newType+`"`)
	return strings.Replace(opf, match, newMatch, 1)
}

// fixManifestItemProperties updates the properties attribute for a manifest item by ID.
func fixManifestItemProperties(opf, itemID, oldProps, newProps string) string {
	escapedID := regexp.QuoteMeta(itemID)

	// Find the <item> element with this ID
	itemRe := regexp.MustCompile(`<item\s[^>]*id="` + escapedID + `"[^>]*/?>`)
	match := itemRe.FindString(opf)
	if match == "" {
		itemRe = regexp.MustCompile(`<item\s[^>]*id='` + escapedID + `'[^>]*/?>`)
		match = itemRe.FindString(opf)
	}
	if match == "" {
		return opf
	}

	var newMatch string
	if oldProps == "" {
		// No existing properties attribute — add one before the closing /> or >
		if strings.HasSuffix(match, "/>") {
			newMatch = match[:len(match)-2] + ` properties="` + newProps + `"/>`
		} else {
			newMatch = match[:len(match)-1] + ` properties="` + newProps + `">`
		}
	} else {
		// Replace existing properties value
		escapedOld := regexp.QuoteMeta(oldProps)
		propRe := regexp.MustCompile(`properties=["']` + escapedOld + `["']`)
		newMatch = propRe.ReplaceAllString(match, `properties="`+newProps+`"`)
	}

	return strings.Replace(opf, match, newMatch, 1)
}

func findClosingTag(content, tagName string) int {
	// Try variants: </tagName>, </ns:tagName>, </dc:tagName>
	idx := strings.Index(content, "</"+tagName+">")
	if idx != -1 {
		return idx
	}
	// Try with any namespace prefix
	re := regexp.MustCompile(`</\w+:` + regexp.QuoteMeta(tagName) + `>`)
	loc := re.FindStringIndex(content)
	if loc != nil {
		return loc[0]
	}
	return -1
}

// navDocHasToc checks whether a navigation document has epub:type="toc".
// Used by doctor mode to scan content features.
func navDocHasToc(data []byte) bool {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok {
			if se.Name.Local == "nav" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "type" {
						for _, t := range strings.Fields(attr.Value) {
							if t == "toc" {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}

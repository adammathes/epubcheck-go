# EPUB Doctor Mode — Proposal & Feasibility Report

## Summary

**Verdict: This is a promising direction.** The prototype implements 8 Tier 1 fixes and successfully reduces a 6-error EPUB to 0 errors in testing. The architecture works well and is extensible.

## How It Works

```
epubverify book.epub --doctor [-o output.epub]
```

1. Opens the EPUB and validates it with the standard checker
2. Reads all files into memory
3. Applies safe, mechanical fixes for known issues
4. Writes a new EPUB (the writer itself fixes ZIP-structural issues by construction)
5. Re-validates the output to confirm fixes worked
6. Reports what changed: before/after error counts and each fix applied

Output goes to `<input>.fixed.epub` by default, or a custom path with `-o`.

## Tier 1 Fixes (Implemented)

These are **safe, deterministic, content-preserving** fixes where the correct action is unambiguous:

| Check ID | Problem | Fix | Risk |
|----------|---------|-----|------|
| OCF-001 | Missing mimetype file | Add with correct content | None |
| OCF-002 | mimetype not first ZIP entry | Writer always puts it first | None |
| OCF-003 | Wrong mimetype content | Write `application/epub+zip` | None |
| OCF-004 | Extra field in mimetype header | Writer omits extra field | None |
| OCF-005 | mimetype compressed | Writer uses Store method | None |
| OPF-004 | Missing `dcterms:modified` | Add `<meta>` with current UTC time | Low — timestamp is synthetic |
| OPF-024 / MED-001 | Media-type mismatch | Correct based on file magic bytes | Low — magic bytes are reliable |
| HTM-005/006/007 | Missing manifest properties | Add `scripted`/`svg`/`mathml` | None — detected from content |
| HTM-010/011 | Non-HTML5 DOCTYPE | Replace with `<!DOCTYPE html>` | Low — EPUB 3 requires HTML5 |

## Integration Test Results

A test EPUB with 6 simultaneous errors:

```
Before: 6 errors, 0 warnings
  ERROR(OCF-002): The mimetype file must be the first entry in the zip archive
  ERROR(OCF-003): The mimetype file must contain exactly 'application/epub+zip'
  ERROR(OPF-004): Package metadata is missing required element dcterms:modified
  ERROR(HTM-010): Irregular DOCTYPE: EPUB 3 content must use <!DOCTYPE html>
  ERROR(HTM-005): Property 'scripted' should be declared in the manifest
  ERROR(MED-001): The file 'cover.jpg' does not appear to match media type 'image/png'

Applied 6 fixes:
  [OCF-003] Fixed mimetype content
  [OCF-002] Reordered mimetype as first ZIP entry
  [OPF-004] Added dcterms:modified
  [OPF-024] Fixed media-type for 'cover.jpg' from 'image/png' to 'image/jpeg'
  [HTM-005] Added 'scripted' property to manifest item 'ch1'
  [HTM-010] Replaced non-HTML5 DOCTYPE with <!DOCTYPE html>

After: 0 errors, 0 warnings
```

## Architecture

```
pkg/doctor/
  doctor.go       — orchestrator: validate → fix → write → re-validate
  fixes.go        — individual fix functions + helpers
  writer.go       — EPUB ZIP writer (correct mimetype handling by construction)
  doctor_test.go  — unit tests (8 tests, one per fix type + valid/round-trip)
  integration_test.go — multi-problem integration test
```

Key design decisions:
- **Non-destructive**: Always writes to a new file, never modifies the original
- **Verify after**: Re-validates the output so you can confirm improvements
- **Fix by construction**: ZIP-structural issues (OCF-002/004/005) are handled by the writer always producing correct structure, rather than trying to patch the ZIP
- **OPF manipulation uses regex**: For the prototype, OPF edits use targeted regex matching on `<item>` elements. This works well for attribute changes but a future version might benefit from a proper XML round-trip (Go's `encoding/xml` doesn't preserve formatting/comments well)

## Potential Tier 2 Fixes (Future)

These are feasible but require more care:

| Check ID | Problem | Fix Approach | Complexity |
|----------|---------|-------------|------------|
| RSC-002 | File in container not in manifest | Add `<item>` to manifest with guessed media-type | Medium — need to generate unique IDs |
| HTM-003 | Empty `href=""` on `<a>` | Remove the href attribute | Low |
| HTM-004 | Obsolete elements (`<center>`, `<font>`, etc.) | Replace with styled `<div>`/`<span>` | Medium — style preservation |
| OPF-039 | `<guide>` in EPUB 3 | Remove `<guide>` element | Low |
| OPF-036 | Bad `dc:date` format | Parse and reformat | Medium |
| CSS-005 | `@import` rules | Inline the imported CSS | High |
| ENC-001 | Non-UTF-8 encoding declared | Transcode to UTF-8 | High |

## What Won't Work in Doctor Mode

Some issues are fundamentally unfixable automatically:

- **HTM-001/OPF-011**: Malformed XML — requires understanding author intent
- **RSC-001**: Missing referenced files — the content doesn't exist
- **OPF-009**: Spine references nonexistent manifest items — structural confusion
- **RSC-003**: Broken fragment identifiers — could be typos, need human judgment
- **HTM-017**: HTML entities in XHTML — need to know correct character
- **OPF-022**: Circular fallback chains — ambiguous how to break the cycle

## Recommendation

Ship this as an experimental `--doctor` flag. The architecture is clean, the fixes are safe, and the test coverage is good. The Tier 1 fixes alone handle the most common "my EPUB won't pass validation" problems that have mechanical solutions.

The main risk is the regex-based OPF editing. It works for the current fix set but would need a more robust approach for Tier 2 fixes that do structural XML modifications. Consider:
1. A proper XML serializer that preserves formatting
2. Or, byte-level splicing using the parsed positions from `encoding/xml`

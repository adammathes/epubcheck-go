# Real-World Testing Strategy

This document describes epubverify's strategy for testing against real-world
EPUBs and comparing results with the reference [epubcheck](https://github.com/w3c/epubcheck)
implementation.

## Goals

1. **Catch false positives** — epubverify should not flag valid EPUBs that
   epubcheck accepts.
2. **Catch false negatives** — epubverify should flag issues that epubcheck
   reports (where our checks overlap).
3. **Repeatable process** — anyone can reproduce the comparison from scratch.
4. **Grow the corpus over time** without breaking the existing tests.

## Quick Start

```bash
# 1. Build epubverify
make build

# 2. Download sample EPUBs (49 from Gutenberg, Feedbooks, IDPF, DAISY, etc.)
./test/realworld/download-samples.sh

# 3. Run the Go integration tests
make realworld-test

# 4. (Optional) Compare side-by-side with epubcheck (requires Java + epubcheck JAR)
EPUBCHECK_JAR=/path/to/epubcheck.jar make realworld-compare
```

## Sample Corpus

The corpus consists of 49 EPUBs from five sources: Project Gutenberg,
Feedbooks, IDPF epub3-samples, DAISY accessibility tests, and
bmaupin/epub-samples. 43 are valid, 6 are known-invalid (both tools agree).

### Valid Samples — Project Gutenberg (24)

| File | Title | Why included |
|------|-------|--------------|
| `pg11-alice.epub` | Alice in Wonderland | Small, simple structure |
| `pg84-frankenstein.epub` | Frankenstein | Multiple authors in metadata |
| `pg1342-pride-and-prejudice.epub` | Pride and Prejudice | Large (24 MB), heavy CSS, `epub:type="normal"` |
| `pg1661-sherlock.epub` | Sherlock Holmes | Multiple chapters |
| `pg2701-moby-dick.epub` | Moby Dick | Complex TOC |
| `pg74-twain-tom-sawyer.epub` | Tom Sawyer | Standard structure |
| `pg98-dickens-two-cities.epub` | A Tale of Two Cities | Standard structure |
| `pg345-dracula.epub` | Dracula | Standard structure |
| `pg1080-dante-inferno.epub` | Dante's Inferno | Translated work |
| `pg4300-joyce-ulysses.epub` | Ulysses | Large, complex |
| `pg2600-war-and-peace.epub` | War and Peace | Multiple `dc:contributor` elements |
| `pg1041-shakespeare-sonnets.epub` | Shakespeare's Sonnets | Poetry |
| `pg1524-hamlet.epub` | Hamlet | Drama |
| `pg996-don-quixote-es.epub` | Don Quixote (Spanish) | Non-English, large (44 MB) |
| `pg2000-don-quixote-es.epub` | Don Quixote (English) | Translation |
| `pg17989-les-miserables-fr.epub` | Les Miserables | French |
| `pg7000-grimm-de.epub` | Grimm's Fairy Tales | German, `dc:contributor` with ID |
| `pg25328-tao-te-ching-zh.epub` | Tao Te Ching | Chinese text |
| `pg1982-siddhartha-jp.epub` | Siddhartha | Multilingual |
| `pg5200-kafka-metamorphosis.epub` | Metamorphosis | Translator as `dc:contributor` |
| `pg46-christmas-carol-epub2.epub` | A Christmas Carol | **EPUB 2**, nested `navPoint` elements |
| `pg174-dorian-gray-epub2.epub` | Picture of Dorian Gray | **EPUB 2** |
| `pg76-twain-huck-finn-epub2.epub` | Huckleberry Finn | **EPUB 2** |
| `pg1232-prince-epub2.epub` | The Prince | **EPUB 2** |

### Valid Samples — IDPF epub3-samples (15)

From the [IDPF epub3-samples](https://github.com/IDPF/epub3-samples)
GitHub releases. These exercise exotic EPUB 3 features not found in
standard novels.

| File | Source | Features |
|------|--------|----------|
| `idpf-haruko-fxl.epub` | haruko-html-jpeg | **Fixed-layout** manga, per-spine rendition overrides |
| `idpf-cole-voyage-fxl.epub` | cole-voyage-of-life | **Fixed-layout** art gallery |
| `idpf-page-blanche-fxl.epub` | page-blanche | **Fixed-layout** with SVG |
| `idpf-svg-in-spine.epub` | svg-in-spine | **SVG content documents** in spine |
| `idpf-linear-algebra-mathml.epub` | linear-algebra | **MathML** equations |
| `idpf-moby-dick-mo.epub` | moby-dick-mo | **Media overlays** (audio sync), multiple font types |
| `idpf-wasteland-woff.epub` | wasteland-woff | **WOFF web fonts**, CSS @import |
| `idpf-wasteland-otf-obf.epub` | wasteland-otf-obf | **Obfuscated OTF fonts** |
| `idpf-arabic-rtl.epub` | regime-anticancer-arabic | **Arabic RTL** text, `alternate-script` metadata |
| `idpf-georgia-pls-ssml.epub` | georgia-pls-ssml | **SSML pronunciation**, PLS lexicons |
| `idpf-childrens-lit.epub` | childrens-literature | Title refinement metadata (`title-type`, `display-seq`) |
| `idpf-figure-gallery.epub` | figure-gallery-bindings | EPUB **bindings**, custom media types |
| `idpf-indexing.epub` | indexing-for-eds-and-auths-3f | Book indexing, TTF fonts, page templates |
| `idpf-israelsailing.epub` | israelsailing | **Hebrew RTL** content |
| `idpf-hefty-water.epub` | hefty-water | **Ultra-minimal** EPUB (4 KB) |

### Valid Samples — DAISY Accessibility Tests (2)

From [DAISY epub-accessibility-tests](https://github.com/daisy/epub-accessibility-tests)
GitHub releases. Rich accessibility metadata.

| File | Features |
|------|----------|
| `daisy-basic-functionality.epub` | Accessibility metadata, WCAG conformance |
| `daisy-non-visual-reading.epub` | Screen reader testing, alt text |

### Valid Samples — bmaupin/epub-samples (2)

From [bmaupin/epub-samples](https://github.com/bmaupin/epub-samples)
GitHub releases. Minimal EPUBs for edge-case testing.

| File | Features |
|------|----------|
| `bm-minimal-v3.epub` | **Minimal valid EPUB 3** (2 KB) |
| `bm-basic-v3plus2.epub` | **EPUB 3+2 hybrid** |

### Known-Invalid Samples (6 — both tools report errors)

| File | Title | Errors |
|------|-------|--------|
| `fb-sherlock-study.epub` | A Study in Scarlet (Feedbooks) | Mimetype trailing CRLF, NCX UID mismatch |
| `fb-art-of-war.epub` | Art of War (Feedbooks) | Mimetype trailing CRLF, NCX UID mismatch, bad date |
| `fb-odyssey.epub` | The Odyssey (Feedbooks) | Mimetype trailing CRLF, NCX UID mismatch |
| `fb-republic.epub` | The Republic (Feedbooks) | Mimetype trailing CRLF, NCX UID mismatch |
| `fb-jane-eyre.epub` | Jane Eyre (Feedbooks) | Mimetype trailing CRLF, NCX UID mismatch |
| `fb-heart-darkness.epub` | Heart of Darkness (Feedbooks) | Mimetype trailing CRLF, NCX UID mismatch |

All samples are public domain and freely available. The download script
(`download-samples.sh`) is polite: it fetches a fixed set of URLs with a
1-second delay between requests.

Sample `.epub` files are git-ignored — they must be downloaded/built locally.

## Test Layers

### 1. Go Integration Test (`test/realworld/realworld_test.go`)

Two test functions:

- **`TestRealWorldSamples`** — validates all samples; valid samples must have
  0 errors; known-invalid samples must have errors.
- **`TestKnownInvalidExpectedErrors`** — verifies known-invalid samples
  produce specific expected check IDs (OCF-003, E2-010).

Run with:
```bash
go test ./test/realworld/ -v
```

Skips gracefully if no samples are downloaded.

### 2. Comparison Script (`test/realworld/compare.sh`)

Runs both epubverify and epubcheck against all samples and produces a
side-by-side table:

```
SAMPLE                                   | EPUBVERIFY   | EPUBCHECK    | MATCH
-----------------------------------------+--------------+--------------+------
fb-art-of-war                            | INVALID E:2 W:6 | INVALID E:3 W:0 | YES
pg11-alice                               | VALID E:0 W:0 | VALID E:0 W:0 | YES
...
```

Exits with code 0 if all validity verdicts match, code 1 if any differ.
JSON results are saved to `test/realworld/results/` for manual inspection.

### 3. Makefile Targets

| Target | Description |
|--------|-------------|
| `make realworld-test` | Run Go integration tests against samples |
| `make realworld-compare` | Run full epubverify vs epubcheck comparison |

## Adding More Samples

To expand the corpus:

1. **Add the URL to `download-samples.sh`** in the `SAMPLES` array:
   ```bash
   SAMPLES=(
     ...existing entries...
     "newfile.epub|https://example.com/book.epub|Description"
   )
   ```

2. **Run the download script**: `./test/realworld/download-samples.sh`

3. **Run the tests**: `make realworld-test`

4. **If tests fail**, the failures indicate bugs to investigate and fix.

5. **If the sample is genuinely invalid** (epubcheck also reports errors),
   add it to the `knownInvalid` map in `realworld_test.go`.

### Good Sources for Samples

- **[Project Gutenberg](https://www.gutenberg.org/)** — Public domain,
  EPUB 3 with images. Append `.epub3.images` to the ebook URL.
  For EPUB 2, use `.epub.noimages`.
- **[Feedbooks](https://www.feedbooks.com/)** — Public domain, EPUB 2.
  URL pattern: `https://www.feedbooks.com/book/{id}.epub`.
- **[IDPF epub3-samples](https://github.com/IDPF/epub3-samples)** —
  Official EPUB 3 sample documents with exotic features (FXL, SVG,
  MathML, media overlays, SSML). Pre-built EPUBs available as
  [GitHub releases](https://github.com/IDPF/epub3-samples/releases).
- **[DAISY accessibility tests](https://github.com/daisy/epub-accessibility-tests)** —
  EPUBs with rich accessibility metadata. Available as GitHub releases.
- **[bmaupin/epub-samples](https://github.com/bmaupin/epub-samples)** —
  Minimal EPUBs validated with epubcheck. Available as GitHub releases.
- **[Standard Ebooks](https://standardebooks.org/)** — High-quality
  EPUB 3. Note: programmatic downloads are currently blocked.
- **[Open Textbook Library](https://open.umn.edu/opentextbooks/)** —
  CC-licensed textbooks with complex structure.

### Guidelines

- Only use freely available, legally distributable EPUBs.
- Don't bulk-download or scrape sites. Add specific URLs one at a time.
- Aim for diversity: different publishers, structures, EPUB versions,
  content types (novels, textbooks, poetry, comics), languages.
- Prefer samples that exercise different validation paths (CSS, images,
  audio, fixed layout, navigation, metadata).

## Bugs Found and Fixed

### Round 1 (5 Gutenberg EPUBs)

All 5 passed epubcheck with 0 errors. epubverify reported false positives
on all 5. Four bugs identified and fixed:

| Check ID | Severity | Description | Fix |
|----------|----------|-------------|-----|
| OPF-037 | ERROR | `refines` target IDs on `dc:creator` not tracked | Added `ID` field to `DCCreator`; parser captures `id` attr; validator includes creator IDs |
| CSS-002 | WARNING | CSS selectors like `a:link` matched as properties | Rewrote to only match inside rule blocks |
| HTM-015 | WARNING | Unknown `epub:type` values flagged as warnings | Downgraded to INFO — vocabulary is extensible per spec |
| NAV-010 | WARNING | Unknown landmark `epub:type` values flagged | Downgraded to INFO — same rationale |

### Round 2 (expanded to 25 EPUBs: +16 Gutenberg, +4 Feedbooks)

New samples exposed 4 more false positives. 6 of 25 failed (epubverify
said INVALID, epubcheck said VALID). Three bugs identified and fixed:

| Check ID | Severity | Description | Fix |
|----------|----------|-------------|-----|
| OPF-037 | ERROR | `dc:contributor` element IDs not tracked as refines targets | Added `Contributors` field to `Metadata`; parser captures contributors; validator includes their IDs |
| E2-007 | ERROR | Nested `navPoint` elements in NCX incorrectly flagged | Rewrote with stack-based tracking for proper nesting |
| OPF-036 | WARNING | Fractional seconds in ISO 8601 dates rejected | Updated W3CDTF regex to allow `.\d+` fractional seconds |

Additionally, the Feedbooks EPUBs revealed a false positive for RSC-002
(flagging ZIP directory entries as unmanifested files):

| Check ID | Severity | Description | Fix |
|----------|----------|-------------|-----|
| RSC-002 | WARNING | ZIP directory entries (trailing `/`) flagged as unmanifested | Skip entries ending with `/` |

After all fixes: **25/25 samples match epubcheck's validity verdict.**

### Round 3 (expanded to 30 EPUBs: +3 Gutenberg EPUB 2, +2 Feedbooks)

No new bugs found. All 30 samples match epubcheck's validity verdict.

### Round 4 (expanded to 42 EPUBs: +12 IDPF epub3-samples)

The IDPF samples exercise exotic EPUB 3 features: fixed-layout, SVG in
spine, MathML, media overlays, SSML pronunciation, RTL text, web fonts,
bindings, and custom media types. These exposed 7 new false positives:

| Check ID | Severity | Description | Fix |
|----------|----------|-------------|-----|
| OPF-037 | ERROR | `dc:title` element IDs not tracked as refines targets | Changed `Titles` from `[]string` to `[]DCTitle` with `ID` field |
| CSS-001 | ERROR | CSS comments with special characters falsely parsed as syntax errors | Strip comments before analyzing CSS syntax |
| OPF-024 | ERROR | Font MIME types `application/vnd.ms-opentype` and `text/javascript` rejected | Added `mediaTypesEquivalent()` for font/JS/MP4 type aliases |
| HTM-013 | ERROR | FXL viewport check ignores per-spine-item `rendition:layout-reflowable` overrides | Check spine itemref properties for rendition overrides |
| HTM-020 | WARNING | Processing instructions flagged as warnings | Downgraded to INFO — PIs are allowed per EPUB spec |
| HTM-031 | ERROR | SSML namespace flagged as forbidden | Disabled check — SSML attributes are explicitly permitted in EPUB 3 |
| MED-004 | ERROR | Non-spine foreign resources (page templates, custom XML) flagged for missing fallback | Only require fallback for spine items |

After all fixes: **42/42 samples match epubcheck's validity verdict.**

### Round 5 (expanded to 49 EPUBs: +7 from IDPF, DAISY, bmaupin)

Added more IDPF samples (obfuscated fonts, Hebrew RTL, ultra-minimal),
DAISY accessibility test EPUBs, and minimal EPUB test files. Two
additional IDPF/ReadBeyond samples were excluded because they require
HTML5 schema validation (RSC-005) which we don't implement.

No new bugs found. **49/49 samples match epubcheck's validity verdict.**

## Future Work

- **Add Standard Ebooks samples** — currently blocked by their anti-bot
  measures. Could build from their GitHub source repos.
- **HTML5 schema validation (RSC-005)** — some EPUBs (e.g.,
  `cc-shared-culture.epub`) have HTML5 schema errors that epubcheck flags
  but we don't detect. This would require integrating an HTML5 validator.

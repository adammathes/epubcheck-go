#!/bin/bash
#
# download-samples.sh - Download public domain EPUB samples for testing
#
# Downloads a curated set of freely available EPUBs from Project Gutenberg
# and Feedbooks. These are used to compare epubverify output against the
# reference epubcheck tool.
#
# Usage: ./download-samples.sh [--force]
#   --force  Re-download even if files already exist
#
# Be polite: this script downloads a small, fixed set of files with a
# delay between requests. Do not modify it to bulk-scrape any site.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SAMPLES_DIR="$SCRIPT_DIR/samples"
FORCE="${1:-}"

mkdir -p "$SAMPLES_DIR"

# Curated list of EPUBs.
# Format: filename|URL|description
#
# Sources:
#   - Project Gutenberg (gutenberg.org): Public domain, EPUB 3 with images
#   - Feedbooks (feedbooks.com): Public domain, EPUB 2
SAMPLES=(
  # --- Project Gutenberg EPUB 3 (valid EPUBs) ---
  "pg11-alice.epub|https://www.gutenberg.org/ebooks/11.epub3.images|Alice in Wonderland (EPUB 3)"
  "pg84-frankenstein.epub|https://www.gutenberg.org/ebooks/84.epub3.images|Frankenstein (EPUB 3)"
  "pg1342-pride-and-prejudice.epub|https://www.gutenberg.org/ebooks/1342.epub3.images|Pride and Prejudice (EPUB 3, large, epub:type=normal)"
  "pg1661-sherlock.epub|https://www.gutenberg.org/ebooks/1661.epub3.images|Sherlock Holmes (EPUB 3)"
  "pg2701-moby-dick.epub|https://www.gutenberg.org/ebooks/2701.epub3.images|Moby Dick (EPUB 3, complex TOC)"
  "pg74-twain-tom-sawyer.epub|https://www.gutenberg.org/ebooks/74.epub3.images|Tom Sawyer (EPUB 3)"
  "pg98-dickens-two-cities.epub|https://www.gutenberg.org/ebooks/98.epub3.images|A Tale of Two Cities (EPUB 3)"
  "pg345-dracula.epub|https://www.gutenberg.org/ebooks/345.epub3.images|Dracula (EPUB 3)"
  "pg1080-dante-inferno.epub|https://www.gutenberg.org/ebooks/1080.epub3.images|Dante's Inferno (EPUB 3)"
  "pg4300-joyce-ulysses.epub|https://www.gutenberg.org/ebooks/4300.epub3.images|Ulysses (EPUB 3, large)"
  "pg2600-war-and-peace.epub|https://www.gutenberg.org/ebooks/2600.epub3.images|War and Peace (EPUB 3, multiple contributors)"
  # Poetry and drama
  "pg1041-shakespeare-sonnets.epub|https://www.gutenberg.org/ebooks/1041.epub3.images|Shakespeare's Sonnets (EPUB 3, poetry)"
  "pg1524-hamlet.epub|https://www.gutenberg.org/ebooks/1524.epub3.images|Hamlet (EPUB 3, drama)"
  # Non-English
  "pg996-don-quixote-es.epub|https://www.gutenberg.org/ebooks/996.epub3.images|Don Quixote Spanish original (EPUB 3, large, Spanish)"
  "pg2000-don-quixote-es.epub|https://www.gutenberg.org/ebooks/2000.epub3.images|Don Quixote English translation (EPUB 3)"
  "pg17989-les-miserables-fr.epub|https://www.gutenberg.org/ebooks/17989.epub3.images|Les Miserables (EPUB 3, French)"
  "pg7000-grimm-de.epub|https://www.gutenberg.org/ebooks/7000.epub3.images|Grimm's Fairy Tales (EPUB 3, German, contributor IDs)"
  "pg25328-tao-te-ching-zh.epub|https://www.gutenberg.org/ebooks/25328.epub3.images|Tao Te Ching (EPUB 3, Chinese)"
  "pg1982-siddhartha-jp.epub|https://www.gutenberg.org/ebooks/1982.epub3.images|Siddhartha (EPUB 3)"
  "pg5200-kafka-metamorphosis.epub|https://www.gutenberg.org/ebooks/5200.epub3.images|Metamorphosis (EPUB 3, translator as contributor)"
  # EPUB 2
  "pg46-christmas-carol-epub2.epub|https://www.gutenberg.org/ebooks/46.epub.noimages|A Christmas Carol (EPUB 2, nested navPoints)"
  "pg174-dorian-gray-epub2.epub|https://www.gutenberg.org/ebooks/174.epub.noimages|Picture of Dorian Gray (EPUB 2)"
  "pg76-twain-huck-finn-epub2.epub|https://www.gutenberg.org/ebooks/76.epub.noimages|Huckleberry Finn (EPUB 2)"
  "pg1232-prince-epub2.epub|https://www.gutenberg.org/ebooks/1232.epub.noimages|The Prince (EPUB 2)"

  # --- Feedbooks EPUB 2 (known-invalid: mimetype CRLF, NCX UID mismatch) ---
  "fb-sherlock-study.epub|https://www.feedbooks.com/book/4453.epub|A Study in Scarlet - Feedbooks (EPUB 2)"
  "fb-art-of-war.epub|https://www.feedbooks.com/book/168.epub|Art of War - Feedbooks (EPUB 2)"
  "fb-odyssey.epub|https://www.feedbooks.com/book/3676.epub|The Odyssey - Feedbooks (EPUB 2)"
  "fb-republic.epub|https://www.feedbooks.com/book/4940.epub|The Republic - Feedbooks (EPUB 2)"
  "fb-jane-eyre.epub|https://www.feedbooks.com/book/95.epub|Jane Eyre - Feedbooks (EPUB 2)"
  "fb-heart-darkness.epub|https://www.feedbooks.com/book/690.epub|Heart of Darkness - Feedbooks (EPUB 2)"
)

downloaded=0
skipped=0
failed=0

for entry in "${SAMPLES[@]}"; do
  IFS='|' read -r filename url description <<< "$entry"
  dest="$SAMPLES_DIR/$filename"

  if [[ -f "$dest" && "$FORCE" != "--force" ]]; then
    echo "SKIP  $filename (already exists)"
    skipped=$((skipped + 1))
    continue
  fi

  echo "GET   $filename - $description"
  curl -L -s -o "$dest" "$url"

  # Verify it's actually a ZIP/EPUB, not an HTML error page
  if file "$dest" | grep -q "EPUB\|Zip"; then
    echo "  OK  $(du -h "$dest" | cut -f1)"
    downloaded=$((downloaded + 1))
  else
    echo "  FAIL  Downloaded file is not a valid EPUB ($(file -b "$dest"))"
    rm -f "$dest"
    failed=$((failed + 1))
  fi

  # Be polite: 1 second between requests
  sleep 1
done

echo ""
echo "Done. Downloaded: $downloaded, Skipped: $skipped, Failed: $failed"

# --- IDPF EPUB 3 Samples (from GitHub releases) ---
# Official EPUB 3 sample documents from IDPF/W3C. These exercise exotic
# EPUB features: fixed-layout, SVG, MathML, media overlays, SSML, RTL, etc.
IDPF_BASE="https://github.com/IDPF/epub3-samples/releases/download/20230704"
IDPF_SAMPLES=(
  "idpf-haruko-fxl.epub|${IDPF_BASE}/haruko-html-jpeg.epub|Fixed-layout manga (IDPF)"
  "idpf-cole-voyage-fxl.epub|${IDPF_BASE}/cole-voyage-of-life.epub|Fixed-layout art (IDPF)"
  "idpf-page-blanche-fxl.epub|${IDPF_BASE}/page-blanche.epub|Fixed-layout SVG (IDPF)"
  "idpf-svg-in-spine.epub|${IDPF_BASE}/svg-in-spine.epub|SVG content documents (IDPF)"
  "idpf-linear-algebra-mathml.epub|${IDPF_BASE}/linear-algebra.epub|MathML equations (IDPF)"
  "idpf-moby-dick-mo.epub|${IDPF_BASE}/moby-dick-mo.epub|Media overlays (IDPF)"
  "idpf-wasteland-woff.epub|${IDPF_BASE}/wasteland-woff.epub|WOFF web fonts (IDPF)"
  "idpf-wasteland-otf-obf.epub|${IDPF_BASE}/wasteland-otf-obf.epub|Obfuscated OTF fonts (IDPF)"
  "idpf-arabic-rtl.epub|${IDPF_BASE}/regime-anticancer-arabic.epub|Arabic RTL text (IDPF)"
  "idpf-georgia-pls-ssml.epub|${IDPF_BASE}/georgia-pls-ssml.epub|SSML pronunciation (IDPF)"
  "idpf-childrens-lit.epub|${IDPF_BASE}/childrens-literature.epub|Title refinement metadata (IDPF)"
  "idpf-figure-gallery.epub|${IDPF_BASE}/figure-gallery-bindings.epub|EPUB bindings (IDPF)"
  "idpf-indexing.epub|${IDPF_BASE}/indexing-for-eds-and-auths-3f.epub|Indexing, TTF fonts (IDPF)"
  "idpf-israelsailing.epub|${IDPF_BASE}/israelsailing.epub|Hebrew RTL content (IDPF)"
  "idpf-hefty-water.epub|${IDPF_BASE}/hefty-water.epub|Ultra-minimal EPUB (IDPF)"
)

# --- DAISY Accessibility Tests ---
DAISY_BASE="https://github.com/daisy/epub-accessibility-tests/releases/download/fundamental-2.0"
DAISY_SAMPLES=(
  "daisy-basic-functionality.epub|${DAISY_BASE}/Fundamental-Accessibility-Tests-Basic-Functionality-v2.0.0.epub|Accessibility metadata (DAISY)"
  "daisy-non-visual-reading.epub|${DAISY_BASE}/Fundamental-Accessibility-Tests-Non-Visual-Reading-v2.0.0.epub|Non-visual reading tests (DAISY)"
)

# --- Minimal EPUB test files (bmaupin/epub-samples) ---
BM_BASE="https://github.com/bmaupin/epub-samples/releases/download/v0.3"
BM_SAMPLES=(
  "bm-minimal-v3.epub|${BM_BASE}/minimal-v3.epub|Minimal valid EPUB 3 (2KB)"
  "bm-basic-v3plus2.epub|${BM_BASE}/basic-v3plus2.epub|Basic EPUB 3+2 hybrid"
)

for extra_array in IDPF_SAMPLES DAISY_SAMPLES BM_SAMPLES; do
  eval 'entries=("${'$extra_array'[@]}")'
  for entry in "${entries[@]}"; do
    IFS='|' read -r filename url description <<< "$entry"
    dest="$SAMPLES_DIR/$filename"

    if [[ -f "$dest" && "$FORCE" != "--force" ]]; then
      echo "SKIP  $filename (already exists)"
      skipped=$((skipped + 1))
      continue
    fi

    echo "GET   $filename - $description"
    curl -L -s -o "$dest" "$url"

    if file "$dest" | grep -q "EPUB\|Zip"; then
      echo "  OK  $(du -h "$dest" | cut -f1)"
      downloaded=$((downloaded + 1))
    else
      echo "  FAIL  Downloaded file is not a valid EPUB ($(file -b "$dest"))"
      rm -f "$dest"
      failed=$((failed + 1))
    fi

    sleep 1
  done
done

echo ""
echo "Samples directory: $SAMPLES_DIR"
echo "Total EPUBs: $(ls "$SAMPLES_DIR"/*.epub 2>/dev/null | wc -l)"

// Package cvtext extracts plain text from an uploaded CV file. It supports plain
// text and DOCX (Office Open XML) using only the standard library; other formats
// (notably PDF) are reported as unsupported so the caller can fall back to asking
// the candidate to paste the CV text. Treat the returned text as untrusted (it is
// candidate-supplied and flows into the no-fabrication extraction pipeline).
package cvtext

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// docBodyName is the part of a DOCX archive holding the document text.
const docBodyName = "word/document.xml"

// maxDocxTextBytes bounds the *decompressed* document body so a zip bomb (a small
// DOCX whose body inflates to gigabytes) cannot exhaust memory on a single
// upload. It is generous relative to the downstream rune budget; a real CV is far
// smaller.
const maxDocxTextBytes = 4 << 20 // 4 MiB

// Extract returns the plain text of a CV file, dispatching on the filename's
// extension. An empty/unknown extension is treated as plain text. PDF and other
// binary formats return a kernel.Invalid error inviting the caller to paste text.
func Extract(filename string, data []byte) (string, error) {
	switch ext := strings.ToLower(filepath.Ext(filename)); ext {
	case "", ".txt", ".text", ".md":
		return strings.TrimSpace(string(data)), nil
	case ".docx":
		return extractDocx(data)
	case ".pdf":
		return "", kernel.Invalid("cvtext: PDF upload is not supported yet; please paste the CV text instead")
	default:
		return "", kernel.Invalidf("cvtext: unsupported file type %q; please paste the CV text instead", ext)
	}
}

// extractDocx pulls the visible text out of a DOCX archive's document body.
func extractDocx(data []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", kernel.Invalid("cvtext: file is not a valid DOCX (could not read the archive)")
	}
	var body *zip.File
	for _, f := range zr.File {
		if f.Name == docBodyName {
			body = f
			break
		}
	}
	if body == nil {
		return "", kernel.Invalid("cvtext: DOCX is missing its document body")
	}
	// Reject an oversized declared body up front, then hard-cap the decompressed
	// stream — defending against a zip bomb whether the archive's size metadata is
	// honest or forged.
	if body.UncompressedSize64 > maxDocxTextBytes {
		return "", kernel.Invalid("cvtext: DOCX document body is too large")
	}
	rc, err := body.Open()
	if err != nil {
		return "", kernel.Wrap(err, kernel.KindInvalid, "cvtext: cannot open the DOCX body")
	}
	defer func() { _ = rc.Close() }()
	return decodeDocxText(io.LimitReader(rc, maxDocxTextBytes))
}

// decodeDocxText walks the WordprocessingML stream, concatenating the text runs
// (<w:t>) and turning paragraph/break elements into whitespace.
func decodeDocxText(r io.Reader) (string, error) {
	dec := xml.NewDecoder(r)
	var b strings.Builder
	var inText bool
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", kernel.Wrap(err, kernel.KindInvalid, "cvtext: malformed DOCX XML")
		}
		inText = appendDocxToken(&b, tok, inText)
	}
	return strings.TrimSpace(b.String()), nil
}

// appendDocxToken writes one XML token's text contribution and returns whether
// the cursor is now inside a <w:t> run. Paragraph ends become newlines; tab/break
// elements become spaces.
func appendDocxToken(b *strings.Builder, tok xml.Token, inText bool) bool {
	switch t := tok.(type) {
	case xml.StartElement:
		if t.Name.Local == "t" {
			return true
		}
		if t.Name.Local == "tab" || t.Name.Local == "br" {
			b.WriteByte(' ')
		}
	case xml.EndElement:
		if t.Name.Local == "t" {
			return false
		}
		if t.Name.Local == "p" {
			b.WriteByte('\n')
		}
	case xml.CharData:
		if inText {
			b.Write(t)
		}
	}
	return inText
}

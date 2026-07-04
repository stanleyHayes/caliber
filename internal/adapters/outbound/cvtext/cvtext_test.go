package cvtext_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/cvtext"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestExtract_DocxRejectsZipBombBody(t *testing.T) {
	// A DOCX whose decompressed body exceeds the cap is rejected before it can
	// inflate in memory — a small compressed payload that would expand to many MiB.
	huge := makeDocx(t, strings.Repeat("A", 5<<20)) // ~5 MiB document.xml, > 4 MiB cap
	assert.Less(t, len(huge), 1<<20, "the compressed payload is small (highly repetitive) — a zip bomb")
	_, err := cvtext.Extract("cv.docx", huge)
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

// makeDocx builds a minimal valid DOCX (a zip with word/document.xml) whose body
// contains the given paragraphs as <w:t> runs.
func makeDocx(t *testing.T, paragraphs ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	require.NoError(t, err)
	var body bytes.Buffer
	body.WriteString(`<?xml version="1.0"?><w:document xmlns:w="x"><w:body>`)
	for _, p := range paragraphs {
		body.WriteString(`<w:p><w:r><w:t>` + p + `</w:t></w:r></w:p>`)
	}
	body.WriteString(`</w:body></w:document>`)
	_, err = w.Write(body.Bytes())
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestExtract_PlainText(t *testing.T) {
	got, err := cvtext.Extract("cv.txt", []byte("  Senior Go engineer  "))
	require.NoError(t, err)
	assert.Equal(t, "Senior Go engineer", got)
}

func TestExtract_NoExtensionTreatedAsText(t *testing.T) {
	got, err := cvtext.Extract("", []byte("raw resume text"))
	require.NoError(t, err)
	assert.Equal(t, "raw resume text", got)
}

func TestExtract_Docx(t *testing.T) {
	docx := makeDocx(t, "Led a payments platform in Go.", "Designed Postgres schemas.")
	got, err := cvtext.Extract("resume.docx", docx)
	require.NoError(t, err)
	assert.Contains(t, got, "Led a payments platform in Go.")
	assert.Contains(t, got, "Designed Postgres schemas.")
}

func TestExtract_DocxIsCaseInsensitive(t *testing.T) {
	docx := makeDocx(t, "Hello")
	got, err := cvtext.Extract("Resume.DOCX", docx)
	require.NoError(t, err)
	assert.Equal(t, "Hello", got)
}

func TestExtract_PDF(t *testing.T) {
	pdf := makePDF(t, "Led a payments platform in Go.", "Designed Postgres schemas.")
	got, err := cvtext.Extract("cv.pdf", pdf)
	require.NoError(t, err)
	assert.Contains(t, got, "Led a payments platform in Go.")
	assert.Contains(t, got, "Designed Postgres schemas.")
}

func TestExtract_PDFIsCaseInsensitive(t *testing.T) {
	pdf := makePDF(t, "Senior Go engineer")
	got, err := cvtext.Extract("Resume.PDF", pdf)
	require.NoError(t, err)
	assert.Contains(t, got, "Senior Go engineer")
}

func TestExtract_CorruptPDFRejected(t *testing.T) {
	_, err := cvtext.Extract("cv.pdf", []byte("%PDF-1.7 ..."))
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestExtract_PDFWithoutTextRejected(t *testing.T) {
	_, err := cvtext.Extract("cv.pdf", makePDF(t))
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
	assert.Contains(t, err.Error(), "no extractable text")
}

func TestExtract_UnknownTypeRejected(t *testing.T) {
	_, err := cvtext.Extract("cv.rtf", []byte("data"))
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func makePDF(t *testing.T, lines ...string) []byte {
	t.Helper()
	var stream strings.Builder
	stream.WriteString("BT\n/F1 12 Tf\n72 720 Td\n")
	for i, line := range lines {
		if i > 0 {
			stream.WriteString("0 -16 Td\n")
		}
		stream.WriteString("(")
		stream.WriteString(escapePDFText(line))
		stream.WriteString(") Tj\n")
	}
	stream.WriteString("ET\n")
	content := stream.String()
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return buf.Bytes()
}

func escapePDFText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "(", `\(`)
	return strings.ReplaceAll(s, ")", `\)`)
}

func TestExtract_CorruptDocxRejected(t *testing.T) {
	_, err := cvtext.Extract("cv.docx", []byte("not a zip"))
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestExtract_DocxMissingBodyRejected(t *testing.T) {
	// A valid zip but without word/document.xml.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("other.xml")
	require.NoError(t, err)
	_, err = w.Write([]byte("<x/>"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	_, err = cvtext.Extract("cv.docx", buf.Bytes())
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

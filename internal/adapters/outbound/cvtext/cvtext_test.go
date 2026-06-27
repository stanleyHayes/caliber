package cvtext_test

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xcreativs/caliber/internal/adapters/outbound/cvtext"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

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

func TestExtract_PDFUnsupported(t *testing.T) {
	_, err := cvtext.Extract("cv.pdf", []byte("%PDF-1.7 ..."))
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
}

func TestExtract_UnknownTypeRejected(t *testing.T) {
	_, err := cvtext.Extract("cv.rtf", []byte("data"))
	require.Error(t, err)
	assert.Equal(t, kernel.KindInvalid, kernel.KindOf(err))
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

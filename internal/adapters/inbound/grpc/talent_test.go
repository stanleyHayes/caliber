package grpcadapter

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/xcreativs/caliber/internal/adapters/outbound/llm"
	"github.com/xcreativs/caliber/internal/adapters/outbound/memory"
	profilesapp "github.com/xcreativs/caliber/internal/app/profiles"
	"github.com/xcreativs/caliber/internal/domain/kernel"
	"github.com/xcreativs/caliber/internal/domain/talent"
	caliberv1 "github.com/xcreativs/caliber/internal/gen/caliber/v1"
)

func TestTalentCreateThenGetProfile(t *testing.T) {
	ctx := context.Background()
	candidates := memory.NewCandidateRepo()
	profiles := memory.NewTalentProfileRepo()
	cand, err := talent.NewCandidate(kernel.NewID(), "", talent.CandidateIntake{})
	require.NoError(t, err)
	require.NoError(t, candidates.Create(ctx, cand))

	srv := NewTalentServer(profilesapp.NewProfileBuilder(candidates, profiles, llm.NewDev()))

	resp, err := srv.CreateProfileFromCV(ctx, &caliberv1.CreateProfileFromCVRequest{
		CandidateId: cand.ID.String(),
		CvText:      "Senior engineer experienced in Go and Postgres at scale, with gRPC services.",
		Intake:      &caliberv1.CandidateIntake{Location: "Accra"},
	})
	require.NoError(t, err)
	names := map[string]bool{}
	for _, c := range resp.GetProfile().GetCompetencies() {
		names[c.GetName()] = true
		assert.NotEmpty(t, c.GetEvidenceQuote(), "every competency cites evidence")
	}
	assert.True(t, names["Go"] && names["Postgres"], "extracted from the CV's actual content")

	got, err := srv.GetTalentProfile(ctx, &caliberv1.GetTalentProfileRequest{CandidateId: cand.ID.String()})
	require.NoError(t, err)
	assert.Len(t, got.GetProfile().GetCompetencies(), len(resp.GetProfile().GetCompetencies()))
}

func buildDocx(t *testing.T, text string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	require.NoError(t, err)
	_, err = w.Write([]byte(`<?xml version="1.0"?><w:document xmlns:w="x"><w:body><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:body></w:document>`))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestCreateProfileFromCV_FileUpload(t *testing.T) {
	ctx := context.Background()
	candidates := memory.NewCandidateRepo()
	profiles := memory.NewTalentProfileRepo()
	cand, err := talent.NewCandidate(kernel.NewID(), "", talent.CandidateIntake{})
	require.NoError(t, err)
	require.NoError(t, candidates.Create(ctx, cand))
	srv := NewTalentServer(profilesapp.NewProfileBuilder(candidates, profiles, llm.NewDev()))

	docx := buildDocx(t, "Senior engineer in Go and Postgres building gRPC services.")
	resp, err := srv.CreateProfileFromCV(ctx, &caliberv1.CreateProfileFromCVRequest{
		CandidateId: cand.ID.String(),
		CvFile:      docx,
		CvFilename:  "resume.docx",
	})
	require.NoError(t, err)
	names := map[string]bool{}
	for _, c := range resp.GetProfile().GetCompetencies() {
		names[c.GetName()] = true
	}
	assert.True(t, names["Go"] && names["Postgres"], "profile extracted from the uploaded DOCX, not cv_text")
}

func TestCreateProfileFromCV_RejectsOversizeAndUnsupported(t *testing.T) {
	srv := NewTalentServer(profilesapp.NewProfileBuilder(memory.NewCandidateRepo(), memory.NewTalentProfileRepo(), llm.NewDev()))
	cid := kernel.NewID().String()

	// Oversize upload is rejected before any parsing/extraction.
	_, err := srv.CreateProfileFromCV(context.Background(), &caliberv1.CreateProfileFromCVRequest{
		CandidateId: cid, CvFile: make([]byte, (10<<20)+1), CvFilename: "big.txt",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// An unsupported format (PDF) is rejected with guidance to paste text.
	_, err = srv.CreateProfileFromCV(context.Background(), &caliberv1.CreateProfileFromCVRequest{
		CandidateId: cid, CvFile: []byte("%PDF-1.7"), CvFilename: "cv.pdf",
	})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

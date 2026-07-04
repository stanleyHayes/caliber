package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/xcreativs/caliber/internal/adapters/outbound/fieldcrypto"
	"github.com/xcreativs/caliber/internal/adapters/outbound/postgres/sqlcdb"
	"github.com/xcreativs/caliber/internal/domain/interview"
	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// InterviewRepo is a Postgres-backed interview.InterviewRepository (CAL-066). An
// interview spans two tables — talent_interviews and interview_turns — so writes
// run in a transaction. A pending (asked-but-unanswered) question is persisted as
// a turn row with a NULL answer; answered turns carry a non-NULL answer, so the
// two are distinguishable on read even when an answer is the empty string.
//
// Interview question and answer text is candidate PII (the screening dialogue),
// so both are encrypted at rest when a field cipher is configured (CAL-117),
// mirroring the talent-profile summary.
type InterviewRepo struct {
	q      *sqlcdb.Queries
	tx     txBeginner
	cipher *fieldcrypto.FieldCipher
}

// InterviewOption customizes an InterviewRepo.
type InterviewOption func(*InterviewRepo)

// WithInterviewCipher encrypts interview question/answer text at rest with the
// given cipher (CAL-117). A nil cipher is ignored (passthrough default kept).
func WithInterviewCipher(c *fieldcrypto.FieldCipher) InterviewOption {
	return func(r *InterviewRepo) {
		if c != nil {
			r.cipher = c
		}
	}
}

// NewInterviewRepo builds the repository from a sqlc DBTX. A pool handle also
// enables the atomic multi-table writes; without one it falls back to sequential
// statements (e.g. when the handle is already an open transaction). Without a
// cipher option the Q&A is stored as plaintext (passthrough).
func NewInterviewRepo(db sqlcdb.DBTX, opts ...InterviewOption) *InterviewRepo {
	repo := &InterviewRepo{q: sqlcdb.New(db), cipher: fieldcrypto.Passthrough()}
	if b, ok := db.(txBeginner); ok {
		repo.tx = b
	}
	for _, o := range opts {
		o(repo)
	}
	return repo
}

// Create inserts a new interview and its turns atomically.
func (r *InterviewRepo) Create(ctx context.Context, iv *interview.Interview) error {
	return r.withTx(ctx, func(q *sqlcdb.Queries) error {
		params, err := r.interviewParams(iv)
		if err != nil {
			return err
		}
		if err := q.CreateInterview(ctx, params); err != nil {
			return err
		}
		return r.insertTurns(ctx, q, iv)
	})
}

// Update replaces the interview row and rewrites its turns; returns NotFound if
// the interview does not exist. Turns are cleared and re-inserted so the pending
// question and any new answered turns converge on the aggregate's current state.
func (r *InterviewRepo) Update(ctx context.Context, iv *interview.Interview) error {
	return r.withTx(ctx, func(q *sqlcdb.Queries) error {
		report, confidence, err := r.marshalReport(iv.Report)
		if err != nil {
			return err
		}
		n, err := q.UpdateInterview(ctx, sqlcdb.UpdateInterviewParams{
			ID:         iv.ID.String(),
			Mode:       iv.Mode.String(),
			Status:     iv.State.String(),
			ReportCard: report,
			Confidence: confidence,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return kernel.NotFound("postgres: interview not found")
		}
		if err := q.DeleteInterviewTurns(ctx, iv.ID.String()); err != nil {
			return err
		}
		return r.insertTurns(ctx, q, iv)
	})
}

// ByID reconstructs the full interview (turns + pending + report) or NotFound.
func (r *InterviewRepo) ByID(ctx context.Context, id kernel.ID) (*interview.Interview, error) {
	row, err := r.q.GetInterview(ctx, id.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, kernel.NotFound("postgres: interview not found")
	}
	if err != nil {
		return nil, err
	}
	return r.hydrate(ctx, row)
}

// ByCandidate returns a page of a candidate's interviews as full aggregates,
// newest first, with the total count for pagination.
func (r *InterviewRepo) ByCandidate(
	ctx context.Context, candidateID kernel.ID, page kernel.Page,
) ([]*interview.Interview, int64, error) {
	rows, err := r.q.ListInterviewsByCandidate(ctx, sqlcdb.ListInterviewsByCandidateParams{
		CandidateID: candidateID.String(),
		Limit:       clampInt32(page.Limit()),
		Offset:      clampInt32(page.Offset()),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]*interview.Interview, 0, len(rows))
	for _, row := range rows {
		iv, herr := r.hydrate(ctx, row)
		if herr != nil {
			return nil, 0, herr
		}
		out = append(out, iv)
	}
	total, err := r.q.CountInterviewsByCandidate(ctx, candidateID.String())
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// DeleteByCandidate hard-removes a candidate's interviews (turns cascade via the
// FK), satisfying the right-to-erasure CandidateScopedEraser port (CAL-118).
func (r *InterviewRepo) DeleteByCandidate(ctx context.Context, candidateID kernel.ID) error {
	return r.q.DeleteInterviewsByCandidate(ctx, candidateID.String())
}

// interviewParams builds the create params, encrypting the report card at rest.
func (r *InterviewRepo) interviewParams(iv *interview.Interview) (sqlcdb.CreateInterviewParams, error) {
	report, confidence, err := r.marshalReport(iv.Report)
	if err != nil {
		return sqlcdb.CreateInterviewParams{}, err
	}
	created := iv.StartedAt
	if created.IsZero() {
		created = time.Now().UTC()
	}
	return sqlcdb.CreateInterviewParams{
		ID:          iv.ID.String(),
		RoleID:      iv.RoleID.String(),
		CandidateID: iv.CandidateID.String(),
		Mode:        iv.Mode.String(),
		Status:      iv.State.String(),
		ReportCard:  report,
		Confidence:  confidence,
		CreatedAt:   pgtype.Timestamptz{Time: created, Valid: true},
	}, nil
}

func (r *InterviewRepo) withTx(ctx context.Context, fn func(*sqlcdb.Queries) error) error {
	if r.tx == nil {
		return fn(r.q)
	}
	tx, err := r.tx.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit
	if err := fn(r.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *InterviewRepo) hydrate(ctx context.Context, row sqlcdb.TalentInterview) (*interview.Interview, error) {
	iv := &interview.Interview{
		ID:          kernel.ID(row.ID),
		RoleID:      kernel.ID(row.RoleID),
		CandidateID: kernel.ID(row.CandidateID),
		Mode:        dbToMode(row.Mode),
		State:       dbToState(row.Status),
		StartedAt:   row.CreatedAt.Time,
	}
	if len(row.ReportCard) > 0 {
		var card interview.ReportCard
		if err := decodeEncryptedJSON(r.cipher, row.ReportCard, &card); err != nil {
			return nil, err
		}
		iv.Report = &card
	}
	turns, err := r.q.ListInterviewTurns(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	for _, t := range turns {
		question, err := r.cipher.Decrypt(t.Question)
		if err != nil {
			return nil, err
		}
		if t.Answer.Valid {
			answer, err := r.cipher.Decrypt(t.Answer.String)
			if err != nil {
				return nil, err
			}
			iv.Turns = append(iv.Turns, interview.InterviewTurn{
				ID:            kernel.ID(t.ID),
				Ordinal:       int(t.Ordinal),
				Question:      question,
				Answer:        answer,
				CompetencyTag: t.CompetencyTag.String,
			})
			continue
		}
		iv.Pending = &interview.PendingQuestion{
			Ordinal:       int(t.Ordinal),
			Text:          question,
			CompetencyTag: t.CompetencyTag.String,
		}
	}
	return iv, nil
}

// insertTurns writes the answered turns (non-NULL answer) then, if present, the
// pending question as a single NULL-answer row. Question and answer text are
// encrypted at rest through the field cipher (CAL-117).
func (r *InterviewRepo) insertTurns(ctx context.Context, q *sqlcdb.Queries, iv *interview.Interview) error {
	for _, t := range iv.Turns {
		question, err := r.cipher.Encrypt(t.Question)
		if err != nil {
			return err
		}
		answer, err := r.cipher.Encrypt(t.Answer)
		if err != nil {
			return err
		}
		if err := q.InsertInterviewTurn(ctx, sqlcdb.InsertInterviewTurnParams{
			ID:            t.ID.String(),
			InterviewID:   iv.ID.String(),
			Ordinal:       clampInt32(t.Ordinal),
			Question:      question,
			Answer:        pgtype.Text{String: answer, Valid: true},
			CompetencyTag: optText(t.CompetencyTag),
		}); err != nil {
			return err
		}
	}
	if iv.Pending == nil {
		return nil
	}
	pendingQ, err := r.cipher.Encrypt(iv.Pending.Text)
	if err != nil {
		return err
	}
	return q.InsertInterviewTurn(ctx, sqlcdb.InsertInterviewTurnParams{
		ID:            kernel.NewID().String(),
		InterviewID:   iv.ID.String(),
		Ordinal:       clampInt32(iv.Pending.Ordinal),
		Question:      pendingQ,
		Answer:        pgtype.Text{Valid: false},
		CompetencyTag: optText(iv.Pending.CompetencyTag),
	})
}

// marshalReport renders the report card for the report_card JSONB column. The
// card carries transcript-quote evidence (candidate PII), so it is encrypted at
// rest through the shared field-cipher JSONB helper (CAL-117); the sibling
// confidence value is denormalized as plaintext for queryability.
func (r *InterviewRepo) marshalReport(card *interview.ReportCard) ([]byte, pgtype.Text, error) {
	if card == nil {
		return nil, pgtype.Text{Valid: false}, nil
	}
	b, err := encodeEncryptedJSON(r.cipher, card)
	if err != nil {
		return nil, pgtype.Text{}, err
	}
	return b, pgtype.Text{String: card.Confidence.String(), Valid: true}, nil
}

func optText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func dbToMode(s string) interview.InterviewMode {
	switch s {
	case "text":
		return interview.ModeText
	case "voice":
		return interview.ModeVoice
	default:
		return interview.ModeUnspecified
	}
}

func dbToState(s string) interview.State {
	switch s {
	case "open":
		return interview.StateOpen
	case "asking":
		return interview.StateAsking
	case "scoring":
		return interview.StateScoring
	case "closed":
		return interview.StateClosed
	default:
		return interview.StateUnspecified
	}
}

var _ interview.InterviewRepository = (*InterviewRepo)(nil)

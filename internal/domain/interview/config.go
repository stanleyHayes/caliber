package interview

import "time"

// Config caps the adaptive interview so it stays within the demo latency budget
// and respects the candidate's time. These are FSM-level policy constants, not
// prompt content, so they live in the pure domain.
type Config struct {
	MaxQuestions int
	MaxDuration  time.Duration
}

const (
	defaultMaxQuestions = 4
	defaultMaxDuration  = 10 * time.Minute
)

// DefaultConfig returns the standard interview caps: 4 questions or 10 minutes,
// whichever is reached first.
func DefaultConfig() Config {
	return Config{MaxQuestions: defaultMaxQuestions, MaxDuration: defaultMaxDuration}
}

// WithDefaults fills in any zero or invalid values from the built-in defaults.
func (c Config) WithDefaults() Config {
	if c.MaxQuestions <= 0 {
		c.MaxQuestions = defaultMaxQuestions
	}
	if c.MaxDuration <= 0 {
		c.MaxDuration = defaultMaxDuration
	}
	return c
}

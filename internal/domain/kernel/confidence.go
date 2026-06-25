package kernel

// Confidence is a coarse confidence level attached to matches and report cards.
type Confidence int

// Confidence levels.
const (
	ConfidenceUnknown Confidence = iota
	ConfidenceLow
	ConfidenceMedium
	ConfidenceHigh
)

// Valid reports whether the confidence is a known, non-zero level.
func (c Confidence) Valid() bool { return c >= ConfidenceLow && c <= ConfidenceHigh }

// String renders the confidence level.
func (c Confidence) String() string {
	switch c {
	case ConfidenceLow:
		return "low"
	case ConfidenceMedium:
		return "medium"
	case ConfidenceHigh:
		return "high"
	default:
		return "unknown"
	}
}

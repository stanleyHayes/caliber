package privacy

import (
	"encoding/json"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

// JSON renders the data export as a stable, indented JSON document — the artifact
// a candidate receives for a DSAR (Ghana DPA 2012, right of access). It includes
// every section so the subject sees exactly what is processed about them; an
// absent profile is rendered as null rather than omitted, so the shape is
// predictable for tooling.
func (d *DataExport) JSON() ([]byte, error) {
	doc := struct {
		Candidate    any `json:"candidate"`
		Profile      any `json:"profile"`
		Applications any `json:"applications"`
		Interviews   any `json:"interviews"`
		Contests     any `json:"contests"`
	}{
		Candidate:    d.Candidate,
		Profile:      d.Profile,
		Applications: nonNil(d.Applications),
		Interviews:   nonNil(d.Interviews),
		Contests:     nonNil(d.Contests),
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, kernel.Wrap(err, kernel.KindInternal, "privacy: encode data export")
	}
	return b, nil
}

// nonNil returns the slice as-is when populated, or an empty (non-nil) slice so a
// section with no records serialises as [] rather than null.
func nonNil[T any](s []*T) []*T {
	if s == nil {
		return []*T{}
	}
	return s
}

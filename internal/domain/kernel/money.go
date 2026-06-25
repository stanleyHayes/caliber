package kernel

import "strings"

// SalaryBand is a currency-denominated range. A zero band (Low==High==0) means
// "unspecified". Used by roles and candidate preferences.
type SalaryBand struct {
	Currency string
	Low      float64
	High     float64
}

// Validate checks the band is internally consistent.
func (b SalaryBand) Validate() error {
	if b.Low < 0 || b.High < 0 {
		return Invalid("salary band bounds must be non-negative")
	}
	if b.High != 0 && b.High < b.Low {
		return Invalid("salary band high must be >= low")
	}
	if (b.Low != 0 || b.High != 0) && strings.TrimSpace(b.Currency) == "" {
		return Invalid("salary band currency is required when bounds are set")
	}
	return nil
}

// IsZero reports whether the band is unspecified.
func (b SalaryBand) IsZero() bool { return b.Low == 0 && b.High == 0 }

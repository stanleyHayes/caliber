package matching

import (
	"reflect"
	"sort"
	"testing"

	"github.com/xcreativs/caliber/internal/domain/kernel"
)

func TestProtectedAttributes(t *testing.T) {
	got := ProtectedAttributes()
	want := []string{"age", "disability", "ethnicity", "gender", "marital_status", "nationality", "religion"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ProtectedAttributes() = %v, want %v", got, want)
	}
	if !sort.StringsAreSorted(got) {
		t.Fatalf("ProtectedAttributes() must be sorted")
	}

	// Mutating the returned slice must not affect future calls.
	got[0] = "MUTATED"
	again := ProtectedAttributes()
	if again[0] != "age" {
		t.Fatalf("ProtectedAttributes() returned a non-isolated slice")
	}
}

func TestEnsureBiasSafe(t *testing.T) {
	tests := []struct {
		name    string
		keys    []string
		wantErr bool
	}{
		{"empty", nil, false},
		{"all safe", []string{"years_experience", "skills", "tenure"}, false},
		{"exact protected", []string{"skills", "gender"}, true},
		{"uppercase protected", []string{"GENDER"}, true},
		{"mixed case protected", []string{"Ethnicity"}, true},
		{"whitespace padded protected", []string{"  age  "}, true},
		{"underscore protected", []string{"marital_status"}, true},
		{"safe lookalike", []string{"genders_team", "ageing_systems"}, false},
		{"disability protected", []string{"disability"}, true},
		{"religion protected", []string{"religion"}, true},
		{"nationality protected", []string{"nationality"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EnsureBiasSafe(tt.keys)
			if tt.wantErr != (err != nil) {
				t.Fatalf("EnsureBiasSafe(%v) err=%v, wantErr=%v", tt.keys, err, tt.wantErr)
			}
			if tt.wantErr && kernel.KindOf(err) != kernel.KindInvalid {
				t.Fatalf("expected KindInvalid, got %v", kernel.KindOf(err))
			}
		})
	}
}

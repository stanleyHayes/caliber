package kernel

import (
	"strings"
	"unicode"
)

// Tokens splits s into whole tokens on commas, slashes, semicolons, and
// whitespace, preserving other in-token punctuation so compound skill names stay
// intact ("C++", "C#", ".NET", "Node.js"). It is the single shared tokenizer for
// competency and location matching, so every gate — the must-have filter and the
// no-fabrication grounding check — tokenizes identically and cannot disagree
// about whether a profile covers a skill. Callers lower-case as needed.
func Tokens(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '/' || r == ';' || unicode.IsSpace(r)
	})
}

// HasPhrase reports whether phrase appears as a contiguous run of tokens within
// haystack (whole-token matching, so "go" is not found inside "ago"). An empty
// phrase never matches.
func HasPhrase(haystack, phrase []string) bool {
	if len(phrase) == 0 {
		return false
	}
	for start := 0; start+len(phrase) <= len(haystack); start++ {
		matched := true
		for j := range phrase {
			if haystack[start+j] != phrase[j] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// Package privacy provides best-effort masking of personal data (152-ФЗ
// technical measures) for values that may end up in diagnostic log fields. It is
// not a substitute for not logging sensitive content in the first place: the
// chat pipeline never logs message bodies or prompts. Redact is a safety net for
// the rare cases where a short string must be logged.
package privacy

import "regexp"

var (
	emailRe = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	// Phone-like sequences: optional +, then 10+ digits possibly separated by
	// spaces, dashes or parentheses.
	phoneRe = regexp.MustCompile(`\+?\d[\d\s\-()]{8,}\d`)
)

// Redact masks email addresses and phone-like sequences in s.
func Redact(s string) string {
	if s == "" {
		return s
	}
	s = emailRe.ReplaceAllString(s, "[email]")
	s = phoneRe.ReplaceAllString(s, "[phone]")
	return s
}

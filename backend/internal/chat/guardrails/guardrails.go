// Package guardrails implements the Б5 defense-in-depth checks layered on top of
// the chat pipeline: deterministic context gates, verifiable citation
// post-checks, a transparent confidence heuristic and a system-prompt leak
// filter. The functions are pure so they can be unit tested in isolation and
// reused by the eval harness.
package guardrails

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
)

// FilterCandidates applies the cheap deterministic gates to retriever output: it
// drops candidates scoring below minScore (when minScore > 0) and, when a theme
// allow-list is set, keeps only candidates whose theme is allowed. Candidates
// with no theme are always kept, mirroring the kb_embedding vector filter.
func FilterCandidates(items []domain.RAGCandidate, minScore float64, allowed map[uuid.UUID]bool) []domain.RAGCandidate {
	out := make([]domain.RAGCandidate, 0, len(items))
	for _, item := range items {
		if minScore > 0 && item.Score < minScore {
			continue
		}
		if len(allowed) > 0 && item.ThemeID != nil && !allowed[*item.ThemeID] {
			continue
		}
		out = append(out, item)
	}
	return out
}

// CitationResult is the outcome of comparing the citations in an answer against
// the set of source IDs actually supplied to the model.
type CitationResult struct {
	Cited      []string
	Valid      []string
	Fabricated []string
}

var citationRe = regexp.MustCompile(`\[([^\[\]\n]+)\]`)

// ExtractCitations returns the distinct source-id-like tokens cited inside square
// brackets in the answer. A single bracket group may contain several
// comma-separated ids.
func ExtractCitations(answer string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range citationRe.FindAllStringSubmatch(answer, -1) {
		for _, part := range strings.Split(m[1], ",") {
			token := strings.TrimSpace(part)
			if token == "" || !looksLikeSourceID(token) {
				continue
			}
			if !seen[token] {
				seen[token] = true
				out = append(out, token)
			}
		}
	}
	return out
}

// looksLikeSourceID keeps the extractor from treating ordinary bracketed prose
// (for example "[важно]") as a citation. Source IDs follow
// "entity_type:entity_id:chunk_idx" and therefore contain colons.
func looksLikeSourceID(token string) bool {
	return strings.Count(token, ":") >= 2
}

// CheckCitations classifies the citations in answer against validIDs. Fabricated
// citations (ids the model invented) are the signal that the answer is not
// grounded and must be rejected.
func CheckCitations(answer string, validIDs map[string]bool) CitationResult {
	res := CitationResult{}
	for _, id := range ExtractCitations(answer) {
		res.Cited = append(res.Cited, id)
		if validIDs[id] {
			res.Valid = append(res.Valid, id)
		} else {
			res.Fabricated = append(res.Fabricated, id)
		}
	}
	return res
}

// usedSourcesFooterRe matches a line that IS the service sources footer the
// prompt asks the model to append («Источники: [...]», «Использованные
// источники: ...»). Anchored so prose that merely mentions sources survives.
var usedSourcesFooterRe = regexp.MustCompile(`(?i)^(использ[а-яё]+\s+)?источники?\s*[:：]`)

var danglingSpaceRe = regexp.MustCompile(`[ \t]+([.,;:!?])`)

// sourcesTailRe matches a sources footer glued to the end of a prose line
// («… Чем могу помочь? Источники: []»), which the line-based footer stripper
// cannot remove. Only bracketed tokens and punctuation may follow the header so
// real prose is never eaten.
var sourcesTailRe = regexp.MustCompile(`(?i)(использ[а-яё]+\s+)?источники\s*[:：](\s|\[[^\]\n]*\]|[.,;])*$`)

// StripCitationMarkers removes machine-readable source_id citations from the
// user-facing answer text. Call only after CheckCitations so guardrails still
// see the raw model output.
func StripCitationMarkers(answer string) string {
	answer = stripInlineCitations(answer)
	answer = stripCitationFooter(answer)
	answer = sourcesTailRe.ReplaceAllString(answer, "")
	answer = danglingSpaceRe.ReplaceAllString(answer, "$1")
	return strings.TrimSpace(answer)
}

func stripInlineCitations(answer string) string {
	return citationRe.ReplaceAllStringFunc(answer, func(match string) string {
		submatch := citationRe.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		parts := strings.Split(submatch[1], ",")
		allSourceIDs := true
		hasToken := false
		for _, part := range parts {
			token := strings.TrimSpace(part)
			if token == "" {
				continue
			}
			hasToken = true
			if !looksLikeSourceID(token) {
				allSourceIDs = false
				break
			}
		}
		if hasToken && allSourceIDs {
			return ""
		}
		return match
	})
}

func stripCitationFooter(answer string) string {
	lines := strings.Split(answer, "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	for len(lines) > 0 {
		last := strings.TrimSpace(lines[len(lines)-1])
		if last == "" {
			lines = lines[:len(lines)-1]
			continue
		}
		if usedSourcesFooterRe.MatchString(last) || isCitationOnlyLine(last) {
			lines = lines[:len(lines)-1]
			continue
		}
		break
	}
	return strings.Join(lines, "\n")
}

// isCitationOnlyLine reports whether a trailing line carries no prose beyond
// bracketed references. All bracket groups count here (not only source-id-like
// ones): small models cite sources like «[price.example.ru]», and a trailing
// bracket-only line is citation debris, not an answer.
func isCitationOnlyLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	without := citationRe.ReplaceAllString(trimmed, "")
	without = strings.Map(func(r rune) rune {
		switch r {
		case '.', ',', ';', ':', ' ':
			return -1
		default:
			return r
		}
	}, without)
	return without == ""
}

var greetingPrefixRe = regexp.MustCompile(`(?i)^\s*(здравствуйте|добрый день|доброе утро|добрый вечер|приветствую|привет)[!.,\s]*`)

// StripRepeatedGreeting removes a leading greeting from an answer inside an
// ongoing dialog: a good administrator greets once, but small models re-greet
// on every turn regardless of prompt instructions. When the greeting is the
// whole message it is kept as is.
func StripRepeatedGreeting(answer string) string {
	stripped := strings.TrimSpace(greetingPrefixRe.ReplaceAllString(answer, ""))
	if stripped == "" || stripped == answer {
		return answer
	}
	runes := []rune(stripped)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// ConfidenceInput carries the transparent signals used to score an answer.
type ConfidenceInput struct {
	SourceCount    int
	ValidCitations int
	HasFabrication bool
}

// Confidence returns a value in [0,1] from explainable signals: having grounded
// context, citing it validly and citing more than one source raise confidence;
// any fabricated citation collapses it.
func Confidence(in ConfidenceInput) float64 {
	if in.HasFabrication {
		return 0
	}
	if in.SourceCount == 0 {
		return 0
	}
	score := 0.5
	if in.ValidCitations > 0 {
		score += 0.3
	}
	if in.SourceCount >= 2 {
		score += 0.2
	}
	if score > 1 {
		score = 1
	}
	return score
}

// leakMarkers are distinctive fragments of our system prompt / scaffolding. If
// the model echoes them back it is leaking instructions and the answer must be
// suppressed.
var leakMarkers = []string{
	"<context>",
	"</context>",
	"правила персоны",
	"правила стиля",
	"системный промпт",
	"system prompt",
	"выявление потребностей —",
}

// personaEchoPrefixes open our own system prompt («Ты — {имя}, администратор
// компании»). They are only a leak at the very start of the answer: matching
// them anywhere would brand ordinary prose («значит, ты — владелец собаки»)
// as a leak and silently refuse a perfectly good reply.
var personaEchoPrefixes = []string{"ты — ", "ты - ", "ты—", "ты-"}

// LeaksSystemPrompt reports whether the answer appears to expose the system
// prompt, persona scaffolding or context delimiters.
func LeaksSystemPrompt(answer string) bool {
	lower := strings.ToLower(answer)
	for _, marker := range leakMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	trimmed := strings.TrimSpace(lower)
	for _, prefix := range personaEchoPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

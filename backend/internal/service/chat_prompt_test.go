package service

import (
	"strings"
	"testing"

	"github.com/knowledgeos/backend/internal/chat/tools"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/llm"
)

func promptSettings() *domain.BotSettings {
	return &domain.BotSettings{
		PersonaName: "Администратор",
		PersonaTone: "friendly_professional",
	}
}

func TestBuildSystemPromptAdministratorStyle(t *testing.T) {
	prompt := buildSystemPrompt(promptSettings(), nil, nil, true)
	for _, want := range []string{
		"администратор компании",
		"на «вы»",
		"начни ответ с приветствия",
		"дружелюбный и профессиональный",
		"Не выдумывай цены, адреса, часы работы и услуги",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "friendly_professional") {
		t.Fatalf("tone slug must be translated, not passed through:\n%s", prompt)
	}
	// Without tools the prompt must not mention tools or booking.
	for _, absent := range []string{"yclients", "request_handoff", "search_knowledge"} {
		if strings.Contains(prompt, absent) {
			t.Fatalf("prompt without tools must not mention %q:\n%s", absent, prompt)
		}
	}
}

func TestBuildSystemPromptUnknownTonePassesThrough(t *testing.T) {
	settings := promptSettings()
	settings.PersonaTone = "строгий, но справедливый"
	prompt := buildSystemPrompt(settings, nil, nil, true)
	if !strings.Contains(prompt, "строгий, но справедливый") {
		t.Fatalf("free-text tone must pass through:\n%s", prompt)
	}
}

func TestBuildSystemPromptWithBookingTools(t *testing.T) {
	defs := []llm.Tool{
		{Name: "search_knowledge"},
		{Name: tools.RequestHandoffToolName},
		{Name: tools.ToolYClientsGetServices},
		{Name: tools.ToolYClientsGetStaff},
		{Name: tools.ToolYClientsGetTimes},
		{Name: tools.ToolYClientsCreateBooking},
	}
	prompt := buildSystemPrompt(promptSettings(), nil, defs, true)
	for _, want := range []string{
		"записываешь клиентов на приём",
		"yclients_get_services",
		"yclients_get_times",
		"Только после явного согласия клиента вызови yclients_create_booking",
		"числовые id не показывай",
		"request_handoff",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestBuildSystemPromptToolsWithoutBooking(t *testing.T) {
	defs := []llm.Tool{
		{Name: "search_knowledge"},
		{Name: tools.RequestHandoffToolName},
	}
	prompt := buildSystemPrompt(promptSettings(), nil, defs, true)
	if strings.Contains(prompt, "yclients") {
		t.Fatalf("prompt must not mention yclients when booking module is off:\n%s", prompt)
	}
	if !strings.Contains(prompt, "search_knowledge") || !strings.Contains(prompt, "request_handoff") {
		t.Fatalf("prompt must mention available tools:\n%s", prompt)
	}
}

// The greeting is decided in Go from the dialog history, not guessed by the
// model: the client complained the bot never greeted on the opening turn.
func TestBuildSystemPromptGreetsOnlyOnFirstReply(t *testing.T) {
	first := buildSystemPrompt(promptSettings(), nil, nil, true)
	if !strings.Contains(first, "начни ответ с приветствия") {
		t.Fatalf("first reply must instruct to greet:\n%s", first)
	}
	if strings.Contains(first, "не здоровайся повторно") {
		t.Fatalf("first reply must not suppress the greeting:\n%s", first)
	}

	later := buildSystemPrompt(promptSettings(), nil, nil, false)
	if !strings.Contains(later, "не здоровайся повторно") {
		t.Fatalf("later replies must suppress the greeting:\n%s", later)
	}
	if strings.Contains(later, "начни ответ с приветствия") {
		t.Fatalf("later replies must not ask for a greeting:\n%s", later)
	}
}

// Needs discovery: the bot must gather the client's situation, but ask one
// question at a time (a conversation, not a bulk questionnaire).
func TestBuildSystemPromptRequiresNeedsDiscovery(t *testing.T) {
	prompt := buildSystemPrompt(promptSettings(), nil, nil, true)
	for _, want := range []string{
		"Выявление потребностей",
		"Односложные справки недопустимы",
		"по одному уточняющему вопросу за раз",
		"Не переспрашивай то, что клиент уже назвал",
		"пересчитай ответ под новые данные",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}

	// Old rules that contradict the current behaviour must stay removed:
	// the terse «one concrete question» rule, and the bulk «2–5 questions
	// at once» list format the client later rejected.
	for _, absent := range []string{
		"задай один конкретный вопрос",
		"обычно 1–4 предложения",
		"от 2 до 5 в одном сообщении",
	} {
		if strings.Contains(prompt, absent) {
			t.Fatalf("prompt must no longer contain %q:\n%s", absent, prompt)
		}
	}
}

func TestBuildSystemPromptContextAndPersonaRules(t *testing.T) {
	settings := promptSettings()
	settings.PersonaRules = "Ты работаешь в ветеринарной клинике."
	sources := []domain.ChatSource{{SourceID: "qa:abc:0", Title: "Цена", Content: "800 рублей"}}
	prompt := buildSystemPrompt(settings, sources, nil, true)
	for _, want := range []string{
		"Правила персоны:\nТы работаешь в ветеринарной клинике.",
		"[qa:abc:0] Цена\n800 рублей",
		"<context>",
		"</context>",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

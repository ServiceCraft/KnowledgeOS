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
	prompt := buildSystemPrompt(promptSettings(), nil, nil)
	for _, want := range []string{
		"администратор компании",
		"на «вы»",
		"Здоровайся только в самом первом сообщении",
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
	prompt := buildSystemPrompt(settings, nil, nil)
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
	prompt := buildSystemPrompt(promptSettings(), nil, defs)
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
	prompt := buildSystemPrompt(promptSettings(), nil, defs)
	if strings.Contains(prompt, "yclients") {
		t.Fatalf("prompt must not mention yclients when booking module is off:\n%s", prompt)
	}
	if !strings.Contains(prompt, "search_knowledge") || !strings.Contains(prompt, "request_handoff") {
		t.Fatalf("prompt must mention available tools:\n%s", prompt)
	}
}

func TestBuildSystemPromptContextAndPersonaRules(t *testing.T) {
	settings := promptSettings()
	settings.PersonaRules = "Ты работаешь в ветеринарной клинике."
	sources := []domain.ChatSource{{SourceID: "qa:abc:0", Title: "Цена", Content: "800 рублей"}}
	prompt := buildSystemPrompt(settings, sources, nil)
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

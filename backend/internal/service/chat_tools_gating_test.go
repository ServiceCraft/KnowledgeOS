package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/chat/tools"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/llm"
)

type stubToolset struct {
	defs []llm.Tool
}

func (s stubToolset) Definitions() []llm.Tool { return s.defs }

func (s stubToolset) Execute(ctx context.Context, companyID uuid.UUID, call llm.ToolCall) (tools.Result, error) {
	return tools.Result{}, nil
}

// toolNames returns the set of advertised tool names for a given EnabledModules.
func toolNamesFor(t *testing.T, modules string) map[string]bool {
	t.Helper()
	svc := &ChatService{toolset: stubToolset{defs: []llm.Tool{
		{Name: "search_knowledge"},
		{Name: tools.ToolYClientsListBranches},
		{Name: tools.ToolYClientsGetServices},
		{Name: tools.ToolYClientsGetStaff},
		{Name: tools.ToolYClientsGetTimes},
		{Name: tools.ToolYClientsCreateBooking},
	}}}
	settings := &domain.BotSettings{EnabledModules: json.RawMessage(modules)}
	names := map[string]bool{}
	for _, d := range svc.toolDefinitions(settings) {
		names[d.Name] = true
	}
	return names
}

func TestYClientsReadOnlyWithoutAutobook(t *testing.T) {
	names := toolNamesFor(t, `{"yclients_booking":true}`)

	for _, read := range []string{tools.ToolYClientsListBranches, tools.ToolYClientsGetServices, tools.ToolYClientsGetStaff, tools.ToolYClientsGetTimes} {
		if !names[read] {
			t.Fatalf("read tool %q must be advertised when yclients_booking is on", read)
		}
	}
	if names[tools.ToolYClientsCreateBooking] {
		t.Fatalf("create_booking must NOT be advertised without yclients_autobook (read-only phase-1 mode)")
	}
	if !names["search_knowledge"] {
		t.Fatalf("ungated tool search_knowledge must always be advertised")
	}
}

func TestYClientsAutobookEnablesCreateBooking(t *testing.T) {
	names := toolNamesFor(t, `{"yclients_booking":true,"yclients_autobook":true}`)
	if !names[tools.ToolYClientsCreateBooking] {
		t.Fatalf("create_booking must be advertised when yclients_autobook is on")
	}
}

func TestYClientsAutobookAloneStillNeedsReadTools(t *testing.T) {
	// Autobook on but booking off: create tool is gated by its own module and
	// appears, but the read tools it depends on stay hidden — a misconfiguration
	// we surface rather than silently paper over.
	names := toolNamesFor(t, `{"yclients_autobook":true}`)
	if names[tools.ToolYClientsGetServices] {
		t.Fatalf("read tools must stay hidden when yclients_booking is off")
	}
	if !names[tools.ToolYClientsCreateBooking] {
		t.Fatalf("create_booking follows yclients_autobook independently")
	}
}

func TestYClientsAllHiddenWhenModulesOff(t *testing.T) {
	names := toolNamesFor(t, `{}`)
	for _, n := range []string{tools.ToolYClientsGetServices, tools.ToolYClientsGetStaff, tools.ToolYClientsGetTimes, tools.ToolYClientsCreateBooking} {
		if names[n] {
			t.Fatalf("yclients tool %q must be hidden when no modules are enabled", n)
		}
	}
}

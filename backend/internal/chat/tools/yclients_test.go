package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/integrations/yclients"
)

type fakeSecretReader struct {
	token string
	meta  json.RawMessage
	err   error
}

func (f fakeSecretReader) GetPlaintextWithMetadata(_ context.Context, _ uuid.UUID, _ domain.SecretKind) (string, json.RawMessage, error) {
	return f.token, f.meta, f.err
}

type fakeYClientsAPI struct {
	services   []yclients.Service
	staff      []yclients.Staff
	slots      []yclients.Slot
	bookResult []yclients.BookResult
	lastReq    yclients.BookRequest
	lastCompID string
}

func (f *fakeYClientsAPI) GetServices(_ context.Context, companyID string) ([]yclients.Service, error) {
	f.lastCompID = companyID
	return f.services, nil
}
func (f *fakeYClientsAPI) GetStaff(_ context.Context, _ string, _ []int) ([]yclients.Staff, error) {
	return f.staff, nil
}
func (f *fakeYClientsAPI) GetBookTimes(_ context.Context, _ string, _ int, _ string) ([]yclients.Slot, error) {
	return f.slots, nil
}
func (f *fakeYClientsAPI) CreateBookRecord(_ context.Context, companyID string, req yclients.BookRequest) ([]yclients.BookResult, error) {
	f.lastCompID = companyID
	f.lastReq = req
	return f.bookResult, nil
}

func factoryFor(api YClientsAPI) YClientsClientFactory {
	return func(_, _ string) YClientsAPI { return api }
}

func TestYClientsCreateBookingHappyPath(t *testing.T) {
	api := &fakeYClientsAPI{
		services:   []yclients.Service{{ID: 7, Title: "УЗИ"}},
		bookResult: []yclients.BookResult{{RecordID: 99}},
	}
	secrets := fakeSecretReader{token: "partner", meta: json.RawMessage(`{"company_id":"777"}`)}
	tool := NewCreateYClientsBookingTool(secrets, factoryFor(api))

	args := json.RawMessage(`{"fullname":"Иван","phone":"+7 (999) 123-45-67","service_id":7,"staff_id":3,"datetime":"2026-07-10T10:00:00+03:00"}`)
	res, err := tool.Execute(context.Background(), uuid.New(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(res.Content, `"record_id":99`) {
		t.Fatalf("unexpected content: %s", res.Content)
	}
	if api.lastCompID != "777" {
		t.Errorf("expected company_id 777, got %q", api.lastCompID)
	}
	if api.lastReq.Phone != "79991234567" {
		t.Errorf("expected normalized phone, got %q", api.lastReq.Phone)
	}
	if len(api.lastReq.Appointments) != 1 || api.lastReq.Appointments[0].Services[0] != 7 {
		t.Errorf("unexpected appointment: %+v", api.lastReq.Appointments)
	}
}

func TestYClientsCreateBookingRejectsUnknownService(t *testing.T) {
	// Модель не должна создавать запись с выдуманным service_id: инструмент
	// сверяет id с реальным списком услуг и возвращает ошибку-подсказку.
	api := &fakeYClientsAPI{
		services:   []yclients.Service{{ID: 7, Title: "УЗИ"}},
		bookResult: []yclients.BookResult{{RecordID: 99}},
	}
	secrets := fakeSecretReader{token: "partner", meta: json.RawMessage(`{"company_id":"777"}`)}
	tool := NewCreateYClientsBookingTool(secrets, factoryFor(api))

	args := json.RawMessage(`{"fullname":"Иван","phone":"+79991234567","service_id":1,"staff_id":3,"datetime":"2026-07-10T10:00:00+03:00"}`)
	res, err := tool.Execute(context.Background(), uuid.New(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(res.Content, "не найден среди услуг") {
		t.Fatalf("expected unknown-service error, got: %s", res.Content)
	}
	if api.lastReq.Phone != "" {
		t.Fatalf("booking must not be created for unknown service, got request: %+v", api.lastReq)
	}
}

func TestYClientsResolveMissingSecret(t *testing.T) {
	secrets := fakeSecretReader{err: errors.New("secret not found")}
	tool := NewGetYClientsServicesTool(secrets, factoryFor(&fakeYClientsAPI{}))

	res, err := tool.Execute(context.Background(), uuid.New(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute should not error on missing secret: %v", err)
	}
	if !strings.Contains(res.Content, "error") || !strings.Contains(res.Content, "не настроена") {
		t.Fatalf("expected not-configured payload, got %s", res.Content)
	}
}

func TestYClientsResolveIncompleteConfig(t *testing.T) {
	// token present but company_id missing
	secrets := fakeSecretReader{token: "partner", meta: json.RawMessage(`{}`)}
	tool := NewGetYClientsServicesTool(secrets, factoryFor(&fakeYClientsAPI{}))

	res, _ := tool.Execute(context.Background(), uuid.New(), json.RawMessage(`{}`))
	if !strings.Contains(res.Content, "не полностью") {
		t.Fatalf("expected incomplete-config payload, got %s", res.Content)
	}
}

func TestYClientsGetServices(t *testing.T) {
	api := &fakeYClientsAPI{services: []yclients.Service{{ID: 1, Title: "Стрижка", SeanceLength: 3600}}}
	secrets := fakeSecretReader{token: "partner", meta: json.RawMessage(`{"company_id":"5"}`)}
	tool := NewGetYClientsServicesTool(secrets, factoryFor(api))

	res, err := tool.Execute(context.Background(), uuid.New(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(res.Content, "Стрижка") || !strings.Contains(res.Content, `"duration_min":60`) {
		t.Fatalf("unexpected content: %s", res.Content)
	}
}

func TestParseYClientsMetaNumericCompanyID(t *testing.T) {
	// Legacy single company_id stored as a JSON number must still resolve to a
	// single branch whose id is the stringified number.
	branches, baseURL := parseYClientsMeta(json.RawMessage(`{"company_id":777,"api_base":"https://x"}`))
	if len(branches) != 1 || branches[0].ID != "777" {
		t.Fatalf("expected single branch 777, got %+v", branches)
	}
	if baseURL != "https://x" {
		t.Fatalf("expected api_base https://x, got %q", baseURL)
	}
}

func TestParseYClientsMetaBranches(t *testing.T) {
	branches, _ := parseYClientsMeta(json.RawMessage(`{"branches":[{"id":"111","name":"На Ленина","address":"ул. Ленина, 1"},{"company_id":222,"name":"На Мира"}]}`))
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %d (%+v)", len(branches), branches)
	}
	if branches[0].ID != "111" || branches[0].Name != "На Ленина" || branches[0].Address != "ул. Ленина, 1" {
		t.Fatalf("unexpected first branch: %+v", branches[0])
	}
	if branches[1].ID != "222" || branches[1].Name != "На Мира" {
		t.Fatalf("unexpected second branch: %+v", branches[1])
	}
}

func TestSelectBranch(t *testing.T) {
	one := []yclientsBranch{{ID: "111"}}
	if id, errRes := selectBranch(one, ""); errRes != nil || id != "111" {
		t.Fatalf("single branch should auto-select: id=%q err=%v", id, errRes)
	}
	multi := []yclientsBranch{{ID: "111"}, {ID: "222"}}
	if _, errRes := selectBranch(multi, ""); errRes == nil {
		t.Fatal("multi branch without company_id must return an error result asking to clarify")
	}
	if id, errRes := selectBranch(multi, "222"); errRes != nil || id != "222" {
		t.Fatalf("valid company_id should select it: id=%q err=%v", id, errRes)
	}
	if _, errRes := selectBranch(multi, "999"); errRes == nil {
		t.Fatal("unknown company_id must return an error result")
	}
}

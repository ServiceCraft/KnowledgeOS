package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/integrations/yclients"
	applog "github.com/knowledgeos/backend/internal/logger"
)

// YClients tool names and the EnabledModules keys that gate them. Read-only
// tools (branches/services/staff/times) are gated by ModuleYClientsBooking so the
// bot can show real availability. Writing a booking is gated separately by
// ModuleYClientsAutobook: when it is off, the bot has no create tool and falls
// back to handoff, letting an operator enter the record manually (phase-1 mode).
const (
	ToolYClientsListBranches  = "yclients_list_branches"
	ToolYClientsGetServices   = "yclients_get_services"
	ToolYClientsGetStaff      = "yclients_get_staff"
	ToolYClientsGetTimes      = "yclients_get_times"
	ToolYClientsCreateBooking = "yclients_create_booking"

	ModuleYClientsBooking  = "yclients_booking"
	ModuleYClientsAutobook = "yclients_autobook"
)

// YClientsAPI is the subset of the YCLIENTS client used by the booking tools.
type YClientsAPI interface {
	GetServices(ctx context.Context, companyID string) ([]yclients.Service, error)
	GetStaff(ctx context.Context, companyID string, serviceIDs []int) ([]yclients.Staff, error)
	GetBookTimes(ctx context.Context, companyID string, staffID int, date string) ([]yclients.Slot, error)
	CreateBookRecord(ctx context.Context, companyID string, req yclients.BookRequest) ([]yclients.BookResult, error)
}

// YClientsClientFactory builds a per-company API client from the stored partner
// token and optional base URL override.
type YClientsClientFactory func(partnerToken, baseURL string) YClientsAPI

// yclientsSecretReader reads the decrypted YClients secret plus its metadata.
type yclientsSecretReader interface {
	GetPlaintextWithMetadata(ctx context.Context, companyID uuid.UUID, kind domain.SecretKind) (string, json.RawMessage, error)
}

// yclientsBranch is one bookable filial of the clinic. A tenant can configure a
// single filial (legacy company_id) or a list; the bot asks the client which one
// they want before serving availability for it.
type yclientsBranch struct {
	ID      string
	Name    string
	Address string
}

// yclientsBase resolves the per-company YClients client from tenant secrets and
// is embedded by every YClients tool.
type yclientsBase struct {
	secrets yclientsSecretReader
	factory YClientsClientFactory
}

// resolve loads credentials for the company and builds an API client together
// with the configured filials. When the integration is missing or incomplete it
// returns a non-nil *Result carrying an error payload for the model (so the bot
// can tell the user) and a nil client.
func (b yclientsBase) resolve(ctx context.Context, companyID uuid.UUID) (YClientsAPI, []yclientsBranch, *Result) {
	token, meta, err := b.secrets.GetPlaintextWithMetadata(ctx, companyID, domain.SecretKindYClients)
	if err != nil {
		return nil, nil, yclientsErr("Интеграция YClients не настроена. Задайте партнёрский токен и company_id филиала в настройках интеграций.")
	}
	branches, baseURL := parseYClientsMeta(meta)
	if strings.TrimSpace(token) == "" || len(branches) == 0 {
		return nil, nil, yclientsErr("Интеграция YClients настроена не полностью: нужны партнёрский токен и хотя бы один филиал (company_id).")
	}
	return b.factory(token, baseURL), branches, nil
}

// selectBranch picks the filial company_id to operate on. With a single filial
// it is used automatically; with several the model must pass the client's chosen
// company_id, otherwise it is told to ask via yclients_list_branches.
func selectBranch(branches []yclientsBranch, requested string) (string, *Result) {
	requested = strings.TrimSpace(requested)
	if len(branches) == 0 {
		return "", yclientsErr("Интеграция YClients настроена не полностью: не задан ни один филиал.")
	}
	if requested != "" {
		for _, br := range branches {
			if br.ID == requested {
				return br.ID, nil
			}
		}
		return "", yclientsErr(fmt.Sprintf("Филиал company_id=%s не найден. Вызови yclients_list_branches и уточни у клиента, какой филиал ему подходит.", requested))
	}
	if len(branches) == 1 {
		return branches[0].ID, nil
	}
	return "", yclientsErr("В клинике несколько филиалов. Сначала вызови yclients_list_branches, уточни у клиента нужный филиал и передай его company_id в этот инструмент.")
}

// parseYClientsMeta reads filials and the optional API base override from the
// stored secret metadata. It accepts a multi-filial `branches` array and falls
// back to a single legacy `company_id` for backward compatibility.
func parseYClientsMeta(raw json.RawMessage) (branches []yclientsBranch, baseURL string) {
	if len(raw) == 0 {
		return nil, ""
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, ""
	}
	baseURL = jsonScalarString(m["api_base"])
	if rawBranches, ok := m["branches"]; ok {
		var arr []map[string]json.RawMessage
		if json.Unmarshal(rawBranches, &arr) == nil {
			for _, e := range arr {
				id := jsonScalarString(e["id"])
				if id == "" {
					id = jsonScalarString(e["company_id"])
				}
				if id == "" {
					continue
				}
				branches = append(branches, yclientsBranch{
					ID:      id,
					Name:    jsonScalarString(e["name"]),
					Address: jsonScalarString(e["address"]),
				})
			}
		}
	}
	if len(branches) == 0 {
		if cid := jsonScalarString(m["company_id"]); cid != "" {
			branches = append(branches, yclientsBranch{ID: cid})
		}
	}
	return branches, baseURL
}

// branchIDFromArgs extracts the optional company_id (filial) argument the model
// may pass to any YClients tool, coercing a string or number.
func branchIDFromArgs(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	return jsonScalarString(m["company_id"])
}

// jsonScalarString coerces a JSON string or number to a trimmed string.
func jsonScalarString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	return ""
}

func yclientsErr(msg string) *Result {
	payload, _ := json.Marshal(map[string]string{"error": msg})
	return &Result{Content: string(payload)}
}

func yclientsResult(v any) (Result, error) {
	payload, err := json.Marshal(v)
	if err != nil {
		return Result{}, err
	}
	return Result{Content: string(payload)}, nil
}

// companyIDSchemaProp is the shared JSON-schema fragment for the optional filial
// argument every availability/booking tool accepts.
const companyIDSchemaProp = `"company_id": {"type": "string", "description": "ID филиала из yclients_list_branches. Если филиал один — можно не указывать."}`

// --- yclients_list_branches --------------------------------------------------

// ListYClientsBranchesTool lists the configured filials so the bot can ask the
// client which one they want before showing availability.
type ListYClientsBranchesTool struct{ yclientsBase }

// NewListYClientsBranchesTool executes the tools.NewListYClientsBranchesTool operation.
func NewListYClientsBranchesTool(secrets yclientsSecretReader, factory YClientsClientFactory) *ListYClientsBranchesTool {
	return &ListYClientsBranchesTool{yclientsBase{secrets: secrets, factory: factory}}
}

func (t *ListYClientsBranchesTool) Name() string { return ToolYClientsListBranches }

func (t *ListYClientsBranchesTool) Description() string {
	return "Получить список филиалов клиники для записи в YClients (company_id, название, адрес). Если филиалов несколько — уточни у клиента нужный и передавай его company_id в остальные инструменты YClients."
}

func (t *ListYClientsBranchesTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (t *ListYClientsBranchesTool) Execute(ctx context.Context, companyID uuid.UUID, _ json.RawMessage) (Result, error) {
	applog.TraceCall(ctx, "tools.ListYClientsBranchesTool.Execute")
	_, branches, errResult := t.resolve(ctx, companyID)
	if errResult != nil {
		return *errResult, nil
	}
	items := make([]map[string]any, 0, len(branches))
	for _, br := range branches {
		items = append(items, map[string]any{
			"company_id": br.ID,
			"name":       br.Name,
			"address":    br.Address,
		})
	}
	return yclientsResult(map[string]any{"branches": items})
}

// --- yclients_get_services ---------------------------------------------------

// GetYClientsServicesTool lists services bookable via YClients.
type GetYClientsServicesTool struct{ yclientsBase }

// NewGetYClientsServicesTool executes the tools.NewGetYClientsServicesTool operation.
func NewGetYClientsServicesTool(secrets yclientsSecretReader, factory YClientsClientFactory) *GetYClientsServicesTool {
	return &GetYClientsServicesTool{yclientsBase{secrets: secrets, factory: factory}}
}

func (t *GetYClientsServicesTool) Name() string { return ToolYClientsGetServices }

func (t *GetYClientsServicesTool) Description() string {
	return "Получить список услуг для записи в YClients с их id, названием и ценой. Вызывай перед записью, чтобы узнать service_id. Если филиалов несколько, укажи company_id нужного филиала."
}

func (t *GetYClientsServicesTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{` + companyIDSchemaProp + `}}`)
}

func (t *GetYClientsServicesTool) Execute(ctx context.Context, companyID uuid.UUID, args json.RawMessage) (Result, error) {
	applog.TraceCall(ctx, "tools.GetYClientsServicesTool.Execute")
	api, branches, errResult := t.resolve(ctx, companyID)
	if errResult != nil {
		return *errResult, nil
	}
	ycCompany, errResult := selectBranch(branches, branchIDFromArgs(args))
	if errResult != nil {
		return *errResult, nil
	}
	services, err := api.GetServices(ctx, ycCompany)
	if err != nil {
		return *yclientsErr(err.Error()), nil
	}
	items := make([]map[string]any, 0, len(services))
	for _, s := range services {
		items = append(items, map[string]any{
			"id":           s.ID,
			"title":        s.Title,
			"price_min":    s.PriceMin,
			"price_max":    s.PriceMax,
			"duration_min": s.SeanceLength / 60,
		})
	}
	return yclientsResult(map[string]any{"services": items})
}

// --- yclients_get_staff ------------------------------------------------------

// GetYClientsStaffTool lists specialists bookable via YClients.
type GetYClientsStaffTool struct{ yclientsBase }

// NewGetYClientsStaffTool executes the tools.NewGetYClientsStaffTool operation.
func NewGetYClientsStaffTool(secrets yclientsSecretReader, factory YClientsClientFactory) *GetYClientsStaffTool {
	return &GetYClientsStaffTool{yclientsBase{secrets: secrets, factory: factory}}
}

func (t *GetYClientsStaffTool) Name() string { return ToolYClientsGetStaff }

func (t *GetYClientsStaffTool) Description() string {
	return "Получить список специалистов (мастеров) для записи в YClients с их id и именем. Можно отфильтровать по service_id. Если филиалов несколько, укажи company_id нужного филиала."
}

func (t *GetYClientsStaffTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"service_id": {"type": "integer", "description": "Показать только мастеров, оказывающих эту услугу (необязательно)"},
			` + companyIDSchemaProp + `
		}
	}`)
}

type getYClientsStaffArgs struct {
	ServiceID int `json:"service_id"`
}

func (t *GetYClientsStaffTool) Execute(ctx context.Context, companyID uuid.UUID, args json.RawMessage) (Result, error) {
	applog.TraceCall(ctx, "tools.GetYClientsStaffTool.Execute")
	var in getYClientsStaffArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &in); err != nil {
			return Result{}, &ValidationError{Msg: "invalid arguments for yclients_get_staff"}
		}
	}
	api, branches, errResult := t.resolve(ctx, companyID)
	if errResult != nil {
		return *errResult, nil
	}
	ycCompany, errResult := selectBranch(branches, branchIDFromArgs(args))
	if errResult != nil {
		return *errResult, nil
	}
	var serviceIDs []int
	if in.ServiceID > 0 {
		serviceIDs = []int{in.ServiceID}
	}
	staff, err := api.GetStaff(ctx, ycCompany, serviceIDs)
	if err != nil {
		return *yclientsErr(err.Error()), nil
	}
	items := make([]map[string]any, 0, len(staff))
	for _, s := range staff {
		items = append(items, map[string]any{
			"id":             s.ID,
			"name":           s.Name,
			"specialization": s.Specialization,
		})
	}
	return yclientsResult(map[string]any{"staff": items})
}

// --- yclients_get_times ------------------------------------------------------

// GetYClientsTimesTool lists free booking slots for a staff member on a day.
type GetYClientsTimesTool struct{ yclientsBase }

// NewGetYClientsTimesTool executes the tools.NewGetYClientsTimesTool operation.
func NewGetYClientsTimesTool(secrets yclientsSecretReader, factory YClientsClientFactory) *GetYClientsTimesTool {
	return &GetYClientsTimesTool{yclientsBase{secrets: secrets, factory: factory}}
}

func (t *GetYClientsTimesTool) Name() string { return ToolYClientsGetTimes }

func (t *GetYClientsTimesTool) Description() string {
	return "Получить свободные окна для записи к мастеру на конкретную дату. Возвращает datetime в формате ISO8601 — используй его как есть при создании записи. Если филиалов несколько, укажи company_id нужного филиала."
}

func (t *GetYClientsTimesTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"staff_id": {"type": "integer", "description": "ID мастера из yclients_get_staff"},
			"date": {"type": "string", "minLength": 10, "description": "Дата в формате YYYY-MM-DD"},
			` + companyIDSchemaProp + `
		},
		"required": ["staff_id", "date"]
	}`)
}

type getYClientsTimesArgs struct {
	StaffID int    `json:"staff_id"`
	Date    string `json:"date"`
}

func (t *GetYClientsTimesTool) Execute(ctx context.Context, companyID uuid.UUID, args json.RawMessage) (Result, error) {
	applog.TraceCall(ctx, "tools.GetYClientsTimesTool.Execute")
	var in getYClientsTimesArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return Result{}, &ValidationError{Msg: "invalid arguments for yclients_get_times"}
	}
	if in.StaffID <= 0 || strings.TrimSpace(in.Date) == "" {
		return Result{}, &ValidationError{Msg: "yclients_get_times requires staff_id and date"}
	}
	api, branches, errResult := t.resolve(ctx, companyID)
	if errResult != nil {
		return *errResult, nil
	}
	ycCompany, errResult := selectBranch(branches, branchIDFromArgs(args))
	if errResult != nil {
		return *errResult, nil
	}
	slots, err := api.GetBookTimes(ctx, ycCompany, in.StaffID, strings.TrimSpace(in.Date))
	if err != nil {
		return *yclientsErr(err.Error()), nil
	}
	items := make([]map[string]any, 0, len(slots))
	for _, s := range slots {
		items = append(items, map[string]any{"time": s.Time, "datetime": s.Datetime})
	}
	return yclientsResult(map[string]any{"slots": items})
}

// --- yclients_create_booking -------------------------------------------------

// CreateYClientsBookingTool creates a booking in YClients.
type CreateYClientsBookingTool struct{ yclientsBase }

// NewCreateYClientsBookingTool executes the tools.NewCreateYClientsBookingTool operation.
func NewCreateYClientsBookingTool(secrets yclientsSecretReader, factory YClientsClientFactory) *CreateYClientsBookingTool {
	return &CreateYClientsBookingTool{yclientsBase{secrets: secrets, factory: factory}}
}

func (t *CreateYClientsBookingTool) Name() string { return ToolYClientsCreateBooking }

func (t *CreateYClientsBookingTool) Description() string {
	return "Создать запись клиента в YClients. Перед вызовом подтверди у клиента услугу, мастера, дату и время, имя и телефон. service_id и staff_id бери из yclients_get_services/yclients_get_staff, datetime — из yclients_get_times. Если филиалов несколько, укажи company_id того же филиала, по которому смотрел услуги и время."
}

func (t *CreateYClientsBookingTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"fullname": {"type": "string", "minLength": 1, "description": "Имя клиента"},
			"phone": {"type": "string", "minLength": 5, "description": "Телефон клиента"},
			"service_id": {"type": "integer", "description": "ID услуги из yclients_get_services"},
			"staff_id": {"type": "integer", "description": "ID мастера из yclients_get_staff"},
			"datetime": {"type": "string", "minLength": 10, "description": "Время записи в ISO8601 из yclients_get_times"},
			"email": {"type": "string", "description": "Email клиента (необязательно)"},
			"comment": {"type": "string", "description": "Комментарий к записи (необязательно)"},
			` + companyIDSchemaProp + `
		},
		"required": ["fullname", "phone", "service_id", "staff_id", "datetime"]
	}`)
}

type createYClientsBookingArgs struct {
	Fullname  string `json:"fullname"`
	Phone     string `json:"phone"`
	ServiceID int    `json:"service_id"`
	StaffID   int    `json:"staff_id"`
	Datetime  string `json:"datetime"`
	Email     string `json:"email"`
	Comment   string `json:"comment"`
}

func (t *CreateYClientsBookingTool) Execute(ctx context.Context, companyID uuid.UUID, args json.RawMessage) (Result, error) {
	applog.TraceCall(ctx, "tools.CreateYClientsBookingTool.Execute")
	var in createYClientsBookingArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return Result{}, &ValidationError{Msg: "invalid arguments for yclients_create_booking"}
	}
	if strings.TrimSpace(in.Fullname) == "" || in.ServiceID <= 0 || in.StaffID <= 0 || strings.TrimSpace(in.Datetime) == "" {
		return Result{}, &ValidationError{Msg: "yclients_create_booking requires fullname, phone, service_id, staff_id and datetime"}
	}
	phone := yclients.NormalizePhone(in.Phone)
	if phone == "" {
		return Result{}, &ValidationError{Msg: "yclients_create_booking requires a valid phone number"}
	}
	api, branches, errResult := t.resolve(ctx, companyID)
	if errResult != nil {
		return *errResult, nil
	}
	ycCompany, errResult := selectBranch(branches, branchIDFromArgs(args))
	if errResult != nil {
		return *errResult, nil
	}
	// The model must book real services, not guessed ids: verify service_id
	// against the live list before creating the record.
	services, err := api.GetServices(ctx, ycCompany)
	if err != nil {
		return *yclientsErr(err.Error()), nil
	}
	validService := false
	for _, s := range services {
		if s.ID == in.ServiceID {
			validService = true
			break
		}
	}
	if !validService {
		return *yclientsErr(fmt.Sprintf("service_id %d не найден среди услуг филиала — вызови yclients_get_services по нужному company_id и используй реальный id услуги", in.ServiceID)), nil
	}
	results, err := api.CreateBookRecord(ctx, ycCompany, yclients.BookRequest{
		Phone:    phone,
		Fullname: strings.TrimSpace(in.Fullname),
		Email:    strings.TrimSpace(in.Email),
		Comment:  strings.TrimSpace(in.Comment),
		Appointments: []yclients.Appointment{{
			ID:       1,
			Services: []int{in.ServiceID},
			StaffID:  in.StaffID,
			Datetime: strings.TrimSpace(in.Datetime),
		}},
	})
	if err != nil {
		return *yclientsErr(err.Error()), nil
	}
	if len(results) == 0 {
		return *yclientsErr("YClients не вернул созданную запись"), nil
	}
	return yclientsResult(map[string]any{
		"status":    "created",
		"record_id": results[0].RecordID,
		"message":   fmt.Sprintf("Запись создана (record_id %d)", results[0].RecordID),
	})
}

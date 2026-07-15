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

// YClients tool names and the EnabledModules key that gates them. The chat
// service uses ModuleYClientsBooking to advertise/execute these tools only for
// companies that have the booking module switched on.
const (
	ToolYClientsGetServices   = "yclients_get_services"
	ToolYClientsGetStaff      = "yclients_get_staff"
	ToolYClientsGetTimes      = "yclients_get_times"
	ToolYClientsCreateBooking = "yclients_create_booking"

	ModuleYClientsBooking = "yclients_booking"
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

// yclientsBase resolves the per-company YClients client from tenant secrets and
// is embedded by every YClients tool.
type yclientsBase struct {
	secrets yclientsSecretReader
	factory YClientsClientFactory
}

// resolve loads credentials for the company and builds an API client. When the
// integration is missing or incomplete it returns a non-nil *Result carrying an
// error payload for the model (so the bot can tell the user) and a nil client.
func (b yclientsBase) resolve(ctx context.Context, companyID uuid.UUID) (YClientsAPI, string, *Result) {
	token, meta, err := b.secrets.GetPlaintextWithMetadata(ctx, companyID, domain.SecretKindYClients)
	if err != nil {
		return nil, "", yclientsErr("Интеграция YClients не настроена. Задайте партнёрский токен и company_id в настройках интеграций.")
	}
	ycCompanyID, baseURL := parseYClientsMeta(meta)
	if strings.TrimSpace(token) == "" || ycCompanyID == "" {
		return nil, "", yclientsErr("Интеграция YClients настроена не полностью: нужны партнёрский токен и company_id.")
	}
	return b.factory(token, baseURL), ycCompanyID, nil
}

func parseYClientsMeta(raw json.RawMessage) (companyID, baseURL string) {
	if len(raw) == 0 {
		return "", ""
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return "", ""
	}
	return jsonScalarString(m["company_id"]), jsonScalarString(m["api_base"])
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

// --- yclients_get_services ---------------------------------------------------

// GetYClientsServicesTool lists services bookable via YClients.
type GetYClientsServicesTool struct{ yclientsBase }

// NewGetYClientsServicesTool executes the tools.NewGetYClientsServicesTool operation.
func NewGetYClientsServicesTool(secrets yclientsSecretReader, factory YClientsClientFactory) *GetYClientsServicesTool {
	return &GetYClientsServicesTool{yclientsBase{secrets: secrets, factory: factory}}
}

func (t *GetYClientsServicesTool) Name() string { return ToolYClientsGetServices }

func (t *GetYClientsServicesTool) Description() string {
	return "Получить список услуг для записи в YClients с их id, названием и ценой. Вызывай перед записью, чтобы узнать service_id."
}

func (t *GetYClientsServicesTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}

func (t *GetYClientsServicesTool) Execute(ctx context.Context, companyID uuid.UUID, _ json.RawMessage) (Result, error) {
	applog.TraceCall(ctx, "tools.GetYClientsServicesTool.Execute")
	api, ycCompany, errResult := t.resolve(ctx, companyID)
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
	return "Получить список специалистов (мастеров) для записи в YClients с их id и именем. Можно отфильтровать по service_id."
}

func (t *GetYClientsStaffTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"service_id": {"type": "integer", "description": "Показать только мастеров, оказывающих эту услугу (необязательно)"}
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
	api, ycCompany, errResult := t.resolve(ctx, companyID)
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
	return "Получить свободные окна для записи к мастеру на конкретную дату. Возвращает datetime в формате ISO8601 — используй его как есть при создании записи."
}

func (t *GetYClientsTimesTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"staff_id": {"type": "integer", "description": "ID мастера из yclients_get_staff"},
			"date": {"type": "string", "minLength": 10, "description": "Дата в формате YYYY-MM-DD"}
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
	api, ycCompany, errResult := t.resolve(ctx, companyID)
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
	return "Создать запись клиента в YClients. Перед вызовом подтверди у клиента услугу, мастера, дату и время, имя и телефон. service_id и staff_id бери из yclients_get_services/yclients_get_staff, datetime — из yclients_get_times."
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
			"comment": {"type": "string", "description": "Комментарий к записи (необязательно)"}
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
	api, ycCompany, errResult := t.resolve(ctx, companyID)
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
		return *yclientsErr(fmt.Sprintf("service_id %d не найден среди услуг компании — вызови yclients_get_services и используй реальный id услуги", in.ServiceID)), nil
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

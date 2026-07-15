package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/config"
	secretcrypto "github.com/knowledgeos/backend/internal/crypto"
	"github.com/knowledgeos/backend/internal/domain"
	"gorm.io/gorm"
)

type fakeBotSettingsRepo struct {
	byCompany map[uuid.UUID]*domain.BotSettings
}

func newFakeBotSettingsRepo() *fakeBotSettingsRepo {
	return &fakeBotSettingsRepo{byCompany: map[uuid.UUID]*domain.BotSettings{}}
}

func (r *fakeBotSettingsRepo) Get(_ context.Context, companyID uuid.UUID) (*domain.BotSettings, error) {
	settings, ok := r.byCompany[companyID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *settings
	return &copy, nil
}

func (r *fakeBotSettingsRepo) Upsert(_ context.Context, companyID uuid.UUID, settings *domain.BotSettings) error {
	copy := *settings
	copy.CompanyID = companyID
	r.byCompany[companyID] = &copy
	return nil
}

type fakeTenantSecretRepo struct {
	byCompany map[uuid.UUID]map[domain.SecretKind]*domain.TenantSecret
}

func newFakeTenantSecretRepo() *fakeTenantSecretRepo {
	return &fakeTenantSecretRepo{byCompany: map[uuid.UUID]map[domain.SecretKind]*domain.TenantSecret{}}
}

func (r *fakeTenantSecretRepo) List(_ context.Context, companyID uuid.UUID) ([]domain.TenantSecret, error) {
	items := r.byCompany[companyID]
	out := make([]domain.TenantSecret, 0, len(items))
	for _, item := range items {
		copy := *item
		out = append(out, copy)
	}
	return out, nil
}

func (r *fakeTenantSecretRepo) Get(_ context.Context, companyID uuid.UUID, kind domain.SecretKind) (*domain.TenantSecret, error) {
	items := r.byCompany[companyID]
	item, ok := items[kind]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copy := *item
	return &copy, nil
}

func (r *fakeTenantSecretRepo) Upsert(_ context.Context, companyID uuid.UUID, secret *domain.TenantSecret) error {
	if r.byCompany[companyID] == nil {
		r.byCompany[companyID] = map[domain.SecretKind]*domain.TenantSecret{}
	}
	copy := *secret
	copy.ID = uuid.New()
	copy.CompanyID = companyID
	copy.UpdatedAt = time.Now()
	r.byCompany[companyID][secret.Kind] = &copy
	return nil
}

func (r *fakeTenantSecretRepo) Delete(_ context.Context, companyID uuid.UUID, kind domain.SecretKind) error {
	delete(r.byCompany[companyID], kind)
	return nil
}

func boolPtr(v bool) *bool                                 { return &v }
func floatPtr(v float64) *float64                          { return &v }
func intPtr(v int) *int                                    { return &v }
func providerPtr(v domain.BotProvider) *domain.BotProvider { return &v }
func tierPtr(v domain.BotModelTier) *domain.BotModelTier   { return &v }

func TestBotSettingsServiceDefaultAndUpdate(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()
	repo := newFakeBotSettingsRepo()
	svc := NewBotSettingsService(repo)

	settings, err := svc.Get(ctx, companyID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if settings.CompanyID != companyID || settings.Provider != domain.BotProviderYandex || settings.ModelTier != domain.BotModelTierLite {
		t.Fatalf("unexpected defaults: %+v", settings)
	}
	if string(settings.EnabledModules) != "{}" {
		t.Fatalf("EnabledModules = %s, want {}", settings.EnabledModules)
	}

	model := " gpt://folder/model/latest "
	name := " Бот "
	tone := "strict"
	rules := " use sources "
	updated, err := svc.Update(ctx, companyID, UpdateBotSettingsRequest{
		Enabled:        boolPtr(true),
		Provider:       providerPtr(domain.BotProviderYandex),
		ModelTier:      tierPtr(domain.BotModelTierPro),
		Model:          &model,
		Temperature:    floatPtr(0.7),
		MaxTokens:      intPtr(2048),
		PersonaName:    &name,
		PersonaTone:    &tone,
		PersonaRules:   &rules,
		EnabledModules: json.RawMessage(`{"rag":true}`),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if !updated.Enabled || updated.ModelTier != domain.BotModelTierPro || updated.Model == nil || *updated.Model != strings.TrimSpace(model) {
		t.Fatalf("unexpected updated settings: %+v", updated)
	}
	if updated.PersonaName != "Бот" || updated.PersonaRules != "use sources" {
		t.Fatalf("persona fields were not normalized: %+v", updated)
	}
	if string(updated.EnabledModules) != `{"rag":true}` {
		t.Fatalf("EnabledModules = %s", updated.EnabledModules)
	}
}

func TestBotSettingsServiceValidation(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()
	svc := NewBotSettingsService(newFakeBotSettingsRepo())

	tests := []struct {
		name string
		req  UpdateBotSettingsRequest
	}{
		{name: "invalid provider", req: UpdateBotSettingsRequest{Provider: providerPtr("openai")}},
		{name: "invalid tier", req: UpdateBotSettingsRequest{ModelTier: tierPtr("ultra")}},
		{name: "temperature below range", req: UpdateBotSettingsRequest{Temperature: floatPtr(-0.1)}},
		{name: "temperature above range", req: UpdateBotSettingsRequest{Temperature: floatPtr(2.1)}},
		{name: "max tokens below range", req: UpdateBotSettingsRequest{MaxTokens: intPtr(0)}},
		{name: "max tokens above range", req: UpdateBotSettingsRequest{MaxTokens: intPtr(8193)}},
		{name: "empty persona name", req: UpdateBotSettingsRequest{PersonaName: ptrStr(" ")}},
		{name: "empty persona tone", req: UpdateBotSettingsRequest{PersonaTone: ptrStr(" ")}},
		{name: "modules not object", req: UpdateBotSettingsRequest{EnabledModules: json.RawMessage(`[]`)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Update(ctx, companyID, tt.req); HTTPStatus(err) != 400 {
				t.Fatalf("Update() status = %d, err = %v, want 400", HTTPStatus(err), err)
			}
		})
	}
}

func TestBotSettingsServiceClearsModelOverride(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()
	svc := NewBotSettingsService(newFakeBotSettingsRepo())

	model := "gpt://folder/custom/latest"
	if _, err := svc.Update(ctx, companyID, UpdateBotSettingsRequest{Model: &model}); err != nil {
		t.Fatalf("set model override: %v", err)
	}
	blank := " "
	updated, err := svc.Update(ctx, companyID, UpdateBotSettingsRequest{Model: &blank})
	if err != nil {
		t.Fatalf("clear model override: %v", err)
	}
	if updated.Model != nil {
		t.Fatalf("Model = %q, want nil", *updated.Model)
	}
}

func TestTenantSecretServiceStatusSetGetAndDelete(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()
	cipher, err := secretcrypto.NewCipher("test-secret-key")
	if err != nil {
		t.Fatalf("NewCipher() error = %v", err)
	}
	repo := newFakeTenantSecretRepo()
	svc := NewTenantSecretService(repo, cipher)

	statuses, err := svc.ListStatus(ctx, companyID)
	if err != nil {
		t.Fatalf("ListStatus() error = %v", err)
	}
	if len(statuses) != len(domain.SupportedSecretKinds()) {
		t.Fatalf("statuses len = %d", len(statuses))
	}
	for _, status := range statuses {
		if status.IsSet {
			t.Fatalf("status %+v should not be set", status)
		}
	}

	status, err := svc.Set(ctx, companyID, domain.SecretKindLLM, SetTenantSecretRequest{
		Value:    "iam-token",
		Metadata: json.RawMessage(`{"auth_type":"iam"}`),
	})
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if !status.IsSet || strings.Contains(string(status.Metadata), "iam-token") {
		t.Fatalf("unexpected status: %+v", status)
	}

	stored := repo.byCompany[companyID][domain.SecretKindLLM]
	if strings.Contains(string(stored.Ciphertext), "iam-token") {
		t.Fatal("ciphertext contains plaintext")
	}
	plaintext, err := svc.GetPlaintext(ctx, companyID, domain.SecretKindLLM)
	if err != nil {
		t.Fatalf("GetPlaintext() error = %v", err)
	}
	if plaintext != "iam-token" {
		t.Fatalf("GetPlaintext() = %q", plaintext)
	}

	if err := svc.Delete(ctx, companyID, domain.SecretKindLLM); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := svc.GetPlaintext(ctx, companyID, domain.SecretKindLLM); HTTPStatus(err) != 404 {
		t.Fatalf("GetPlaintext() status = %d, err = %v, want 404", HTTPStatus(err), err)
	}
}

func TestTenantSecretServiceValidationAndIsolation(t *testing.T) {
	ctx := context.Background()
	companyA := uuid.New()
	companyB := uuid.New()
	cipher, err := secretcrypto.NewCipher("test-secret-key")
	if err != nil {
		t.Fatalf("NewCipher() error = %v", err)
	}
	repo := newFakeTenantSecretRepo()
	svc := NewTenantSecretService(repo, cipher)

	tests := []struct {
		name string
		kind domain.SecretKind
		req  SetTenantSecretRequest
	}{
		{name: "invalid kind", kind: "unknown", req: SetTenantSecretRequest{Value: "x", Metadata: json.RawMessage(`{}`)}},
		{name: "empty value", kind: domain.SecretKindLLM, req: SetTenantSecretRequest{Value: " ", Metadata: json.RawMessage(`{}`)}},
		{name: "metadata not object", kind: domain.SecretKindLLM, req: SetTenantSecretRequest{Value: "x", Metadata: json.RawMessage(`[]`)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Set(ctx, companyA, tt.kind, tt.req); HTTPStatus(err) != 400 {
				t.Fatalf("Set() status = %d, err = %v, want 400", HTTPStatus(err), err)
			}
		})
	}

	if _, err := NewTenantSecretService(repo, nil).Set(ctx, companyA, domain.SecretKindLLM, SetTenantSecretRequest{Value: "x", Metadata: json.RawMessage(`{}`)}); HTTPStatus(err) != 400 {
		t.Fatalf("Set() without cipher status = %d, err = %v, want 400", HTTPStatus(err), err)
	}

	if _, err := svc.Set(ctx, companyA, domain.SecretKindLLM, SetTenantSecretRequest{Value: "company-a", Metadata: json.RawMessage(`{}`)}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if _, err := svc.GetPlaintext(ctx, companyB, domain.SecretKindLLM); HTTPStatus(err) != 404 {
		t.Fatalf("company B GetPlaintext() status = %d, err = %v, want 404", HTTPStatus(err), err)
	}
}

func TestLLMFactoryModelAndEmbeddingSelection(t *testing.T) {
	companyID := uuid.New()
	settingsSvc := NewBotSettingsService(newFakeBotSettingsRepo())
	secretsSvc := NewTenantSecretService(newFakeTenantSecretRepo(), nil)
	factory := NewLLMFactory(&config.Config{
		YandexFolderID:             "folder",
		YandexDefaultChatModelLite: "gpt://folder/lite",
		YandexDefaultChatModelPro:  "gpt://folder/pro",
		YandexEmbeddingDocModel:    "text-embeddings-v2-doc",
		YandexEmbeddingQueryModel:  "emb://other/query/latest",
	}, settingsSvc, secretsSvc)

	settings := DefaultBotSettings(companyID)
	if got := factory.chatModel(settings); got != "gpt://folder/lite" {
		t.Fatalf("lite chatModel = %q", got)
	}
	settings.ModelTier = domain.BotModelTierPro
	if got := factory.chatModel(settings); got != "gpt://folder/pro" {
		t.Fatalf("pro chatModel = %q", got)
	}
	override := " gpt://folder/custom "
	settings.Model = &override
	if got := factory.chatModel(settings); got != "gpt://folder/custom" {
		t.Fatalf("override chatModel = %q", got)
	}
	if got := factory.embeddingModel("text-embeddings-v2-doc", "fallback"); got != "emb://folder/text-embeddings-v2-doc/latest" {
		t.Fatalf("embedding model = %q", got)
	}
	if got := factory.embeddingModel("emb://folder/full/latest", "fallback"); got != "emb://folder/full/latest" {
		t.Fatalf("full embedding model = %q", got)
	}
}

func TestLLMFactoryFallbackModelsAndAuthType(t *testing.T) {
	ctx := context.Background()
	companyID := uuid.New()
	settingsRepo := newFakeBotSettingsRepo()
	secretsRepo := newFakeTenantSecretRepo()
	cipher, err := secretcrypto.NewCipher("test-secret-key")
	if err != nil {
		t.Fatalf("NewCipher() error = %v", err)
	}
	settingsSvc := NewBotSettingsService(settingsRepo)
	secretsSvc := NewTenantSecretService(secretsRepo, cipher)
	factory := NewLLMFactory(&config.Config{YandexFolderID: "folder"}, settingsSvc, secretsSvc)

	settings := DefaultBotSettings(companyID)
	if got := factory.chatModel(settings); got != "gpt://folder/yandexgpt-lite/latest" {
		t.Fatalf("fallback lite model = %q", got)
	}
	settings.ModelTier = domain.BotModelTierPro
	if got := factory.chatModel(settings); got != "gpt://folder/yandexgpt/latest" {
		t.Fatalf("fallback pro model = %q", got)
	}

	// Without a per-company LLM secret the bot is unusable: there is no global
	// env fallback, so ForCompany reports a configuration conflict (409).
	if _, _, err := factory.ForCompany(ctx, companyID); HTTPStatus(err) != 409 {
		t.Fatalf("ForCompany() without secret status = %d, err = %v, want 409", HTTPStatus(err), err)
	}
	if _, _, err := factory.credentials(ctx, companyID); HTTPStatus(err) != 409 {
		t.Fatalf("credentials() without secret status = %d, err = %v, want 409", HTTPStatus(err), err)
	}
	// Even when a global env key is configured it must NOT be used as a fallback.
	envFactory := NewLLMFactory(&config.Config{YandexFolderID: "folder", YandexAPIKey: "env-api-key"}, settingsSvc, secretsSvc)
	if _, _, err := envFactory.credentials(ctx, companyID); HTTPStatus(err) != 409 {
		t.Fatalf("credentials() must ignore env key, status = %d, err = %v, want 409", HTTPStatus(err), err)
	}
	if _, err := secretsSvc.Set(ctx, companyID, domain.SecretKindLLM, SetTenantSecretRequest{
		Value:    "api-key",
		Metadata: json.RawMessage(`{"auth_type":"api_key"}`),
	}); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	token, authType, err := factory.credentials(ctx, companyID)
	if err != nil {
		t.Fatalf("credentials() error = %v", err)
	}
	if token != "api-key" || authType != "api_key" {
		t.Fatalf("credentials() = %q/%q", token, authType)
	}
	provider, embedder, err := factory.ForCompany(ctx, companyID)
	if err != nil {
		t.Fatalf("ForCompany() error = %v", err)
	}
	if provider == nil || embedder == nil {
		t.Fatal("ForCompany() returned nil provider/embedder")
	}
}

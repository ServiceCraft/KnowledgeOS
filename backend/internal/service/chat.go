package service

import (
	"context"
	"encoding/json"
	"errors"
	applog "github.com/knowledgeos/backend/internal/logger"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/knowledgeos/backend/internal/chat/tools"
	"github.com/knowledgeos/backend/internal/domain"
	"github.com/knowledgeos/backend/internal/llm"
	"gorm.io/gorm"
)

const (
	chatHistoryLimit     = 40
	chatHistoryMaxRunes  = 12000
	chatContextMaxChunks = 8
	maxToolIterations    = 5

	chatNoAnswerFallback = "Не нашёл подходящей информации в базе знаний. Уточните вопрос или обратитесь к оператору."
)

type chatToolset interface {
	Definitions() []llm.Tool
	Execute(ctx context.Context, companyID uuid.UUID, call llm.ToolCall) (tools.Result, error)
}

type chatSettingsProvider interface {
	Get(ctx context.Context, companyID uuid.UUID) (*domain.BotSettings, error)
}

type chatRetriever interface {
	Search(ctx context.Context, companyID uuid.UUID, req domain.RetrieveRequest) (*domain.RetrieveResult, error)
}

type chatLLMFactory interface {
	ForCompany(ctx context.Context, companyID uuid.UUID) (llm.Provider, llm.Embedder, error)
}

type ChatService struct {
	chats      domain.ChatRepository
	settings   chatSettingsProvider
	retriever  chatRetriever
	llmFactory chatLLMFactory
	toolset    chatToolset
}

// NewChatService executes the service.NewChatService operation.
func NewChatService(chats domain.ChatRepository, settings chatSettingsProvider, retriever chatRetriever, llmFactory chatLLMFactory, toolset chatToolset) *ChatService {
	return &ChatService{chats: chats, settings: settings, retriever: retriever, llmFactory: llmFactory, toolset: toolset}
}

func (s *ChatService) toolDefinitions() []llm.Tool {
	if s.toolset == nil {
		return nil
	}
	return s.toolset.Definitions()
}

type CreateChatSessionRequest struct {
	Channel        domain.ChatChannel `json:"channel,omitempty"`
	ExternalChatID *string            `json:"external_chat_id,omitempty"`
	Title          string             `json:"title,omitempty"`
}

type SendChatMessageRequest struct {
	Content string `json:"content"`
}

type ChatExchange struct {
	Session *domain.ChatSession `json:"session"`
	User    *domain.ChatMessage `json:"user"`
	Message *domain.ChatMessage `json:"message"`
	Sources []domain.ChatSource `json:"sources"`
}

type ChatStreamEvent struct {
	Type    string              `json:"type"`
	Delta   string              `json:"delta,omitempty"`
	Message *domain.ChatMessage `json:"message,omitempty"`
	Sources []domain.ChatSource `json:"sources,omitempty"`
	Usage   *llm.Usage          `json:"usage,omitempty"`
	Error   string              `json:"error,omitempty"`
}

// CreateSession executes the service.ChatService.CreateSession operation.
func (s *ChatService) CreateSession(ctx context.Context, companyID uuid.UUID, req CreateChatSessionRequest) (*domain.ChatSession, error) {
	applog.TraceCall(ctx, "service.ChatService.CreateSession")
	channel := req.Channel
	if channel == "" {
		channel = domain.ChatChannelPlayground
	}
	if !validChatChannel(channel) {
		return nil, badRequest("invalid channel")
	}
	title := strings.TrimSpace(req.Title)
	session := &domain.ChatSession{
		Channel:        channel,
		ExternalChatID: req.ExternalChatID,
		State:          domain.ChatStateBot,
		Title:          title,
	}
	if err := s.chats.CreateSession(ctx, companyID, session); err != nil {
		return nil, err
	}
	return session, nil
}

// ListSessions executes the service.ChatService.ListSessions operation.
func (s *ChatService) ListSessions(ctx context.Context, companyID uuid.UUID, filter domain.ChatSessionFilter) ([]domain.ChatSession, int64, error) {
	applog.TraceCall(ctx, "service.ChatService.ListSessions")
	return s.chats.ListSessions(ctx, companyID, filter)
}

// GetSession executes the service.ChatService.GetSession operation.
func (s *ChatService) GetSession(ctx context.Context, companyID, sessionID uuid.UUID) (*domain.ChatSessionWithMessages, error) {
	applog.TraceCall(ctx, "service.ChatService.GetSession")
	session, err := s.chats.GetSession(ctx, companyID, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFound("chat session not found")
		}
		return nil, err
	}
	messages, err := s.chats.ListMessages(ctx, companyID, sessionID, 200)
	if err != nil {
		return nil, err
	}
	return &domain.ChatSessionWithMessages{Session: session, Messages: messages}, nil
}

// SendMessage executes the service.ChatService.SendMessage operation.
func (s *ChatService) SendMessage(ctx context.Context, companyID, sessionID uuid.UUID, req SendChatMessageRequest) (*ChatExchange, error) {
	applog.TraceCall(ctx, "service.ChatService.SendMessage")
	session, userMessage, history, err := s.prepareTurn(ctx, companyID, sessionID, req)
	if err != nil {
		return nil, err
	}
	assistant, sources, err := s.generate(ctx, companyID, sessionID, history, false, nil)
	if err != nil {
		return nil, err
	}
	assistant.SessionID = sessionID
	if err := s.chats.AppendMessage(ctx, companyID, assistant); err != nil {
		return nil, err
	}
	refreshed, _ := s.chats.GetSession(ctx, companyID, sessionID)
	if refreshed != nil {
		session = refreshed
	}
	return &ChatExchange{Session: session, User: userMessage, Message: assistant, Sources: sources}, nil
}

// StreamMessage executes the service.ChatService.StreamMessage operation.
func (s *ChatService) StreamMessage(ctx context.Context, companyID, sessionID uuid.UUID, req SendChatMessageRequest, emit func(ChatStreamEvent) error) error {
	applog.TraceCall(ctx, "service.ChatService.StreamMessage")
	session, _, history, err := s.prepareTurn(ctx, companyID, sessionID, req)
	if err != nil {
		return err
	}
	_ = session
	assistant, _, err := s.generate(ctx, companyID, sessionID, history, true, emit)
	if err != nil {
		return err
	}
	assistant.SessionID = sessionID
	if err := s.chats.AppendMessage(ctx, companyID, assistant); err != nil {
		return err
	}
	if emit != nil {
		if err := emit(ChatStreamEvent{Type: "message", Message: assistant}); err != nil {
			return err
		}
		return emit(ChatStreamEvent{Type: "done"})
	}
	return nil
}

func (s *ChatService) prepareTurn(ctx context.Context, companyID, sessionID uuid.UUID, req SendChatMessageRequest) (*domain.ChatSession, *domain.ChatMessage, []domain.ChatMessage, error) {
	applog.TraceCall(ctx, "service.ChatService.prepareTurn")
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, nil, nil, badRequest("content is required")
	}
	session, err := s.chats.GetSession(ctx, companyID, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, notFound("chat session not found")
		}
		return nil, nil, nil, err
	}
	if session.State != domain.ChatStateBot {
		return nil, nil, nil, conflict("chat session is not handled by bot")
	}
	userMessage := &domain.ChatMessage{
		SessionID: sessionID,
		Role:      domain.ChatRoleUser,
		Content:   content,
	}
	if err := s.chats.AppendMessage(ctx, companyID, userMessage); err != nil {
		return nil, nil, nil, err
	}
	history, err := s.chats.ListMessages(ctx, companyID, sessionID, chatHistoryLimit)
	if err != nil {
		return nil, nil, nil, err
	}
	return session, userMessage, history, nil
}

func (s *ChatService) generate(ctx context.Context, companyID, sessionID uuid.UUID, history []domain.ChatMessage, stream bool, emit func(ChatStreamEvent) error) (*domain.ChatMessage, []domain.ChatSource, error) {
	applog.TraceCall(ctx, "service.ChatService.generate")
	settings, err := s.settings.Get(ctx, companyID)
	if err != nil {
		return nil, nil, err
	}
	if !settings.Enabled {
		return nil, nil, conflict("bot is disabled")
	}
	toolDefs := s.toolDefinitions()
	hasTools := len(toolDefs) > 0

	query := latestUserContent(history)
	retrieveResult, err := s.retriever.Search(ctx, companyID, domain.RetrieveRequest{
		Query:      query,
		HybridTopK: chatContextMaxChunks,
		Rewrite:    true,
	})
	if err != nil {
		return nil, nil, err
	}
	sources := chatSourcesFromRAG(retrieveResult.Results)
	// Without tools the bot cannot recover from an empty knowledge base, so we
	// keep the early refusal. With tools the model may call search_knowledge or
	// get_pricing itself, so we still invoke the LLM.
	if len(sources) == 0 && !hasTools {
		message := assistantMessage(chatNoAnswerFallback, nil, sources, llm.Usage{})
		if stream && emit != nil {
			if err := emit(ChatStreamEvent{Type: "delta", Delta: chatNoAnswerFallback}); err != nil {
				return nil, nil, err
			}
		}
		return message, sources, nil
	}

	provider, _, err := s.llmFactory.ForCompany(ctx, companyID)
	if err != nil {
		return nil, nil, err
	}

	if !hasTools {
		req := llm.ChatRequest{
			Messages:         buildLLMMessages(settings, sources, false, history),
			GenerationParams: llm.GenerationParams{Temperature: settings.Temperature, MaxTokens: settings.MaxTokens},
		}
		if stream {
			return s.generateStream(ctx, provider, req, sources, emit)
		}
		resp, err := provider.Chat(ctx, req)
		if err != nil {
			return nil, nil, err
		}
		content := strings.TrimSpace(resp.Message.Content)
		if content == "" {
			content = chatNoAnswerFallback
		}
		return assistantMessage(content, nil, sources, resp.Usage), sources, nil
	}

	return s.runToolLoop(ctx, companyID, sessionID, settings, provider, toolDefs, sources, history, stream, emit)
}

// runToolLoop drives the bounded agent loop: it calls the LLM with the tool
// definitions, executes any requested tools, feeds the results back as tool
// messages and repeats until the model produces a final answer or the iteration
// budget is exhausted. Intermediate assistant tool-call and tool-result
// messages are persisted so multi-turn history can be reconstructed.
func (s *ChatService) runToolLoop(ctx context.Context, companyID, sessionID uuid.UUID, settings *domain.BotSettings, provider llm.Provider, toolDefs []llm.Tool, prefetch []domain.ChatSource, history []domain.ChatMessage, stream bool, emit func(ChatStreamEvent) error) (*domain.ChatMessage, []domain.ChatSource, error) {
	applog.TraceCall(ctx, "service.ChatService.runToolLoop")
	llmMessages := buildLLMMessages(settings, prefetch, true, history)
	collected := append([]domain.ChatSource{}, prefetch...)
	if stream && emit != nil {
		if err := emit(ChatStreamEvent{Type: "sources", Sources: dedupeSources(collected)}); err != nil {
			return nil, nil, err
		}
	}

	var usage llm.Usage
	for iter := 0; iter <= maxToolIterations; iter++ {
		req := llm.ChatRequest{
			Messages:         llmMessages,
			GenerationParams: llm.GenerationParams{Temperature: settings.Temperature, MaxTokens: settings.MaxTokens},
		}
		// On the final iteration tools are withheld to force a textual answer
		// and guarantee termination.
		if iter < maxToolIterations {
			req.Tools = toolDefs
		}
		resp, err := provider.Chat(ctx, req)
		if err != nil {
			return nil, nil, err
		}
		usage = addUsage(usage, resp.Usage)

		calls := resp.Message.ToolCalls
		if len(calls) == 0 {
			content := strings.TrimSpace(resp.Message.Content)
			if content == "" {
				content = chatNoAnswerFallback
			}
			finalSources := dedupeSources(collected)
			if stream && emit != nil {
				if err := emit(ChatStreamEvent{Type: "delta", Delta: content}); err != nil {
					return nil, nil, err
				}
				if usage.TotalTokens > 0 {
					u := usage
					if err := emit(ChatStreamEvent{Type: "usage", Usage: &u}); err != nil {
						return nil, nil, err
					}
				}
			}
			return assistantMessage(content, nil, finalSources, usage), finalSources, nil
		}

		callMsg := assistantToolCallMessage(sessionID, resp.Message.Content, calls)
		if err := s.chats.AppendMessage(ctx, companyID, callMsg); err != nil {
			return nil, nil, err
		}
		llmMessages = append(llmMessages, llm.Message{Role: llm.RoleAssistant, Content: resp.Message.Content, ToolCalls: calls})

		for _, call := range calls {
			content, srcs := s.executeToolCall(ctx, companyID, call)
			collected = append(collected, srcs...)
			toolMsg := toolResultMessage(sessionID, call.ID, content, srcs)
			if err := s.chats.AppendMessage(ctx, companyID, toolMsg); err != nil {
				return nil, nil, err
			}
			llmMessages = append(llmMessages, llm.Message{Role: llm.RoleTool, ToolCallID: call.ID, Content: content})
		}
	}

	// Safety net: the loop always disables tools on the last iteration, so this
	// is only reached if a provider keeps emitting tool calls regardless.
	finalSources := dedupeSources(collected)
	if stream && emit != nil {
		if err := emit(ChatStreamEvent{Type: "delta", Delta: chatNoAnswerFallback}); err != nil {
			return nil, nil, err
		}
	}
	return assistantMessage(chatNoAnswerFallback, nil, finalSources, usage), finalSources, nil
}

func (s *ChatService) executeToolCall(ctx context.Context, companyID uuid.UUID, call llm.ToolCall) (string, []domain.ChatSource) {
	applog.TraceCall(ctx, "service.ChatService.executeToolCall")
	result, err := s.toolset.Execute(ctx, companyID, call)
	if err != nil {
		applog.From(ctx).Debug().Err(err).Str("tool", call.Name).Msg("tool call rejected or failed")
		payload, _ := json.Marshal(map[string]string{"error": err.Error()})
		return string(payload), nil
	}
	content := result.Content
	if strings.TrimSpace(content) == "" {
		content = "{}"
	}
	return content, result.Sources
}

func (s *ChatService) generateStream(ctx context.Context, provider llm.Provider, req llm.ChatRequest, sources []domain.ChatSource, emit func(ChatStreamEvent) error) (*domain.ChatMessage, []domain.ChatSource, error) {
	applog.TraceCall(ctx, "service.ChatService.generateStream")
	if emit != nil {
		if err := emit(ChatStreamEvent{Type: "sources", Sources: sources}); err != nil {
			return nil, nil, err
		}
	}
	chunks, err := provider.ChatStream(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	var content strings.Builder
	var usage llm.Usage
	var toolCalls []llm.ToolCall
	for chunk := range chunks {
		if chunk.Err != nil {
			return nil, nil, chunk.Err
		}
		if chunk.Delta != "" {
			content.WriteString(chunk.Delta)
			if emit != nil {
				if err := emit(ChatStreamEvent{Type: "delta", Delta: chunk.Delta}); err != nil {
					return nil, nil, err
				}
			}
		}
		if len(chunk.ToolCalls) > 0 {
			toolCalls = append(toolCalls, chunk.ToolCalls...)
		}
		if chunk.Usage.TotalTokens > 0 {
			usage = chunk.Usage
			if emit != nil {
				usageCopy := usage
				if err := emit(ChatStreamEvent{Type: "usage", Usage: &usageCopy}); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	out := strings.TrimSpace(content.String())
	if out == "" && len(toolCalls) > 0 {
		out = "Для ответа требуется инструмент, который будет подключён на следующем этапе."
	}
	return assistantMessage(out, toolCalls, sources, usage), sources, nil
}

func assistantMessage(content string, toolCalls []llm.ToolCall, sources []domain.ChatSource, usage llm.Usage) *domain.ChatMessage {
	toolRaw, _ := json.Marshal(toolCalls)
	sourceRaw, _ := json.Marshal(sources)
	return &domain.ChatMessage{
		Role:             domain.ChatRoleAssistant,
		Content:          content,
		ToolCalls:        toolRaw,
		Sources:          sourceRaw,
		TokensPrompt:     usage.PromptTokens,
		TokensCompletion: usage.CompletionTokens,
	}
}

func assistantToolCallMessage(sessionID uuid.UUID, content string, calls []llm.ToolCall) *domain.ChatMessage {
	raw, _ := json.Marshal(calls)
	return &domain.ChatMessage{
		SessionID: sessionID,
		Role:      domain.ChatRoleAssistant,
		Content:   content,
		ToolCalls: raw,
	}
}

func toolResultMessage(sessionID uuid.UUID, toolCallID, content string, sources []domain.ChatSource) *domain.ChatMessage {
	raw, _ := json.Marshal(sources)
	return &domain.ChatMessage{
		SessionID:  sessionID,
		Role:       domain.ChatRoleTool,
		Content:    content,
		ToolCallID: toolCallID,
		Sources:    raw,
	}
}

func addUsage(a, b llm.Usage) llm.Usage {
	return llm.Usage{
		PromptTokens:     a.PromptTokens + b.PromptTokens,
		CompletionTokens: a.CompletionTokens + b.CompletionTokens,
		TotalTokens:      a.TotalTokens + b.TotalTokens,
	}
}

func dedupeSources(sources []domain.ChatSource) []domain.ChatSource {
	out := make([]domain.ChatSource, 0, len(sources))
	seen := make(map[string]bool, len(sources))
	for _, src := range sources {
		key := src.SourceID
		if key == "" {
			key = string(src.EntityType) + ":" + src.EntityID.String() + ":" + strconv.Itoa(src.ChunkIdx)
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, src)
	}
	return out
}

func decodeToolCalls(raw json.RawMessage) []llm.ToolCall {
	if len(raw) == 0 {
		return nil
	}
	var calls []llm.ToolCall
	if err := json.Unmarshal(raw, &calls); err != nil {
		return nil
	}
	return calls
}

func buildLLMMessages(settings *domain.BotSettings, sources []domain.ChatSource, hasTools bool, history []domain.ChatMessage) []llm.Message {
	messages := []llm.Message{{Role: llm.RoleSystem, Content: buildSystemPrompt(settings, sources, hasTools)}}
	// pending tracks tool_call IDs introduced by the most recent assistant
	// message, so a tool message is only kept when it follows its assistant
	// tool call. This avoids orphaned tool messages when history is trimmed.
	pending := map[string]bool{}
	for _, item := range trimChatHistory(history) {
		role := llmRole(item.Role)
		if role == "" {
			continue
		}
		msg := llm.Message{Role: role, Content: item.Content}
		switch item.Role {
		case domain.ChatRoleAssistant, domain.ChatRoleOperator:
			pending = map[string]bool{}
			if calls := decodeToolCalls(item.ToolCalls); len(calls) > 0 {
				msg.ToolCalls = calls
				for _, c := range calls {
					if c.ID != "" {
						pending[c.ID] = true
					}
				}
			}
		case domain.ChatRoleTool:
			if item.ToolCallID == "" || !pending[item.ToolCallID] {
				continue
			}
			msg.ToolCallID = item.ToolCallID
			delete(pending, item.ToolCallID)
		default:
			pending = map[string]bool{}
		}
		if strings.TrimSpace(msg.Content) == "" && len(msg.ToolCalls) == 0 && msg.ToolCallID == "" {
			continue
		}
		messages = append(messages, msg)
	}
	return messages
}

func buildSystemPrompt(settings *domain.BotSettings, sources []domain.ChatSource, hasTools bool) string {
	var b strings.Builder
	b.WriteString("Ты — ")
	b.WriteString(settings.PersonaName)
	b.WriteString(", администратор колл-центра. Отвечай в тоне: ")
	b.WriteString(settings.PersonaTone)
	b.WriteString(". Используй только данные из блока CONTEXT. Если в CONTEXT нет ответа, скажи, что информации нет в базе знаний.\n")
	if hasTools {
		b.WriteString("Если данных в CONTEXT недостаточно, вызывай инструменты (search_knowledge, get_pricing, get_service_info) и используй их результаты как дополнительный контекст. Не придумывай данные вне CONTEXT и результатов инструментов.\n")
	}
	if strings.TrimSpace(settings.PersonaRules) != "" {
		b.WriteString("Правила персоны:\n")
		b.WriteString(settings.PersonaRules)
		b.WriteString("\n")
	}
	b.WriteString("CONTEXT:\n")
	for _, src := range sources {
		b.WriteString("[")
		b.WriteString(src.SourceID)
		b.WriteString("] ")
		b.WriteString(src.Title)
		b.WriteString("\n")
		b.WriteString(src.Content)
		b.WriteString("\n---\n")
	}
	return b.String()
}

func trimChatHistory(history []domain.ChatMessage) []domain.ChatMessage {
	total := 0
	out := make([]domain.ChatMessage, 0, len(history))
	for i := len(history) - 1; i >= 0; i-- {
		item := history[i]
		total += len([]rune(item.Content))
		if total > chatHistoryMaxRunes && len(out) > 0 {
			break
		}
		out = append(out, item)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func latestUserContent(history []domain.ChatMessage) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == domain.ChatRoleUser {
			return history[i].Content
		}
	}
	return ""
}

func chatSourcesFromRAG(items []domain.RAGCandidate) []domain.ChatSource {
	if len(items) > chatContextMaxChunks {
		items = items[:chatContextMaxChunks]
	}
	out := make([]domain.ChatSource, 0, len(items))
	for _, item := range items {
		out = append(out, domain.ChatSource{
			SourceID:   item.SourceID,
			EntityType: item.EntityType,
			EntityID:   item.EntityID,
			ChunkIdx:   item.ChunkIdx,
			Title:      item.Title,
			Content:    item.Content,
			Snippet:    item.Snippet,
			Score:      item.Score,
		})
	}
	return out
}

func llmRole(role domain.ChatRole) llm.Role {
	switch role {
	case domain.ChatRoleUser:
		return llm.RoleUser
	case domain.ChatRoleAssistant, domain.ChatRoleOperator:
		return llm.RoleAssistant
	case domain.ChatRoleTool:
		return llm.RoleTool
	default:
		return ""
	}
}

func validChatChannel(channel domain.ChatChannel) bool {
	switch channel {
	case domain.ChatChannelPlayground, domain.ChatChannelAPI, domain.ChatChannelTelegram, domain.ChatChannelMAX, domain.ChatChannelVK:
		return true
	default:
		return false
	}
}

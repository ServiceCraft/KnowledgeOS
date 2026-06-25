package tools

import "github.com/knowledgeos/backend/internal/llm"

// Registry is the whitelist of tools available to the bot. Only registered
// tools can be advertised to or invoked by the LLM.
type Registry struct {
	tools map[string]Tool
	order []string
}

// NewRegistry builds a registry from the given tools, preserving registration
// order for deterministic LLM tool definitions.
func NewRegistry(tools ...Tool) *Registry {
	r := &Registry{tools: make(map[string]Tool, len(tools))}
	for _, t := range tools {
		r.Register(t)
	}
	return r
}

// Register adds a tool to the whitelist. Nil or unnamed tools are ignored.
func (r *Registry) Register(t Tool) {
	if t == nil || t.Name() == "" {
		return
	}
	if _, exists := r.tools[t.Name()]; !exists {
		r.order = append(r.order, t.Name())
	}
	r.tools[t.Name()] = t
}

// Get returns the tool registered under name, if any.
func (r *Registry) Get(name string) (Tool, bool) {
	if r == nil {
		return nil, false
	}
	t, ok := r.tools[name]
	return t, ok
}

// Definitions returns the tool descriptors to advertise in an LLM ChatRequest.
func (r *Registry) Definitions() []llm.Tool {
	if r == nil {
		return nil
	}
	out := make([]llm.Tool, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		out = append(out, llm.Tool{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Schema(),
		})
	}
	return out
}

// Len reports how many tools are registered.
func (r *Registry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.order)
}

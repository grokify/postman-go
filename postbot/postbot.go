// Package postbot provides a high-level client for Postman's Postbot API.
//
// Postbot can generate AI agent tool code from a public collection request in
// Postman's Public API Network. Access requires no special plan, but the
// endpoint is deprecated and rate-limited to 300 calls every 3 hours.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	result, _ := client.Postbot().GenerateTool(ctx, &postbot.GenerateToolInput{
//		RequestID:    "req-id",
//		CollectionID: "collection-id",
//		Language:     postbot.LanguagePython,
//		AgentFramework: postbot.AgentFrameworkOpenAI,
//	})
package postbot

import (
	"context"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Language is a programming language supported for generated tool code.
type Language string

// Supported Language values.
const (
	LanguageJavaScript Language = "javascript"
	LanguageTypeScript Language = "typescript"
	LanguagePython     Language = "python"
)

// AgentFramework is an AI agent framework supported for generated tool code.
type AgentFramework string

// Supported AgentFramework values. Note that AgentFrameworkAutogen only
// supports LanguagePython.
const (
	AgentFrameworkOpenAI    AgentFramework = "openai"
	AgentFrameworkMistral   AgentFramework = "mistral"
	AgentFrameworkGemini    AgentFramework = "gemini"
	AgentFrameworkAnthropic AgentFramework = "anthropic"
	AgentFrameworkLangChain AgentFramework = "langchain"
	AgentFrameworkAutogen   AgentFramework = "autogen"
)

// Service is the high-level Postbot client. Obtain one via
// postman.Client.Postbot.
type Service struct {
	api *api.Client
}

// New creates a Postbot service over the given generated API client. Most
// callers should use postman.Client.Postbot instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- GenerateTool ------------------------------------------------------------

// GenerateToolInput holds the fields for generating an AI agent tool.
type GenerateToolInput struct {
	// RequestID is the public collection's request ID.
	RequestID string
	// CollectionID is the Public API Network collection's ID.
	CollectionID string
	// Language is the programming language to use for the generated code.
	Language Language
	// AgentFramework is the AI agent framework to use for the generated code.
	AgentFramework AgentFramework
}

// GenerateToolResult holds the generated tool code.
type GenerateToolResult struct {
	// Text is the generated tool code.
	Text string
}

// GenerateTool generates code for an AI agent tool using a collection and
// request from the Public API Network.
//
// Deprecated: this endpoint is deprecated upstream by Postman. It only
// supports public Postman collections and requests, and is rate-limited to
// 300 calls every 3 hours; this does not accrue Postbot usage.
func (s *Service) GenerateTool(ctx context.Context, in *GenerateToolInput) (*GenerateToolResult, error) {
	if in == nil {
		in = &GenerateToolInput{}
	}
	req := &api.GenerateTool{
		RequestId:    in.RequestID,
		CollectionId: in.CollectionID,
		Config: api.GenerateToolConfig{
			Language:       api.ConfigLanguage(in.Language),
			AgentFramework: api.AgentFramework(in.AgentFramework),
		},
	}

	res, err := s.api.GenerateTool(ctx, req)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GenerateToolResponse:
		out := &GenerateToolResult{}
		if data, ok := r.Data.Get(); ok {
			out.Text = data.Text.Or("")
		}
		return out, nil
	case *api.GenerateToolBadRequestResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusTooManyRequests)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- error helpers ------------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}

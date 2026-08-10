package postman_test

import (
	"errors"
	"testing"

	postman "github.com/grokify/postman-go"
)

func TestNewClientNoAPIKey(t *testing.T) {
	t.Setenv(postman.EnvAPIKey, "")
	_, err := postman.NewClient()
	if !errors.Is(err, postman.ErrNoAPIKey) {
		t.Fatalf("err = %v, want ErrNoAPIKey", err)
	}
}

func TestNewClientWithAPIKey(t *testing.T) {
	c, err := postman.NewClient(postman.WithAPIKey("PMAK-test"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.BaseURL() != postman.DefaultBaseURL {
		t.Errorf("BaseURL() = %q, want %q", c.BaseURL(), postman.DefaultBaseURL)
	}

	services := map[string]bool{
		"Analytics":            c.Analytics() != nil,
		"APISecurity":          c.APISecurity() != nil,
		"AuditLogs":            c.AuditLogs() != nil,
		"Billing":              c.Billing() != nil,
		"CollectionAccessKeys": c.CollectionAccessKeys() != nil,
		"CollectionFolders":    c.CollectionFolders() != nil,
		"CollectionItems":      c.CollectionItems() != nil,
		"CollectionRequests":   c.CollectionRequests() != nil,
		"CollectionResponses":  c.CollectionResponses() != nil,
		"Collections":          c.Collections() != nil,
		"Comments":             c.Comments() != nil,
		"Components":           c.Components() != nil,
		"Environments":         c.Environments() != nil,
		"Groups":               c.Groups() != nil,
		"Imports":              c.Imports() != nil,
		"Mocks":                c.Mocks() != nil,
		"Monitors":             c.Monitors() != nil,
		"OAuth2":               c.OAuth2() != nil,
		"Postbot":              c.Postbot() != nil,
		"PrivateAPINetwork":    c.PrivateAPINetwork() != nil,
		"PullRequests":         c.PullRequests() != nil,
		"SDKGen":               c.SDKGen() != nil,
		"Search":               c.Search() != nil,
		"SecretScanner":        c.SecretScanner() != nil,
		"ServiceAccounts":      c.ServiceAccounts() != nil,
		"Specs":                c.Specs() != nil,
		"Tags":                 c.Tags() != nil,
		"Teams":                c.Teams() != nil,
		"Users":                c.Users() != nil,
		"Webhooks":             c.Webhooks() != nil,
		"Workspaces":           c.Workspaces() != nil,
	}
	for name, ok := range services {
		if !ok {
			t.Errorf("%s() is nil", name)
		}
	}
	if got := len(services); got != 31 {
		t.Errorf("checked %d services, want 31", got)
	}
}

func TestNewClientFromEnv(t *testing.T) {
	t.Setenv(postman.EnvAPIKey, "PMAK-env")
	c, err := postman.NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c == nil {
		t.Fatal("client is nil")
	}
}

func TestWithBaseURL(t *testing.T) {
	c, err := postman.NewClient(postman.WithAPIKey("k"), postman.WithBaseURL(postman.EUBaseURL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.BaseURL() != postman.EUBaseURL {
		t.Errorf("BaseURL() = %q, want %q", c.BaseURL(), postman.EUBaseURL)
	}
}

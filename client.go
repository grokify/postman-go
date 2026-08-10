// Package postman provides a Go client for the Postman API.
//
// It wraps an internal ogen-generated API client with a higher-level, hand-
// written surface that handles authentication and exposes convenient,
// domain-oriented services covering the Postman API.
//
// Example:
//
//	client, err := postman.NewClient(postman.WithAPIKey("PMAK-..."))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	types, err := client.SecretScanner().SecretTypes(ctx)
package postman

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/grokify/postman-go/analytics"
	"github.com/grokify/postman-go/apisecurity"
	"github.com/grokify/postman-go/auditlogs"
	"github.com/grokify/postman-go/billing"
	"github.com/grokify/postman-go/collectionaccesskeys"
	"github.com/grokify/postman-go/collectionfolders"
	"github.com/grokify/postman-go/collectionitems"
	"github.com/grokify/postman-go/collectionrequests"
	"github.com/grokify/postman-go/collectionresponses"
	"github.com/grokify/postman-go/collections"
	"github.com/grokify/postman-go/comments"
	"github.com/grokify/postman-go/components"
	"github.com/grokify/postman-go/environments"
	"github.com/grokify/postman-go/groups"
	"github.com/grokify/postman-go/imports"
	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/mocks"
	"github.com/grokify/postman-go/monitors"
	"github.com/grokify/postman-go/oauth2"
	"github.com/grokify/postman-go/postbot"
	"github.com/grokify/postman-go/postmanerr"
	"github.com/grokify/postman-go/privateapinetwork"
	"github.com/grokify/postman-go/pullrequests"
	"github.com/grokify/postman-go/sdkgen"
	"github.com/grokify/postman-go/search"
	"github.com/grokify/postman-go/secretscanner"
	"github.com/grokify/postman-go/serviceaccounts"
	"github.com/grokify/postman-go/specs"
	"github.com/grokify/postman-go/tags"
	"github.com/grokify/postman-go/teams"
	"github.com/grokify/postman-go/users"
	"github.com/grokify/postman-go/webhooks"
	"github.com/grokify/postman-go/workspaces"
)

// Version is the SDK version.
const Version = "0.1.0"

const (
	// DefaultBaseURL is the default (US region) Postman API base URL.
	DefaultBaseURL = "https://api.postman.com"
	// EUBaseURL is the EU region Postman API base URL.
	EUBaseURL = "https://api.eu.postman.com"

	// EnvAPIKey is the environment variable read for the API key when one is not
	// supplied via WithAPIKey.
	EnvAPIKey = "POSTMAN_API_KEY" //nolint:gosec // G101: This is an environment variable name, not a credential
)

// APIError is an error returned by the Postman API. It is an alias for
// postmanerr.APIError so callers can type-assert against a single type
// regardless of which service produced the error.
type APIError = postmanerr.APIError

// ErrNoAPIKey is returned by NewClient when no API key is configured.
var ErrNoAPIKey = errors.New("postman: API key is required")

// Client is the top-level Postman API client.
type Client struct {
	apiClient *api.Client
	apiKey    string
	baseURL   string

	analyticsSvc            *analytics.Service
	apiSecuritySvc          *apisecurity.Service
	auditLogsSvc            *auditlogs.Service
	billingSvc              *billing.Service
	collectionAccessKeysSvc *collectionaccesskeys.Service
	collectionFoldersSvc    *collectionfolders.Service
	collectionItemsSvc      *collectionitems.Service
	collectionRequestsSvc   *collectionrequests.Service
	collectionResponsesSvc  *collectionresponses.Service
	collectionsSvc          *collections.Service
	commentsSvc             *comments.Service
	componentsSvc           *components.Service
	environmentsSvc         *environments.Service
	groupsSvc               *groups.Service
	importsSvc              *imports.Service
	mocksSvc                *mocks.Service
	monitorsSvc             *monitors.Service
	oauth2Svc               *oauth2.Service
	postbotSvc              *postbot.Service
	privateAPINetworkSvc    *privateapinetwork.Service
	pullRequestsSvc         *pullrequests.Service
	sdkGenSvc               *sdkgen.Service
	searchSvc               *search.Service
	secretScannerSvc        *secretscanner.Service
	serviceAccountsSvc      *serviceaccounts.Service
	specsSvc                *specs.Service
	tagsSvc                 *tags.Service
	teamsSvc                *teams.Service
	usersSvc                *users.Service
	webhooksSvc             *webhooks.Service
	workspacesSvc           *workspaces.Service
}

// NewClient creates a new Postman client. The API key may be supplied via
// WithAPIKey or the POSTMAN_API_KEY environment variable.
func NewClient(opts ...Option) (*Client, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	if options.apiKey == "" {
		options.apiKey = os.Getenv(EnvAPIKey)
	}
	if options.apiKey == "" {
		return nil, ErrNoAPIKey
	}

	httpClient := options.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: options.timeout}
	}
	// Wrap the transport to add SDK identification headers.
	base := httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	httpClient.Transport = &userAgentTransport{base: base}

	apiClient, err := api.NewClient(
		options.baseURL,
		staticSecuritySource{apiKey: options.apiKey},
		api.WithClient(httpClient),
	)
	if err != nil {
		return nil, err
	}

	c := &Client{
		apiClient: apiClient,
		apiKey:    options.apiKey,
		baseURL:   options.baseURL,

		analyticsSvc:            analytics.New(apiClient),
		apiSecuritySvc:          apisecurity.New(apiClient),
		auditLogsSvc:            auditlogs.New(apiClient),
		billingSvc:              billing.New(apiClient),
		collectionAccessKeysSvc: collectionaccesskeys.New(apiClient),
		collectionFoldersSvc:    collectionfolders.New(apiClient),
		collectionItemsSvc:      collectionitems.New(apiClient),
		collectionRequestsSvc:   collectionrequests.New(apiClient),
		collectionResponsesSvc:  collectionresponses.New(apiClient),
		collectionsSvc:          collections.New(apiClient),
		commentsSvc:             comments.New(apiClient),
		componentsSvc:           components.New(apiClient),
		environmentsSvc:         environments.New(apiClient),
		groupsSvc:               groups.New(apiClient),
		importsSvc:              imports.New(apiClient),
		mocksSvc:                mocks.New(apiClient),
		monitorsSvc:             monitors.New(apiClient),
		oauth2Svc:               oauth2.New(apiClient),
		postbotSvc:              postbot.New(apiClient),
		privateAPINetworkSvc:    privateapinetwork.New(apiClient),
		pullRequestsSvc:         pullrequests.New(apiClient),
		sdkGenSvc:               sdkgen.New(apiClient),
		searchSvc:               search.New(apiClient),
		secretScannerSvc:        secretscanner.New(apiClient),
		serviceAccountsSvc:      serviceaccounts.New(apiClient),
		specsSvc:                specs.New(apiClient),
		tagsSvc:                 tags.New(apiClient),
		teamsSvc:                teams.New(apiClient),
		usersSvc:                users.New(apiClient),
		webhooksSvc:             webhooks.New(apiClient),
		workspacesSvc:           workspaces.New(apiClient),
	}
	return c, nil
}

// Analytics returns the Analytics service.
func (c *Client) Analytics() *analytics.Service { return c.analyticsSvc }

// APISecurity returns the API Security validation service.
func (c *Client) APISecurity() *apisecurity.Service { return c.apiSecuritySvc }

// AuditLogs returns the Audit Logs service.
func (c *Client) AuditLogs() *auditlogs.Service { return c.auditLogsSvc }

// Billing returns the Billing service.
func (c *Client) Billing() *billing.Service { return c.billingSvc }

// CollectionAccessKeys returns the Collection Access Keys service.
func (c *Client) CollectionAccessKeys() *collectionaccesskeys.Service {
	return c.collectionAccessKeysSvc
}

// CollectionFolders returns the Collection Folders comments service.
func (c *Client) CollectionFolders() *collectionfolders.Service { return c.collectionFoldersSvc }

// CollectionItems returns the Collection Items (folders/requests/responses) service.
func (c *Client) CollectionItems() *collectionitems.Service { return c.collectionItemsSvc }

// CollectionRequests returns the Collection Requests comments service.
func (c *Client) CollectionRequests() *collectionrequests.Service { return c.collectionRequestsSvc }

// CollectionResponses returns the Collection Responses comments service.
func (c *Client) CollectionResponses() *collectionresponses.Service {
	return c.collectionResponsesSvc
}

// Collections returns the Collections service.
func (c *Client) Collections() *collections.Service { return c.collectionsSvc }

// Comments returns the Comments service.
func (c *Client) Comments() *comments.Service { return c.commentsSvc }

// Components returns the API Spec Hub Components service.
func (c *Client) Components() *components.Service { return c.componentsSvc }

// Environments returns the Environments service.
func (c *Client) Environments() *environments.Service { return c.environmentsSvc }

// Groups returns the Groups service.
func (c *Client) Groups() *groups.Service { return c.groupsSvc }

// Imports returns the Import (OpenAPI definition import) service.
func (c *Client) Imports() *imports.Service { return c.importsSvc }

// Mocks returns the Mock Servers service.
func (c *Client) Mocks() *mocks.Service { return c.mocksSvc }

// Monitors returns the Monitors service.
func (c *Client) Monitors() *monitors.Service { return c.monitorsSvc }

// OAuth2 returns the OAuth 2.0 token service.
func (c *Client) OAuth2() *oauth2.Service { return c.oauth2Svc }

// Postbot returns the Postbot AI test-generation service.
func (c *Client) Postbot() *postbot.Service { return c.postbotSvc }

// PrivateAPINetwork returns the Private API Network service.
func (c *Client) PrivateAPINetwork() *privateapinetwork.Service { return c.privateAPINetworkSvc }

// PullRequests returns the Pull Requests service.
func (c *Client) PullRequests() *pullrequests.Service { return c.pullRequestsSvc }

// SDKGen returns the SDK Generation service.
func (c *Client) SDKGen() *sdkgen.Service { return c.sdkGenSvc }

// Search returns the Search service.
func (c *Client) Search() *search.Service { return c.searchSvc }

// SecretScanner returns the Secret Scanner service.
func (c *Client) SecretScanner() *secretscanner.Service { return c.secretScannerSvc }

// ServiceAccounts returns the Service Accounts service.
func (c *Client) ServiceAccounts() *serviceaccounts.Service { return c.serviceAccountsSvc }

// Specs returns the API Spec Hub service.
func (c *Client) Specs() *specs.Service { return c.specsSvc }

// Tags returns the Tags service.
func (c *Client) Tags() *tags.Service { return c.tagsSvc }

// Teams returns the Teams service.
func (c *Client) Teams() *teams.Service { return c.teamsSvc }

// Users returns the Users service.
func (c *Client) Users() *users.Service { return c.usersSvc }

// Webhooks returns the Webhooks service.
func (c *Client) Webhooks() *webhooks.Service { return c.webhooksSvc }

// Workspaces returns the Workspaces service.
func (c *Client) Workspaces() *workspaces.Service { return c.workspacesSvc }

// API returns the underlying ogen-generated client for advanced use cases not
// covered by the high-level services.
func (c *Client) API() *api.Client {
	return c.apiClient
}

// BaseURL returns the base URL the client is configured to use.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// staticSecuritySource supplies a fixed API key to the generated client.
type staticSecuritySource struct {
	apiKey string
}

func (s staticSecuritySource) ApiKey(_ context.Context, _ api.OperationName) (api.ApiKey, error) {
	return api.ApiKey{APIKey: s.apiKey}, nil
}

// userAgentTransport adds SDK identification headers to each request.
type userAgentTransport struct {
	base http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", "postman-go/"+Version)
	return t.base.RoundTrip(req)
}

// --- options ----------------------------------------------------------------

type clientOptions struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

func defaultOptions() *clientOptions {
	return &clientOptions{
		baseURL: DefaultBaseURL,
		timeout: 30 * time.Second,
	}
}

// Option configures a Client.
type Option func(*clientOptions)

// WithAPIKey sets the Postman API key used for authentication.
func WithAPIKey(apiKey string) Option {
	return func(o *clientOptions) { o.apiKey = apiKey }
}

// WithBaseURL sets a custom API base URL (e.g. EUBaseURL).
func WithBaseURL(baseURL string) Option {
	return func(o *clientOptions) { o.baseURL = baseURL }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(o *clientOptions) { o.httpClient = client }
}

// WithTimeout sets the request timeout used when no custom HTTP client is set.
func WithTimeout(timeout time.Duration) Option {
	return func(o *clientOptions) { o.timeout = timeout }
}

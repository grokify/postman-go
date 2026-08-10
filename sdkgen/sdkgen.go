// Package sdkgen provides a high-level client for Postman's SDK Generation
// API.
//
// This is Postman's feature for generating client SDKs (in various
// languages) from a Postman Collection or API specification, optionally
// keeping the generated SDK's Git repository in sync via a Git connection.
// It is unrelated to this Go module, which is itself a hand-written SDK for
// the Postman API.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	sdks, _ := client.SDKGen().List(ctx, &sdkgen.ListInput{WorkspaceID: wsID})
//	err := client.SDKGen().Generate(ctx, &sdkgen.GenerateInput{
//		Source:   sdkgen.Source{Type: sdkgen.SourceTypeCollection, ID: collectionID},
//		Language: sdkgen.LanguageGo,
//	})
package sdkgen

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Language is an SDK's programming language.
type Language string

// Language values.
const (
	LanguageTypescript Language = "typescript"
	LanguagePython     Language = "python"
	LanguageGo         Language = "go"
	LanguageJava       Language = "java"
	LanguageCsharp     Language = "csharp"
	LanguageRuby       Language = "ruby"
	LanguagePhp        Language = "php"
	LanguageKotlin     Language = "kotlin"
	LanguageRust       Language = "rust"
	LanguageCli        Language = "cli"
)

// BuildStatus is the state of an SDK's asynchronous generation job.
type BuildStatus string

// Build status values.
const (
	BuildStatusQueued     BuildStatus = "queued"
	BuildStatusInProgress BuildStatus = "in_progress"
	BuildStatusSucceeded  BuildStatus = "succeeded"
	BuildStatusFailed     BuildStatus = "failed"
)

// SourceType is the kind of Postman element an SDK (or Git connection) was
// generated from.
type SourceType string

// Source type values.
const (
	SourceTypeCollection SourceType = "collection"
	SourceTypeSpec       SourceType = "spec"
)

// GitConnectionStatus is the lifecycle status of an SDK's Git connection.
type GitConnectionStatus string

// Git connection status values.
const (
	// GitConnectionStatusActive connects (or reconnects) the repository; auto-update pull requests resume.
	GitConnectionStatusActive GitConnectionStatus = "active"
	// GitConnectionStatusDisconnected disconnects the repository; no further auto-update pull requests are opened.
	GitConnectionStatusDisconnected GitConnectionStatus = "disconnected"
	// GitConnectionStatusInaccessible is system-determined and can't be set via UpdateGitConnection.
	GitConnectionStatusInaccessible GitConnectionStatus = "inaccessible"
)

// PRStatus is the status of an SDK-update pull request.
type PRStatus string

// Pull request status values.
const (
	PRStatusOpen   PRStatus = "open"
	PRStatusMerged PRStatus = "merged"
	PRStatusClosed PRStatus = "closed"
)

// Source identifies the Postman Collection or specification an SDK (or Git
// connection) is generated from.
type Source struct {
	Type SourceType
	ID   string
}

// Author is a listed author of a generated SDK.
type Author struct {
	Name  string
	Email string
}

// RetryOptions configures the generated SDK's built-in HTTP retry behavior.
type RetryOptions struct {
	Enabled          bool
	MaxAttempts      int
	RetryDelay       int
	MaxDelay         float64
	BackOffFactor    float64
	RetryDelayJitter float64
	HTTPCodesToRetry []int
	// HTTPMethodsToRetry lists the HTTP methods eligible for retry (e.g.
	// "GET", "POST").
	HTTPMethodsToRetry []string
}

// TypescriptOptions configures a TypeScript SDK's package metadata.
type TypescriptOptions struct {
	NpmOrg  string
	NpmName string
}

// PythonOptions configures a Python SDK's package metadata.
type PythonOptions struct {
	PypiPackageName string
}

// GoOptions configures a Go SDK's module metadata.
type GoOptions struct {
	ModuleName string
}

// JavaOptions configures a Java SDK's package metadata.
type JavaOptions struct {
	GroupID    string
	ArtifactID string
}

// CsharpOptions configures a C# SDK's package metadata.
type CsharpOptions struct {
	PackageID string
}

// RubyOptions configures a Ruby SDK's gem metadata.
type RubyOptions struct {
	GemName string
}

// PhpOptions configures a PHP SDK's package metadata.
type PhpOptions struct {
	PackageName string
}

// KotlinOptions configures a Kotlin SDK's package metadata.
type KotlinOptions struct {
	GroupID    string
	ArtifactID string
}

// RustOptions configures a Rust SDK's crate metadata.
type RustOptions struct {
	PackageName string
}

// CliOptions configures a CLI SDK's module metadata.
type CliOptions struct {
	ModuleName string
}

// BuildError describes why an SDK's generation job failed.
type BuildError struct {
	Code    string
	Message string
}

// PullRequestRef is a lightweight reference to an SDK-update pull request.
type PullRequestRef struct {
	URL    string
	Status string
	SDKID  string
}

// SDK describes a generated SDK, including its build status.
type SDK struct {
	ID          string
	Language    Language
	Source      Source
	WorkspaceID string
	Version     string
	BuildStatus BuildStatus
	// Error is set when BuildStatus is BuildStatusFailed.
	Error       *BuildError
	PullRequest *PullRequestRef
	CreatedAt   string
	UpdatedAt   string
}

// Service is the high-level SDK Generation client. Obtain one via
// postman.Client.SDKGen.
type Service struct {
	api *api.Client
}

// New creates an SDK Generation service over the given generated API client.
// Most callers should use postman.Client.SDKGen instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- List --------------------------------------------------------------

// ListInput holds the filters and pagination options for List.
type ListInput struct {
	// WorkspaceID is the workspace that contains the SDKs. Required.
	WorkspaceID string
	// SDKIDs, if set, returns only these SDKs by ID and ignores all other
	// filters.
	SDKIDs []string
	// BuildStatus filters results by build status.
	BuildStatus BuildStatus
	// Language filters results by SDK language.
	Language Language
	// SourceID filters results by the originating collection or spec ID.
	SourceID string
	// Cursor is the pagination cursor (use ListResult.NextCursor to page).
	Cursor string
	// Limit is the maximum number of rows to return (up to 25).
	Limit int
}

// ListResult is the paginated result of List.
type ListResult struct {
	SDKs       []SDK
	NextCursor string
	Total      int
}

// List lists all SDKs the authenticated user has access to.
func (s *Service) List(ctx context.Context, in *ListInput) (*ListResult, error) {
	if in == nil {
		in = &ListInput{}
	}
	params := api.GetSdksParams{WorkspaceId: in.WorkspaceID}
	if len(in.SDKIDs) > 0 {
		params.SdkIds = in.SDKIDs
	}
	if in.BuildStatus != "" {
		params.BuildStatus = api.NewOptSdkBuildStatus(api.SdkBuildStatus(in.BuildStatus))
	}
	if in.Language != "" {
		params.Language = api.NewOptSdkLanguage(api.SdkLanguage(in.Language))
	}
	if in.SourceID != "" {
		params.SourceId = api.NewOptString(in.SourceID)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}

	res, err := s.api.GetSdks(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SdkList:
		out := &ListResult{Total: rawJSONInt([]byte(r.Meta.Total))}
		out.NextCursor = r.Meta.NextCursor.Or("")
		for _, d := range r.Data {
			out.SDKs = append(out.SDKs, sdkFromAPI(d))
		}
		return out, nil
	case *api.SdkError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusTooManyRequests)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func sdkFromAPI(d api.Sdk) SDK {
	out := SDK{
		ID:          d.ID,
		Language:    Language(d.Language),
		Source:      Source{Type: SourceType(d.Source.Type), ID: d.Source.ID},
		WorkspaceID: d.WorkspaceId,
		Version:     d.Version.Or(""),
		BuildStatus: BuildStatus(d.BuildStatus),
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
	if e, ok := d.Error.Get(); ok {
		out.Error = &BuildError{Code: e.Code, Message: e.Message}
	}
	if pr, ok := d.PullRequest.Get(); ok {
		out.PullRequest = &PullRequestRef{URL: pr.URL, Status: string(pr.Status), SDKID: pr.SdkId}
	}
	return out
}

// --- Generate --------------------------------------------------------------

// GenerateInput holds the fields for Generate. Only the options struct
// matching Language is used; the rest are ignored.
type GenerateInput struct {
	// Source is the collection or specification to generate the SDK from.
	Source Source
	// Language is the SDK's target language.
	Language Language
	// Version is the SDK's semantic version.
	Version string
	Authors []Author
	Retry   *RetryOptions

	TypescriptOptions *TypescriptOptions
	PythonOptions     *PythonOptions
	GoOptions         *GoOptions
	JavaOptions       *JavaOptions
	CsharpOptions     *CsharpOptions
	RubyOptions       *RubyOptions
	PhpOptions        *PhpOptions
	KotlinOptions     *KotlinOptions
	RustOptions       *RustOptions
	CliOptions        *CliOptions
}

// Generate starts an asynchronous generation job for a single SDK (in one
// language) from a collection or specification. Track progress with
// Status; when BuildStatus is BuildStatusSucceeded, the SDK is ready to
// download with Download.
func (s *Service) Generate(ctx context.Context, in *GenerateInput) error {
	if in == nil {
		in = &GenerateInput{}
	}
	req := &api.CreateSdk{
		Source:   api.SdkSource{Type: api.ElementType2(in.Source.Type), ID: in.Source.ID},
		Language: api.SdkLanguage(in.Language),
	}
	if in.Version != "" {
		req.SdkVersion = api.NewOptString(in.Version)
	}
	for _, a := range in.Authors {
		req.Authors = append(req.Authors, api.SdkAuthorData{Name: a.Name, Email: api.NewOptString(a.Email)})
	}
	if in.Retry != nil {
		req.Retry = api.NewOptSdkRetryOptions(retryOptionsToAPI(in.Retry))
	}
	if in.TypescriptOptions != nil {
		req.TypescriptOptions = api.NewOptTypescriptOptions(api.TypescriptOptions{
			NpmOrg:  api.NewOptString(in.TypescriptOptions.NpmOrg),
			NpmName: api.NewOptString(in.TypescriptOptions.NpmName),
		})
	}
	if in.PythonOptions != nil {
		req.PythonOptions = api.NewOptPythonOptions(api.PythonOptions{
			PypiPackageName: api.NewOptString(in.PythonOptions.PypiPackageName),
		})
	}
	if in.GoOptions != nil {
		req.GoOptions = api.NewOptGoOptions(api.GoOptions{
			GoModuleName: api.NewOptString(in.GoOptions.ModuleName),
		})
	}
	if in.JavaOptions != nil {
		req.JavaOptions = api.NewOptJavaOptions(api.JavaOptions{
			GroupId:    api.NewOptString(in.JavaOptions.GroupID),
			ArtifactId: api.NewOptString(in.JavaOptions.ArtifactID),
		})
	}
	if in.CsharpOptions != nil {
		req.CsharpOptions = api.NewOptCsharpOptions(api.CsharpOptions{
			PackageId: api.NewOptString(in.CsharpOptions.PackageID),
		})
	}
	if in.RubyOptions != nil {
		req.RubyOptions = api.NewOptRubyOptions(api.RubyOptions{
			GemName: api.NewOptString(in.RubyOptions.GemName),
		})
	}
	if in.PhpOptions != nil {
		req.PhpOptions = api.NewOptPhpOptions(api.PhpOptions{
			PackageName: api.NewOptString(in.PhpOptions.PackageName),
		})
	}
	if in.KotlinOptions != nil {
		req.KotlinOptions = api.NewOptKotlinOptions(api.KotlinOptions{
			GroupId:    api.NewOptString(in.KotlinOptions.GroupID),
			ArtifactId: api.NewOptString(in.KotlinOptions.ArtifactID),
		})
	}
	if in.RustOptions != nil {
		req.RustOptions = api.NewOptRustOptions(api.RustOptions{
			PackageName: api.NewOptString(in.RustOptions.PackageName),
		})
	}
	if in.CliOptions != nil {
		req.CliOptions = api.NewOptCliOptions(api.CliOptions{
			GoModuleName: api.NewOptString(in.CliOptions.ModuleName),
		})
	}

	res, err := s.api.CreateSdk(ctx, req)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.CreateSdkOK:
		return nil
	case *api.SdkError:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return postmanerr.Empty(http.StatusForbidden)
	case *api.CreateSdkUnprocessableEntity:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusUnprocessableEntity)
	case *api.CreateSdkTooManyRequests:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusTooManyRequests)
	case *api.Common500Error:
		return postmanerr.Empty(http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

func retryOptionsToAPI(r *RetryOptions) api.SdkRetryOptions {
	out := api.SdkRetryOptions{
		RetryDelay: r.RetryDelay,
	}
	if r.Enabled {
		out.Enabled = api.NewOptBool(true)
	}
	if r.MaxAttempts > 0 {
		out.MaxAttempts = api.NewOptInt(r.MaxAttempts)
	}
	if b, err := json.Marshal(r.MaxDelay); err == nil {
		out.MaxDelay = b
	}
	if b, err := json.Marshal(r.BackOffFactor); err == nil {
		out.BackOffFactor = b
	}
	if b, err := json.Marshal(r.RetryDelayJitter); err == nil {
		out.RetryDelayJitter = b
	}
	out.HttpCodesToRetry = r.HTTPCodesToRetry
	for _, m := range r.HTTPMethodsToRetry {
		out.HttpMethodsToRetry = append(out.HttpMethodsToRetry, api.HttpMethodsToRetry(m))
	}
	return out
}

// --- Status ------------------------------------------------------------

// Status returns information about the SDK, including its current build job
// status.
func (s *Service) Status(ctx context.Context, sdkID string) (*SDK, error) {
	res, err := s.api.GetSdk(ctx, api.GetSdkParams{SdkId: sdkID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.Sdk:
		out := sdkFromAPI(*r)
		return &out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetSdkNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GetSdkTooManyRequests:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusTooManyRequests)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Delete --------------------------------------------------------------

// Delete deletes an SDK record and its stored archive. It cannot cancel a
// generation job that is still in progress.
func (s *Service) Delete(ctx context.Context, sdkID string) error {
	res, err := s.api.DeleteSdk(ctx, api.DeleteSdkParams{SdkId: sdkID})
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.DeleteSdkOK:
		return nil
	case *api.Common401Error:
		return postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return postmanerr.Empty(http.StatusForbidden)
	case *api.DeleteSdkConflict:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusConflict)
	case *api.DeleteSdkNotFound:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.DeleteSdkTooManyRequests:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusTooManyRequests)
	case *api.Common500Error:
		return postmanerr.Empty(http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

// --- Download --------------------------------------------------------------

// Download describes a short-lived signed URL for a generated SDK's archive
// (zip). The URL is created on demand and expires within a few minutes.
type Download struct {
	ID        string
	Language  Language
	URL       string
	ExpiresAt string
}

// Download gets a short-lived signed URL for the generated SDK archive.
func (s *Service) Download(ctx context.Context, sdkID string) (*Download, error) {
	res, err := s.api.GetSdkDownloadUrl(ctx, api.GetSdkDownloadUrlParams{SdkId: sdkID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SdkDownload:
		return &Download{ID: r.ID, Language: Language(r.Language), URL: r.URL, ExpiresAt: r.ExpiresAt}, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetSdkDownloadUrlConflict:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusConflict)
	case *api.GetSdkDownloadUrlNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GetSdkDownloadUrlTooManyRequests:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusTooManyRequests)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- GitConnections ----------------------------------------------------

// GitConnection describes a link between one source element (collection or
// specification) and language, and a target Git repository.
type GitConnection struct {
	ID                            string
	Source                        Source
	Language                      Language
	Status                        GitConnectionStatus
	RepositoryURL                 string
	TargetBranch                  string
	AutoUpdatePullRequestsEnabled bool
	// SDK is the SDK currently sent to TargetBranch, when known.
	SDK          *SDK
	PullRequests []PullRequestRef
	CreatedAt    string
	UpdatedAt    string
}

func gitConnectionFromAPI(c api.SdkGitConnection) GitConnection {
	out := GitConnection{
		ID:                            c.SdkGitConnectionId,
		Source:                        Source{Type: SourceType(c.Source.Type), ID: c.Source.ID},
		Language:                      Language(c.Language),
		Status:                        GitConnectionStatus(c.Status),
		RepositoryURL:                 c.RepositoryUrl,
		TargetBranch:                  c.TargetBranch,
		AutoUpdatePullRequestsEnabled: c.AutoUpdatePullRequestsEnabled,
		CreatedAt:                     c.CreatedAt,
		UpdatedAt:                     c.UpdatedAt,
	}
	if sdk, ok := c.Sdk.Get(); ok {
		s := sdkFromAPI(sdk)
		out.SDK = &s
	}
	for _, pr := range c.PullRequests {
		out.PullRequests = append(out.PullRequests, PullRequestRef{URL: pr.URL, Status: string(pr.Status), SDKID: pr.SdkId})
	}
	return out
}

// GitConnectionsInput holds the filters and pagination options for
// GitConnections.
type GitConnectionsInput struct {
	// WorkspaceID is the workspace that owns the source entities. Required.
	WorkspaceID string
	// SourceID filters results by the originating collection or spec ID.
	SourceID string
	// Language filters results by SDK language.
	Language Language
	// Status filters results by connection status.
	Status GitConnectionStatus
	// RepositoryURL filters results by the canonical Git repository URL.
	RepositoryURL string
	// Cursor is the pagination cursor (use GitConnectionsResult.NextCursor to page).
	Cursor string
	// Limit is the maximum number of rows to return (up to 25).
	Limit int
}

// GitConnectionsResult is the paginated result of GitConnections.
type GitConnectionsResult struct {
	Connections []GitConnection
	NextCursor  string
	Total       int
}

// GitConnections gets all Git repository connections the authenticated user
// has access to in the given workspace.
func (s *Service) GitConnections(ctx context.Context, in *GitConnectionsInput) (*GitConnectionsResult, error) {
	if in == nil {
		in = &GitConnectionsInput{}
	}
	params := api.GetSdkGitConnectionsParams{WorkspaceId: in.WorkspaceID}
	if in.SourceID != "" {
		params.SourceId = api.NewOptString(in.SourceID)
	}
	if in.Language != "" {
		params.Language = api.NewOptSdkLanguage(api.SdkLanguage(in.Language))
	}
	if in.Status != "" {
		params.Status = api.NewOptSdkGitConnectionStatus(api.SdkGitConnectionStatus(in.Status))
	}
	if in.RepositoryURL != "" {
		params.RepositoryUrl = api.NewOptString(in.RepositoryURL)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}

	res, err := s.api.GetSdkGitConnections(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SdkGitConnectionList:
		out := &GitConnectionsResult{Total: rawJSONInt([]byte(r.Meta.Total))}
		out.NextCursor = r.Meta.NextCursor.Or("")
		for _, d := range r.Data {
			out.Connections = append(out.Connections, gitConnectionFromAPI(d))
		}
		return out, nil
	case *api.SdkError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusTooManyRequests)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- ConnectGit --------------------------------------------------------

// ConnectGitInput holds the fields for ConnectGit.
type ConnectGitInput struct {
	// Source is the collection or specification to connect.
	Source Source
	// Language is the SDK language this connection generates.
	Language Language
	// RepositoryURL is the target Git repository's canonical URL.
	RepositoryURL string
	// TargetBranch is the branch to push SDK updates to.
	TargetBranch string
	// AutoUpdatePullRequestsEnabled opens pull requests automatically when the
	// source changes. Enterprise plans only; always false on Team plans.
	AutoUpdatePullRequestsEnabled bool
}

// ConnectGit connects a Postman source element (collection or specification)
// to a Git repository for one SDK language, creating a new connection in the
// active state. Each source/language pair maps to a single connection; if one
// already exists, this returns a 409 Conflict error.
func (s *Service) ConnectGit(ctx context.Context, in *ConnectGitInput) (*GitConnection, error) {
	if in == nil {
		in = &ConnectGitInput{}
	}
	req := &api.CreateSdkGitConnection{
		Source:        api.SdkSource{Type: api.ElementType2(in.Source.Type), ID: in.Source.ID},
		Language:      api.SdkLanguage(in.Language),
		RepositoryUrl: in.RepositoryURL,
	}
	if in.TargetBranch != "" {
		req.TargetBranch = api.NewOptString(in.TargetBranch)
	}
	if in.AutoUpdatePullRequestsEnabled {
		req.AutoUpdatePullRequestsEnabled = api.NewOptBool(true)
	}

	res, err := s.api.CreateSdkGitConnection(ctx, req)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SdkGitConnection:
		out := gitConnectionFromAPI(*r)
		return &out, nil
	case *api.SdkError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.CreateSdkGitConnectionConflict:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusConflict)
	case *api.CreateSdkGitConnectionTooManyRequests:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusTooManyRequests)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- GitConnection -------------------------------------------------------

// GitConnection gets information about an SDK's Git connection, including
// the SDK currently sent to the target branch and the most recent
// SDK-update pull request.
func (s *Service) GitConnection(ctx context.Context, connectionID string) (*GitConnection, error) {
	res, err := s.api.GetSdkGitConnection(ctx, api.GetSdkGitConnectionParams{SdkGitConnectionId: connectionID})
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SdkGitConnection:
		out := gitConnectionFromAPI(*r)
		return &out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetSdkGitConnectionNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GetSdkGitConnectionTooManyRequests:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusTooManyRequests)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- UpdateGitConnection -------------------------------------------------

// UpdateGitConnectionInput holds the fields for UpdateGitConnection.
type UpdateGitConnectionInput struct {
	// Status is the connection's new lifecycle status. Only
	// GitConnectionStatusActive and GitConnectionStatusDisconnected can be set;
	// GitConnectionStatusInaccessible is system-determined.
	Status GitConnectionStatus
	// AutoUpdatePullRequestsEnabled, when non-nil, sets whether pull requests
	// open automatically when the source changes (Enterprise plans only).
	AutoUpdatePullRequestsEnabled *bool
}

// UpdateGitConnection updates an SDK Git connection's lifecycle status. This
// action is idempotent: setting fields to their current values is a no-op.
func (s *Service) UpdateGitConnection(ctx context.Context, connectionID string, in *UpdateGitConnectionInput) (*GitConnection, error) {
	if in == nil {
		in = &UpdateGitConnectionInput{}
	}
	req := &api.UpdateSdkGitConnection{Status: api.UpdateSdkGitConnectionStatus(in.Status)}
	if in.AutoUpdatePullRequestsEnabled != nil {
		req.AutoUpdatePullRequestsEnabled = api.NewOptBool(*in.AutoUpdatePullRequestsEnabled)
	}
	params := api.UpdateSdkGitConnectionParams{SdkGitConnectionId: connectionID}

	res, err := s.api.UpdateSdkGitConnection(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SdkGitConnection:
		out := gitConnectionFromAPI(*r)
		return &out, nil
	case *api.SdkError:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.UpdateSdkGitConnectionNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.UpdateSdkGitConnectionUnprocessableEntity:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnprocessableEntity)
	case *api.UpdateSdkGitConnectionTooManyRequests:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusTooManyRequests)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- GitConnectionPullRequests -------------------------------------------

// PullRequest is a single SDK-update pull request opened for a Git
// connection.
type PullRequest struct {
	// Number is the pull request number, rendered as decimal text (the
	// reconstructed OpenAPI spec could not resolve this field's exact numeric
	// type).
	Number    string
	URL       string
	Status    string
	SDK       *SDK
	CreatedAt string
	UpdatedAt string
}

// PullRequestsInput holds the filters and pagination options for
// GitConnectionPullRequests.
type PullRequestsInput struct {
	// Status filters results by pull request status.
	Status PRStatus
	// Cursor is the pagination cursor (use PullRequestsResult.NextCursor to page).
	Cursor string
	// Limit is the maximum number of rows to return (up to 25).
	Limit int
}

// PullRequestsResult is the paginated result of GitConnectionPullRequests.
type PullRequestsResult struct {
	PullRequests []PullRequest
	NextCursor   string
	Total        int
}

// GitConnectionPullRequests lists all SDK-update pull requests for the Git
// connection, newest first by updatedAt.
func (s *Service) GitConnectionPullRequests(ctx context.Context, connectionID string, in *PullRequestsInput) (*PullRequestsResult, error) {
	if in == nil {
		in = &PullRequestsInput{}
	}
	params := api.GetSdkGitConnectionPullRequestsParams{SdkGitConnectionId: connectionID}
	if in.Status != "" {
		params.Status = api.NewOptSdkGitConnectionPrStatus(api.SdkGitConnectionPrStatus(in.Status))
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}

	res, err := s.api.GetSdkGitConnectionPullRequests(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SdkGitConnectionPullRequestList:
		out := &PullRequestsResult{Total: rawJSONInt([]byte(r.Meta.Total))}
		out.NextCursor = r.Meta.NextCursor.Or("")
		for _, d := range r.Data {
			pr := PullRequest{
				Number:    rawJSONText([]byte(d.Number)),
				URL:       d.URL,
				Status:    string(d.Status),
				CreatedAt: d.CreatedAt,
				UpdatedAt: d.UpdatedAt,
			}
			if sdk, ok := d.Sdk.Get(); ok {
				sk := sdkFromAPI(sdk)
				pr.SDK = &sk
			}
			out.PullRequests = append(out.PullRequests, pr)
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetSdkGitConnectionPullRequestsNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GetSdkGitConnectionPullRequestsTooManyRequests:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusTooManyRequests)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- helpers -----------------------------------------------------------

// rawJSONText renders a raw JSON scalar the reconstructed OpenAPI spec could
// not resolve to a concrete type as a display string: JSON strings are
// unquoted, other scalars (numbers, objects) pass through verbatim.
func rawJSONText(raw []byte) string {
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return s
		}
	}
	return string(raw)
}

// rawJSONInt parses a raw JSON numeric scalar, defaulting to 0 if it is
// absent or not a number.
func rawJSONInt(raw []byte) int {
	if len(raw) == 0 {
		return 0
	}
	var n int
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0
	}
	return n
}

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}

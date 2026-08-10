// Package specs provides a high-level client for Postman's API Spec Hub.
//
// Spec Hub stores API specifications (OpenAPI, AsyncAPI, protobuf, GraphQL,
// and Smithy), their files, and the collections generated from (or synced
// with) them. It also tracks specification version tags, which are snapshots
// of a specification at a point in time.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	specs, _ := client.Specs().List(ctx, &specs.ListInput{WorkspaceID: workspaceID})
//	spec, _ := client.Specs().Get(ctx, specs.Specs[0].ID)
package specs

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// SpecType is the type of an API specification.
type SpecType string

// Specification type values.
const (
	SpecTypeOpenAPI20  SpecType = "OPENAPI:2.0"
	SpecTypeOpenAPI30  SpecType = "OPENAPI:3.0"
	SpecTypeOpenAPI31  SpecType = "OPENAPI:3.1"
	SpecTypeAsyncAPI20 SpecType = "ASYNCAPI:2.0"
	SpecTypeAsyncAPI30 SpecType = "ASYNCAPI:3.0"
	SpecTypeProtobuf2  SpecType = "PROTOBUF:2"
	SpecTypeProtobuf3  SpecType = "PROTOBUF:3"
	SpecTypeGraphQL    SpecType = "GRAPHQL"
	SpecTypeSmithy20   SpecType = "SMITHY:2.0"
)

// FileFormat is the file format Postman detected for a specification.
type FileFormat string

// File format values.
const (
	FileFormatJSON    FileFormat = "json"
	FileFormatYAML    FileFormat = "yaml"
	FileFormatProto   FileFormat = "proto"
	FileFormatGraphQL FileFormat = "graphql"
	FileFormatSmithy  FileFormat = "smithy"
)

// FileType indicates whether a specification file is the entry point of a
// multi-file specification or one of the files it references.
type FileType string

// File type values.
const (
	// FileTypeRoot is the file containing the full specification structure.
	// Multi-file specifications can only have one root file.
	FileTypeRoot FileType = "ROOT"
	// FileTypeDefault is a file referenced by the root file.
	FileTypeDefault FileType = "DEFAULT"
)

// SyncState is the sync status between a specification and a collection
// generated from (or linked to) it.
type SyncState string

// Sync state values.
const (
	SyncStateInSync         SyncState = "in-sync"
	SyncStateOutOfSync      SyncState = "out-of-sync"
	SyncStateSyncInProgress SyncState = "sync-in-progress"
)

// TaskElementType is the kind of element an asynchronous task belongs to.
type TaskElementType string

// Task element type values.
const (
	TaskElementTypeCollections TaskElementType = "collections"
	TaskElementTypeSpecs       TaskElementType = "specs"
)

// SpecFormat is the serialization format of a generated specification.
type SpecFormat string

// Specification format values.
const (
	SpecFormatJSON SpecFormat = "JSON"
	SpecFormatYAML SpecFormat = "YAML"
)

// RequestNameSource controls how request names are derived when generating a
// collection from a specification.
type RequestNameSource string

// Request name source values.
const (
	RequestNameSourceFallback RequestNameSource = "Fallback"
	RequestNameSourceURL      RequestNameSource = "URL"
)

// IndentCharacter is the whitespace character used to indent a generated
// collection's request bodies.
type IndentCharacter string

// Indent character values.
const (
	IndentCharacterTab   IndentCharacter = "Tab"
	IndentCharacterSpace IndentCharacter = "Space"
)

// FolderStrategy controls how requests are grouped into folders when
// generating a collection from a specification.
type FolderStrategy string

// Folder strategy values.
const (
	FolderStrategyPaths FolderStrategy = "Paths"
	FolderStrategyTags  FolderStrategy = "Tags"
)

// VersionTagEntryType is the kind of entry captured in a specification
// version tag snapshot.
type VersionTagEntryType string

// Version tag entry type values.
const (
	VersionTagEntryTypeFile   VersionTagEntryType = "FILE"
	VersionTagEntryTypeFolder VersionTagEntryType = "FOLDER"
)

// Service is the high-level API Spec Hub client. Obtain one via
// postman.Client.Specs.
type Service struct {
	api *api.Client
}

// New creates a Specs service over the given generated API client. Most
// callers should use postman.Client.Specs instead of calling this directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- List --------------------------------------------------------------

// ListInput holds the filters and pagination options for List.
type ListInput struct {
	// WorkspaceID is required: the workspace to list specifications from.
	WorkspaceID string
	// Cursor is the pagination cursor (use ListResult.NextCursor to page).
	Cursor string
	// Limit is the maximum number of rows to return.
	Limit int
}

// ListResult is the paginated result of List.
type ListResult struct {
	Specs      []SpecSummary
	NextCursor string
}

// SpecSummary is a summary of an API specification.
type SpecSummary struct {
	ID        string
	Name      string
	Type      SpecType
	CreatedBy int
	UpdatedBy int
	CreatedAt string
	UpdatedAt string
}

// List returns all API specifications in a workspace.
func (s *Service) List(ctx context.Context, in *ListInput) (*ListResult, error) {
	if in == nil {
		in = &ListInput{}
	}
	params := api.GetAllSpecsParams{WorkspaceId: in.WorkspaceID}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}

	res, err := s.api.GetAllSpecs(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetAllSpecs:
		out := &ListResult{}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		for _, sp := range r.Specs {
			out.Specs = append(out.Specs, SpecSummary{
				ID:        sp.ID.Or(""),
				Name:      sp.Name.Or(""),
				Type:      SpecType(sp.Type.Or("")),
				CreatedBy: sp.CreatedBy.Or(0),
				UpdatedBy: sp.UpdatedBy.Or(0),
				CreatedAt: sp.CreatedAt.Or(""),
				UpdatedAt: sp.UpdatedAt.Or(""),
			})
		}
		return out, nil
	case *api.GetAllSpecsBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.GetAllSpecsUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Create --------------------------------------------------------------

// CreateInput holds the fields for creating an API specification.
type CreateInput struct {
	// WorkspaceID is required: the workspace to create the specification in.
	WorkspaceID string
	// Name is the specification's name.
	Name string
	// Type is the type of API specification.
	Type SpecType
	// Files is the specification's files and their contents. Multi-file
	// specifications must set Type on every file (exactly one FileTypeRoot);
	// single-file specifications should leave Type empty.
	Files []SpecFileInput
}

// SpecFileInput describes a file to create alongside a new specification.
type SpecFileInput struct {
	// Path is the file's path. Accepts .json, .yaml, .proto, .graphql, and
	// .smithy file types. A path containing "/" creates a folder.
	Path string
	// Content is the file's stringified contents.
	Content string
	// Type is required for multi-file specifications; leave empty for
	// single-file specifications.
	Type FileType
}

// CreateResult is the specification created by Create.
type CreateResult struct {
	ID        string
	Name      string
	Type      SpecType
	CreatedBy int
	UpdatedBy int
	CreatedAt string
	UpdatedAt string
}

// Create creates an API specification in Postman's Spec Hub. Specifications
// can be single or multi-file; Postman supports OpenAPI, AsyncAPI, protobuf,
// GraphQL, and Smithy specifications.
//
//nolint:dupl // Structurally parallel to sibling wrapper methods over distinct generated types; not meaningfully extractable without reflection or per-type adapters
func (s *Service) Create(ctx context.Context, in *CreateInput) (*CreateResult, error) {
	if in == nil {
		in = &CreateInput{}
	}
	files, err := encodeSpecFiles(in.Files)
	if err != nil {
		return nil, err
	}
	req := &api.CreateSpec{
		Name:  in.Name,
		Type:  api.SpecType(in.Type),
		Files: files,
	}
	params := api.CreateSpecParams{WorkspaceId: in.WorkspaceID}

	res, err := s.api.CreateSpec(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateSpecResponse:
		return &CreateResult{
			ID:        r.ID.Or(""),
			Name:      r.Name.Or(""),
			Type:      SpecType(r.Type.Or("")),
			CreatedBy: r.CreatedBy.Or(0),
			UpdatedBy: r.UpdatedBy.Or(0),
			CreatedAt: r.CreatedAt.Or(""),
			UpdatedAt: r.UpdatedAt.Or(""),
		}, nil
	case *api.CreateSpecBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.CreateSpecUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.CreateApiClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func encodeSpecFiles(files []SpecFileInput) ([]api.CreateSpecFiles, error) {
	if len(files) == 0 {
		return nil, nil
	}
	out := make([]api.CreateSpecFiles, 0, len(files))
	for _, f := range files {
		raw, err := json.Marshal(struct {
			Path    string `json:"path"`
			Content string `json:"content"`
			Type    string `json:"type,omitempty"`
		}{
			Path:    f.Path,
			Content: f.Content,
			Type:    string(f.Type),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, api.CreateSpecFiles(raw))
	}
	return out, nil
}

// --- Get --------------------------------------------------------------

// Spec is information about an API specification.
type Spec struct {
	ID         string
	Name       string
	FileFormat FileFormat
	Type       SpecType
	CreatedBy  int
	UpdatedBy  int
	CreatedAt  string
	UpdatedAt  string
}

// Get returns information about an API specification.
func (s *Service) Get(ctx context.Context, specID string) (*Spec, error) {
	params := api.GetSpecParams{SpecId: specID}

	res, err := s.api.GetSpec(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.SpecInformation:
		return &Spec{
			ID:         r.ID.Or(""),
			Name:       r.Name.Or(""),
			FileFormat: FileFormat(r.FileFormat.Or("")),
			Type:       SpecType(r.Type.Or("")),
			CreatedBy:  r.CreatedBy.Or(0),
			UpdatedBy:  r.UpdatedBy.Or(0),
			CreatedAt:  r.CreatedAt.Or(""),
			UpdatedAt:  r.UpdatedAt.Or(""),
		}, nil
	case *api.GetSpecUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetSpecNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- UpdateProperties ----------------------------------------------------

// UpdatePropertiesInput holds the fields for updating a specification's
// properties.
type UpdatePropertiesInput struct {
	// Name is the specification's new name.
	Name string
}

// UpdatePropertiesResult is the specification returned by UpdateProperties.
type UpdatePropertiesResult struct {
	ID        string
	Name      string
	Type      SpecType
	CreatedBy int
	UpdatedBy int
	CreatedAt string
	UpdatedAt string
}

// UpdateProperties updates an API specification's properties, such as its
// name.
//
//nolint:dupl // Structurally parallel to sibling wrapper methods over distinct generated types; not meaningfully extractable without reflection or per-type adapters
func (s *Service) UpdateProperties(ctx context.Context, specID string, in *UpdatePropertiesInput) (*UpdatePropertiesResult, error) {
	if in == nil {
		in = &UpdatePropertiesInput{}
	}
	req := &api.UpdateSpecProperties{Name: in.Name}
	params := api.UpdateSpecPropertiesParams{SpecId: specID}

	res, err := s.api.UpdateSpecProperties(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.UpdateSpecPropertiesResponse:
		return &UpdatePropertiesResult{
			ID:        r.ID.Or(""),
			Name:      r.Name.Or(""),
			Type:      SpecType(r.Type.Or("")),
			CreatedBy: r.CreatedBy.Or(0),
			UpdatedBy: r.UpdatedBy.Or(0),
			CreatedAt: r.CreatedAt.Or(""),
			UpdatedAt: r.UpdatedAt.Or(""),
		}, nil
	case *api.UpdateSpecPropertiesBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.UpdateSpecPropertiesUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.UpdateSpecPropertiesNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Delete --------------------------------------------------------------

// Delete deletes an API specification.
func (s *Service) Delete(ctx context.Context, specID string) error {
	params := api.DeleteSpecParams{SpecId: specID}

	res, err := s.api.DeleteSpec(ctx, params)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.DeleteSpecOK:
		return nil
	case *api.DeleteSpecUnauthorized:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return postmanerr.Empty(http.StatusForbidden)
	case *api.DeleteSpecNotFound:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return postmanerr.Empty(http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

// --- Definition --------------------------------------------------------------

// Definition confirms that the complete contents of an OpenAPI or AsyncAPI
// specification's definition were retrieved successfully.
//
// Known limitation: Postman's generated schema for this endpoint's response
// body is free-form (the TypeScript SDK types it as `any`), which the
// OpenAPI reconstruction this SDK is generated from cannot capture as a
// concrete type. As a result the generated client discards the response
// body entirely, so the definition's contents are not retrievable through
// this method; it only reports whether the request succeeded.
func (s *Service) Definition(ctx context.Context, specID string) error {
	params := api.GetSpecDefinitionParams{SpecId: specID}

	res, err := s.api.GetSpecDefinition(ctx, params)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.GetSpecDefinitionOK:
		return nil
	case *api.GetSpecDefinitionBadRequest:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.GetSpecDefinitionUnauthorized:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return postmanerr.Empty(http.StatusForbidden)
	case *api.GetSpecDefinitionNotFound:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return postmanerr.Empty(http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

// --- Files --------------------------------------------------------------

// FilesResult is the set of files in an API specification.
type FilesResult struct {
	Files      []SpecFileSummary
	NextCursor string
}

// SpecFileSummary is a summary of a file in an API specification.
type SpecFileSummary struct {
	ID        string
	Path      string
	Name      string
	Type      FileType
	CreatedBy int
	UpdatedBy int
	CreatedAt string
	UpdatedAt string
}

// Files returns all the files in an API specification.
func (s *Service) Files(ctx context.Context, specID string) (*FilesResult, error) {
	params := api.GetSpecFilesParams{SpecId: specID}

	res, err := s.api.GetSpecFiles(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetSpecFiles:
		out := &FilesResult{}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		for _, f := range r.Files {
			out.Files = append(out.Files, SpecFileSummary{
				ID:        f.ID.Or(""),
				Path:      f.Path.Or(""),
				Name:      f.Name.Or(""),
				Type:      FileType(f.Type.Or("")),
				CreatedBy: f.CreatedBy.Or(0),
				UpdatedBy: f.UpdatedBy.Or(0),
				CreatedAt: f.CreatedAt.Or(""),
				UpdatedAt: f.UpdatedAt.Or(""),
			})
		}
		return out, nil
	case *api.GetSpecFilesUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetSpecFilesNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- CreateFile --------------------------------------------------------------

// CreateFileInput holds the fields for creating a specification file.
type CreateFileInput struct {
	// Path is the file's path. Accepts JSON or YAML files. A path
	// containing "/" creates a folder. Creating a file assigns it
	// FileTypeDefault.
	Path string
	// Content is the file's stringified contents.
	Content string
}

// FileResult is a specification file returned by CreateFile or UpdateFile.
type FileResult struct {
	ID        string
	Path      string
	Type      FileType
	CreatedBy int
	UpdatedBy int
	CreatedAt string
	UpdatedAt string
}

// CreateFile creates a file for an OpenAPI or a protobuf 2 or 3
// specification. Multi-file specifications can only have one root file;
// files cannot exceed 10 MB.
func (s *Service) CreateFile(ctx context.Context, specID string, in *CreateFileInput) (*FileResult, error) {
	if in == nil {
		in = &CreateFileInput{}
	}
	req := &api.CreateSpecFile{Path: in.Path, Content: in.Content}
	params := api.CreateSpecFileParams{SpecId: specID}

	res, err := s.api.CreateSpecFile(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateUpdateSpecFileResponse:
		return fileResultFromAPI(r), nil
	case *api.CreateSpecFileBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.CreateSpecFileUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.CreateSpecFileNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func fileResultFromAPI(r *api.CreateUpdateSpecFileResponse) *FileResult {
	return &FileResult{
		ID:        r.ID.Or(""),
		Path:      r.Path.Or(""),
		Type:      FileType(r.Type.Or("")),
		CreatedBy: r.CreatedBy.Or(0),
		UpdatedBy: r.UpdatedBy.Or(0),
		CreatedAt: r.CreatedAt.Or(""),
		UpdatedAt: r.UpdatedAt.Or(""),
	}
}

// --- File --------------------------------------------------------------

// FileContent is the contents of an API specification's file.
type FileContent struct {
	ID        string
	Name      string
	Path      string
	Content   string
	Type      FileType
	CreatedBy int
	UpdatedBy int
	CreatedAt string
	UpdatedAt string
}

// File returns the contents of an API specification's file.
func (s *Service) File(ctx context.Context, specID, filePath string) (*FileContent, error) {
	params := api.GetSpecFileParams{SpecId: specID, FilePath: filePath}

	res, err := s.api.GetSpecFile(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetSpecFile:
		return &FileContent{
			ID:        r.ID.Or(""),
			Name:      r.Name.Or(""),
			Path:      r.Path.Or(""),
			Content:   r.Content.Or(""),
			Type:      FileType(r.Type.Or("")),
			CreatedBy: r.CreatedBy.Or(0),
			UpdatedBy: r.UpdatedBy.Or(0),
			CreatedAt: r.CreatedAt.Or(""),
			UpdatedAt: r.UpdatedAt.Or(""),
		}, nil
	case *api.GetSpecFileBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.GetSpecFileUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetSpecFileNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- UpdateFile --------------------------------------------------------------

// UpdateFileInput holds the fields for updating a specification file. This
// endpoint does not accept multiple properties in a single call: set
// exactly one of Name, Type, or Content.
type UpdateFileInput struct {
	// Name is the file's new name.
	Name string
	// Type is the file's new type. Setting FileTypeRoot demotes the
	// previous root file to FileTypeDefault.
	Type FileType
	// Content is the file's new stringified contents.
	Content string
}

// UpdateFile updates a file for an OpenAPI or protobuf 2 or 3 specification.
// Files cannot exceed 10 MB.
func (s *Service) UpdateFile(ctx context.Context, specID, filePath string, in *UpdateFileInput) (*FileResult, error) {
	if in == nil {
		in = &UpdateFileInput{}
	}
	req := &api.UpdateSpecFile{}
	if in.Name != "" {
		req.Name = api.NewOptString(in.Name)
	}
	if in.Type != "" {
		req.Type = api.NewOptUpdateSpecFileType(api.UpdateSpecFileType(in.Type))
	}
	if in.Content != "" {
		req.Content = api.NewOptString(in.Content)
	}
	params := api.UpdateSpecFileParams{SpecId: specID, FilePath: filePath}

	res, err := s.api.UpdateSpecFile(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateUpdateSpecFileResponse:
		return fileResultFromAPI(r), nil
	case *api.UpdateSpecFileBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.UpdateSpecFileUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.UpdateSpecFileNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- DeleteFile --------------------------------------------------------------

// DeleteFile deletes a file in an API specification.
func (s *Service) DeleteFile(ctx context.Context, specID, filePath string) error {
	params := api.DeleteSpecFileParams{SpecId: specID, FilePath: filePath}

	res, err := s.api.DeleteSpecFile(ctx, params)
	if err != nil {
		return err
	}

	switch r := res.(type) {
	case *api.DeleteSpecFileOK:
		return nil
	case *api.DeleteSpecFileBadRequest:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.DeleteSpecFileUnauthorized:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return postmanerr.Empty(http.StatusForbidden)
	case *api.DeleteSpecFileNotFound:
		return postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return postmanerr.Empty(http.StatusInternalServerError)
	default:
		return errUnexpectedResponse(res)
	}
}

// --- Collections --------------------------------------------------------------

// CollectionsInput holds the pagination options for Collections.
type CollectionsInput struct {
	// Limit is the maximum number of rows to return.
	Limit int
	// Cursor is the pagination cursor (use CollectionsResult.NextCursor to page).
	Cursor string
}

// CollectionsResult is the paginated result of Collections.
type CollectionsResult struct {
	Collections []SpecCollection
	NextCursor  string
}

// SpecCollection is a collection generated from an API specification.
type SpecCollection struct {
	ID        string
	Name      string
	State     SyncState
	CreatedBy int
	UpdatedBy int
	CreatedAt string
	UpdatedAt string
}

// Collections returns all of an API specification's generated collections.
func (s *Service) Collections(ctx context.Context, specID string, in *CollectionsInput) (*CollectionsResult, error) {
	if in == nil {
		in = &CollectionsInput{}
	}
	params := api.GetSpecCollectionsParams{SpecId: specID, ElementType: "collection"}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}

	res, err := s.api.GetSpecCollections(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetSpecCollections:
		out := &CollectionsResult{}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		for _, c := range r.Collections {
			out.Collections = append(out.Collections, SpecCollection{
				ID:        c.ID.Or(""),
				Name:      c.Name.Or(""),
				State:     SyncState(c.State.Or("")),
				CreatedBy: c.CreatedBy.Or(0),
				UpdatedAt: c.UpdatedAt.Or(""),
				CreatedAt: c.CreatedAt.Or(""),
			})
		}
		return out, nil
	case *api.GetSpecCollectionsUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetSpecCollectionsNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- GenerateCollection --------------------------------------------------------------

// GenerateCollectionInput holds the fields for generating a collection from
// a specification.
type GenerateCollectionInput struct {
	// Name is the generated collection's name.
	Name string
	// RequestNameSource controls how generated request names are derived.
	RequestNameSource RequestNameSource
	// IndentCharacter is the whitespace character used to indent bodies.
	IndentCharacter IndentCharacter
	// ParametersResolution controls how parameters are resolved into
	// example values.
	ParametersResolution string
	// FolderStrategy controls how requests are grouped into folders.
	FolderStrategy              FolderStrategy
	IncludeAuthInfoInExample    bool
	EnableOptionalParameters    bool
	KeepImplicitHeaders         bool
	IncludeDeprecated           bool
	AlwaysInheritAuthentication bool
	NestedFolderHierarchy       bool
}

// TaskResult is a reference to an asynchronous task created by a generate or
// sync operation. Use TaskStatus to poll it.
type TaskResult struct {
	TaskID string
	URL    string
}

// GenerateCollection creates a collection from the given OpenAPI 2.0, 3.0,
// or 3.1 specification or Smithy specification. The response contains a
// polling link to the task status; use TaskStatus to check on it.
func (s *Service) GenerateCollection(ctx context.Context, specID string, in *GenerateCollectionInput) (*TaskResult, error) {
	if in == nil {
		in = &GenerateCollectionInput{}
	}
	opts := api.GenerateCollectionOptions{}
	if in.RequestNameSource != "" {
		opts.RequestNameSource = api.NewOptRequestNameSource(api.RequestNameSource(in.RequestNameSource))
	}
	if in.IndentCharacter != "" {
		opts.IndentCharacter = api.NewOptIndentCharacter(api.IndentCharacter(in.IndentCharacter))
	}
	if in.ParametersResolution != "" {
		opts.ParametersResolution = api.NewOptString(in.ParametersResolution)
	}
	if in.FolderStrategy != "" {
		opts.FolderStrategy = api.NewOptFolderStrategy(api.FolderStrategy(in.FolderStrategy))
	}
	if in.IncludeAuthInfoInExample {
		opts.IncludeAuthInfoInExample = api.NewOptBool(true)
	}
	if in.EnableOptionalParameters {
		opts.EnableOptionalParameters = api.NewOptBool(true)
	}
	if in.KeepImplicitHeaders {
		opts.KeepImplicitHeaders = api.NewOptBool(true)
	}
	if in.IncludeDeprecated {
		opts.IncludeDeprecated = api.NewOptBool(true)
	}
	if in.AlwaysInheritAuthentication {
		opts.AlwaysInheritAuthentication = api.NewOptBool(true)
	}
	if in.NestedFolderHierarchy {
		opts.NestedFolderHierarchy = api.NewOptBool(true)
	}

	req := &api.GenerateCollection{Name: in.Name, Options: opts}
	params := api.GenerateCollectionParams{SpecId: specID, ElementType: "collection"}

	res, err := s.api.GenerateCollection(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.TaskCreated:
		return &TaskResult{TaskID: r.TaskId.Or(""), URL: r.URL.Or("")}, nil
	case *api.GenerateCollectionUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GenerateCollectionNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GenerateCollectionLocked:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusLocked)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- SyncWithCollection --------------------------------------------------------------

// SyncWithCollection syncs an API specification linked to a collection. This
// only supports OpenAPI 2.0, 3.0, and 3.1 specifications, and only syncs
// collections generated from the given specification ID. This is an
// asynchronous operation; use TaskStatus to poll it.
func (s *Service) SyncWithCollection(ctx context.Context, specID, collectionUID string) (*TaskResult, error) {
	params := api.SyncSpecWithCollectionParams{SpecId: specID, CollectionUid: collectionUID}

	res, err := s.api.SyncSpecWithCollection(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.TaskCreated:
		return &TaskResult{TaskID: r.TaskId.Or(""), URL: r.URL.Or("")}, nil
	case *api.SyncSpecWithCollectionUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.SyncSpecWithCollectionNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- UpdateSyncOptions --------------------------------------------------------------

// SyncOptionsInput holds the sync options for UpdateSyncOptions.
type SyncOptionsInput struct {
	// SyncExamples, when true, syncs the collection's examples with the
	// specification.
	SyncExamples bool
	// DeleteOrphanedRequests, when true, deletes requests in the collection
	// that no longer exist in the specification.
	DeleteOrphanedRequests bool
}

// SyncOptionsResult is the sync options returned by UpdateSyncOptions.
type SyncOptionsResult struct {
	SyncExamples           bool
	DeleteOrphanedRequests bool
}

// UpdateSyncOptions updates the sync options for a specification's generated
// collection.
func (s *Service) UpdateSyncOptions(ctx context.Context, specID, collectionID string, in *SyncOptionsInput) (*SyncOptionsResult, error) {
	if in == nil {
		in = &SyncOptionsInput{}
	}
	req := &api.ApiSpecSyncOptions{
		SyncOptions: api.NewOptSyncOptions(api.SyncOptions{
			SyncExamples:           api.NewOptBool(in.SyncExamples),
			DeleteOrphanedRequests: api.NewOptBool(in.DeleteOrphanedRequests),
		}),
	}
	params := api.UpdateSpecSyncOptionsParams{SpecId: specID, CollectionId: collectionID}

	res, err := s.api.UpdateSpecSyncOptions(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.ApiSpecSyncOptions:
		out := &SyncOptionsResult{}
		if opts, ok := r.SyncOptions.Get(); ok {
			out.SyncExamples = opts.SyncExamples.Or(false)
			out.DeleteOrphanedRequests = opts.DeleteOrphanedRequests.Or(false)
		}
		return out, nil
	case *api.UpdateSpecSyncOptionsBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.ErrorTypeTitleDetailStatusInstance:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.UpdateSpecSyncOptionsNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- CollectionSpecs --------------------------------------------------------------

// CollectionSpecsResult is the paginated result of CollectionSpecs.
type CollectionSpecsResult struct {
	Specs      []GeneratedSpec
	NextCursor string
}

// GeneratedSpec is a specification generated from a collection.
type GeneratedSpec struct {
	ID        string
	Name      string
	State     SyncState
	CreatedBy int
	UpdatedBy int
	CreatedAt string
	UpdatedAt string
}

// CollectionSpecs returns the API specifications generated for the given
// collection.
func (s *Service) CollectionSpecs(ctx context.Context, collectionUID string) (*CollectionSpecsResult, error) {
	params := api.GetGeneratedCollectionSpecsParams{CollectionUid: collectionUID, ElementType: "spec"}

	res, err := s.api.GetGeneratedCollectionSpecs(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetGeneratedCollectionSpecs:
		out := &CollectionSpecsResult{}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		for _, sp := range r.Specs {
			out.Specs = append(out.Specs, GeneratedSpec{
				ID:        sp.ID.Or(""),
				Name:      sp.Name.Or(""),
				State:     SyncState(sp.State.Or("")),
				CreatedBy: sp.CreatedBy.Or(0),
				UpdatedBy: sp.UpdatedBy.Or(0),
				CreatedAt: sp.CreatedAt.Or(""),
				UpdatedAt: sp.UpdatedAt.Or(""),
			})
		}
		return out, nil
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- GenerateFromCollection --------------------------------------------------------------

// GenerateFromCollectionInput holds the fields for generating a
// specification from a collection.
type GenerateFromCollectionInput struct {
	// Name is the generated specification's name.
	Name string
	// Type is the specification type to generate. Only
	// SpecTypeOpenAPI20/30/31 are supported.
	Type SpecType
	// Format is the serialization format of the generated specification.
	Format SpecFormat
}

// GenerateFromCollection generates an OpenAPI 2.0, 3.0, or 3.1 specification
// for the given collection. The response contains a polling link to the
// task status; use TaskStatus to check on it.
func (s *Service) GenerateFromCollection(ctx context.Context, collectionUID string, in *GenerateFromCollectionInput) (*TaskResult, error) {
	if in == nil {
		in = &GenerateFromCollectionInput{}
	}
	req := &api.GenerateSpecFromCollection{Name: in.Name}
	if in.Type != "" {
		req.Type = api.NewOptGenerateSpecFromCollectionType(api.GenerateSpecFromCollectionType(in.Type))
	}
	if in.Format != "" {
		req.Format = api.NewOptFormat(api.Format(in.Format))
	}
	params := api.GenerateSpecFromCollectionParams{CollectionUid: collectionUID, ElementType: "spec"}

	res, err := s.api.GenerateSpecFromCollection(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.TaskCreated:
		return &TaskResult{TaskID: r.TaskId.Or(""), URL: r.URL.Or("")}, nil
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GenerateSpecFromCollectionNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.GenerateSpecFromCollectionLocked:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusLocked)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- SyncCollectionWithSpec --------------------------------------------------------------

// SyncCollectionWithSpec syncs a collection generated from an API
// specification. This only supports OpenAPI 2.0, 3.0, and 3.1
// specification types, and only syncs collections generated from the given
// specification ID. This is an asynchronous operation; use TaskStatus to
// poll it.
func (s *Service) SyncCollectionWithSpec(ctx context.Context, collectionUID, specID string) (*TaskResult, error) {
	params := api.SyncCollectionWithSpecParams{CollectionUid: collectionUID, SpecId: specID}

	res, err := s.api.SyncCollectionWithSpec(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.TaskCreated:
		return &TaskResult{TaskID: r.TaskId.Or(""), URL: r.URL.Or("")}, nil
	case *api.Api403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- TaskStatus --------------------------------------------------------------

// TaskStatusResult is the status of an asynchronous specification or
// collection generation task.
//
// Known limitation: the response schema is a TypeScript union
// (GetAsyncCollectionTaskStatus1 | AsyncTaskFailed) that the OpenAPI
// reconstruction cannot resolve to a concrete type, so the generated client
// returns the raw JSON body instead of typed fields. Unmarshal Raw into your
// own type, or inspect its "status" field ("processing", "completed", or
// "failed").
type TaskStatusResult struct {
	Raw json.RawMessage
}

// TaskStatus returns the status of an asynchronous API specification or
// collection creation task.
func (s *Service) TaskStatus(ctx context.Context, elementType TaskElementType, elementID, taskID string) (*TaskStatusResult, error) {
	params := api.GetAsyncSpecTaskStatusParams{
		ElementType: string(elementType),
		ElementId:   elementID,
		TaskId:      taskID,
	}

	res, err := s.api.GetAsyncSpecTaskStatus(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetAsyncCollectionTaskStatus:
		return &TaskStatusResult{Raw: json.RawMessage(*r)}, nil
	case *api.GetAsyncSpecTaskStatusUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetAsyncSpecTaskStatusNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- VersionTag --------------------------------------------------------------

// VersionTagResult is a snapshot of a specification captured by a version
// tag.
type VersionTagResult struct {
	Entries []VersionTagEntry
}

// VersionTagEntry is a single file or folder captured in a version tag
// snapshot.
type VersionTagEntry struct {
	ID        string
	Name      string
	Type      VersionTagEntryType
	Path      string
	FileType  FileType
	Content   string
	ParentID  string
	CreatedBy string
	UpdatedBy string
	CreatedAt string
	UpdatedAt string
}

// VersionTag returns information about a specification's version tag: a
// snapshot of the specification at a point in time.
func (s *Service) VersionTag(ctx context.Context, specID, tagID string) (*VersionTagResult, error) {
	params := api.GetSpecVersionTagParams{SpecId: specID, TagId: tagID}

	res, err := s.api.GetSpecVersionTag(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetSpecVersionTag:
		out := &VersionTagResult{}
		for _, d := range r.Data {
			out.Entries = append(out.Entries, VersionTagEntry{
				ID:        d.ID.Or(""),
				Name:      d.Name.Or(""),
				Type:      VersionTagEntryType(d.Type.Or("")),
				Path:      d.Path.Or(""),
				FileType:  FileType(d.FileType.Or("")),
				Content:   d.Content.Or(""),
				ParentID:  d.ParentId.Or(""),
				CreatedBy: d.CreatedBy.Or(""),
				UpdatedBy: d.UpdatedBy.Or(""),
				CreatedAt: d.CreatedAt.Or(""),
				UpdatedAt: d.UpdatedAt.Or(""),
			})
		}
		return out, nil
	case *api.GetSpecVersionTagUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetSpecVersionTagNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- VersionTags --------------------------------------------------------------

// VersionTagsInput holds the pagination options for VersionTags.
type VersionTagsInput struct {
	// Cursor is the pagination cursor (use VersionTagsResult.NextCursor to page).
	Cursor string
	// Limit is the maximum number of rows to return.
	Limit int
}

// VersionTagsResult is the paginated result of VersionTags.
type VersionTagsResult struct {
	Tags       []VersionTagSummary
	NextCursor string
}

// VersionTagSummary is a summary of a specification version tag.
type VersionTagSummary struct {
	ID   string
	Name string
}

// VersionTags returns a list of a specification's version tags.
func (s *Service) VersionTags(ctx context.Context, specID string, in *VersionTagsInput) (*VersionTagsResult, error) {
	if in == nil {
		in = &VersionTagsInput{}
	}
	params := api.GetSpecVersionTagsParams{SpecId: specID}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}

	res, err := s.api.GetSpecVersionTags(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetSpecVersionTags:
		out := &VersionTagsResult{}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		for _, d := range r.Data {
			out.Tags = append(out.Tags, VersionTagSummary{
				ID:   d.ID.Or(""),
				Name: d.Name.Or(""),
			})
		}
		return out, nil
	case *api.GetSpecVersionTagsUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetSpecVersionTagsNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- CreateVersionTag --------------------------------------------------------------

// CreateVersionTagInput holds the fields for creating a specification
// version tag.
type CreateVersionTagInput struct {
	// Name is the version tag's name.
	Name string
}

// CreateVersionTag creates a version tag for a specification. Version tags
// are snapshots of a specification at a point in time that let you track
// changes over time.
//
// Conflicts (409) can occur if a version tag already exists for the current
// changelog group; make new changes to the specification to create a new
// changelog group before retrying.
func (s *Service) CreateVersionTag(ctx context.Context, specID string, in *CreateVersionTagInput) (*VersionTagSummary, error) {
	if in == nil {
		in = &CreateVersionTagInput{}
	}
	req := &api.CreateSpecVersionTag{Name: in.Name}
	params := api.CreateSpecVersionTagParams{SpecId: specID}

	res, err := s.api.CreateSpecVersionTag(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateSpecVersionTagResponse:
		out := &VersionTagSummary{}
		if data, ok := r.Data.Get(); ok {
			out.ID = data.ID.Or("")
			out.Name = data.Name.Or("")
		}
		return out, nil
	case *api.CreateSpecVersionTagBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.CreateSpecVersionTagUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.ApiSpec403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.CreateSpecVersionTagNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.CreateSpecVersionTagConflict:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusConflict)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- error helpers ----------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}

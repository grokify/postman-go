# Specs

The API Spec Hub stores API specifications (OpenAPI, AsyncAPI, protobuf,
GraphQL, and Smithy), their files, and the collections generated from (or
synced with) them. It also tracks specification version tags, which are
snapshots of a specification at a point in time. Reachable via
`client.Specs()`.

!!! note "Known API limitations"
    Two endpoints degrade gracefully because Postman's published schema
    can't express their response bodies as concrete types:

    - `Definition` only confirms the request succeeded — Postman types the
      response as free-form (`any`) in its TypeScript SDK, so the generated
      client discards the body and the definition's contents aren't
      retrievable through this method.
    - `TaskStatus` returns raw JSON (`TaskStatusResult.Raw`) instead of typed
      fields, since the response is a TypeScript union the OpenAPI
      reconstruction can't resolve — inspect its `"status"` field
      (`"processing"`, `"completed"`, or `"failed"`).

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/specs"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List specifications in a workspace.
list, err := client.Specs().List(ctx, &specs.ListInput{WorkspaceID: "workspace-id"})
for _, sp := range list.Specs {
	fmt.Println(sp.ID, sp.Name, sp.Type)
}

// Fetch one spec's details.
spec, err := client.Specs().Get(ctx, list.Specs[0].ID)

// Generate a collection from it.
task, err := client.Specs().GenerateCollection(ctx, spec.ID, &specs.GenerateCollectionInput{
	Name: spec.Name + " Collection",
})
status, err := client.Specs().TaskStatus(ctx, specs.TaskElementTypeCollections, spec.ID, task.TaskID)
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx, *ListInput)` | List all API specifications in a workspace. Paginated. |
| `Create(ctx, *CreateInput)` | Create an API specification (single or multi-file) in a workspace. |
| `Get(ctx, specID)` | Return information about a specification. |
| `UpdateProperties(ctx, specID, *UpdatePropertiesInput)` | Update a specification's properties, such as its name. |
| `Delete(ctx, specID)` | Delete a specification. |
| `Definition(ctx, specID)` | Confirm a specification's definition was retrieved (see note above). |
| `Files(ctx, specID)` | List all files in a specification. |
| `CreateFile(ctx, specID, *CreateFileInput)` | Create a file in a specification. |
| `File(ctx, specID, filePath)` | Return a specification file's contents. |
| `UpdateFile(ctx, specID, filePath, *UpdateFileInput)` | Update a specification file's name, type, or content. |
| `DeleteFile(ctx, specID, filePath)` | Delete a specification file. |
| `Collections(ctx, specID, *CollectionsInput)` | List the collections generated from a specification. Paginated. |
| `GenerateCollection(ctx, specID, *GenerateCollectionInput)` | Generate a collection from an OpenAPI or Smithy specification. Async. |
| `SyncWithCollection(ctx, specID, collectionUID)` | Sync a specification with a collection generated from it. Async. |
| `UpdateSyncOptions(ctx, specID, collectionID, *SyncOptionsInput)` | Update sync options for a spec's generated collection. |
| `CollectionSpecs(ctx, collectionUID)` | List the specifications generated for a collection. |
| `GenerateFromCollection(ctx, collectionUID, *GenerateFromCollectionInput)` | Generate an OpenAPI specification from a collection. Async. |
| `SyncCollectionWithSpec(ctx, collectionUID, specID)` | Sync a collection generated from a specification. Async. |
| `TaskStatus(ctx, elementType, elementID, taskID)` | Poll the status of an async generate/sync task (see note above). |
| `VersionTag(ctx, specID, tagID)` | Return a specification version tag's snapshot entries. |
| `VersionTags(ctx, specID, *VersionTagsInput)` | List a specification's version tags. Paginated. |
| `CreateVersionTag(ctx, specID, *CreateVersionTagInput)` | Create a version tag (a point-in-time snapshot) for a specification. |

### Create, Get, UpdateProperties, Delete

```go
created, err := client.Specs().Create(ctx, &specs.CreateInput{
	WorkspaceID: "workspace-id",
	Name:        "Petstore API",
	Type:        specs.SpecTypeOpenAPI31,
	Files: []specs.SpecFileInput{
		{Path: "index.yaml", Content: openapiYAML},
	},
})

spec, err := client.Specs().Get(ctx, created.ID)

updated, err := client.Specs().UpdateProperties(ctx, created.ID, &specs.UpdatePropertiesInput{
	Name: "Petstore API v2",
})

err = client.Specs().Delete(ctx, created.ID)
```

Multi-file specifications must set `Type` on every file, with exactly one
`specs.FileTypeRoot`; single-file specifications should leave `Type` empty.
Supported `SpecType` values include `SpecTypeOpenAPI20/30/31`,
`SpecTypeAsyncAPI20/30`, `SpecTypeProtobuf2/3`, `SpecTypeGraphQL`, and
`SpecTypeSmithy20`.

### Files: Files, CreateFile, File, UpdateFile, DeleteFile

```go
files, err := client.Specs().Files(ctx, "spec-id")

file, err := client.Specs().CreateFile(ctx, "spec-id", &specs.CreateFileInput{
	Path:    "schemas/pet.yaml",
	Content: schemaYAML,
})

content, err := client.Specs().File(ctx, "spec-id", "schemas/pet.yaml")

_, err = client.Specs().UpdateFile(ctx, "spec-id", "schemas/pet.yaml", &specs.UpdateFileInput{
	Content: updatedSchemaYAML,
})

err = client.Specs().DeleteFile(ctx, "spec-id", "schemas/pet.yaml")
```

`CreateFile`/`UpdateFile` support OpenAPI and protobuf 2/3 specifications;
files cannot exceed 10 MB. A `Path` containing `/` creates a folder.
`UpdateFile` does not accept multiple properties in one call: set exactly
one of `Name`, `Type`, or `Content`. Setting `Type` to `specs.FileTypeRoot`
demotes the previous root file to `specs.FileTypeDefault`.

### Definition

```go
err := client.Specs().Definition(ctx, "spec-id")
// err == nil means the definition was retrieved successfully server-side;
// its contents are not returned (see the note above).
```

### Collection generation and sync: Collections, GenerateCollection, SyncWithCollection, UpdateSyncOptions, CollectionSpecs, GenerateFromCollection, SyncCollectionWithSpec, TaskStatus

```go
// Spec -> generated collections.
list, err := client.Specs().Collections(ctx, "spec-id", &specs.CollectionsInput{Limit: 25})

task, err := client.Specs().GenerateCollection(ctx, "spec-id", &specs.GenerateCollectionInput{
	Name:              "Petstore API",
	RequestNameSource: specs.RequestNameSourceURL,
	FolderStrategy:    specs.FolderStrategyTags,
	IndentCharacter:   specs.IndentCharacterSpace,
})

syncTask, err := client.Specs().SyncWithCollection(ctx, "spec-id", "collection-uid")

opts, err := client.Specs().UpdateSyncOptions(ctx, "spec-id", "collection-id", &specs.SyncOptionsInput{
	SyncExamples:           true,
	DeleteOrphanedRequests: true,
})

// Collection -> generated specs (the reverse direction).
generated, err := client.Specs().CollectionSpecs(ctx, "collection-uid")

genTask, err := client.Specs().GenerateFromCollection(ctx, "collection-uid", &specs.GenerateFromCollectionInput{
	Name:   "Petstore API (from collection)",
	Type:   specs.SpecTypeOpenAPI30,
	Format: specs.SpecFormatYAML,
})

syncTask2, err := client.Specs().SyncCollectionWithSpec(ctx, "collection-uid", "spec-id")

// Poll any of the above async tasks.
status, err := client.Specs().TaskStatus(ctx, specs.TaskElementTypeCollections, "spec-id", task.TaskID)
```

`GenerateCollection` and `GenerateFromCollection` only support OpenAPI
2.0/3.0/3.1 (plus Smithy for `GenerateCollection`) and return a `TaskResult`
with a polling link; use `TaskStatus` to check on it. `SyncWithCollection`
and `SyncCollectionWithSpec` only support OpenAPI 2.0/3.0/3.1 specifications
and only sync collections generated from the given specification.

### Version tags: VersionTag, VersionTags, CreateVersionTag

```go
tags, err := client.Specs().VersionTags(ctx, "spec-id", &specs.VersionTagsInput{Limit: 25})

tag, err := client.Specs().CreateVersionTag(ctx, "spec-id", &specs.CreateVersionTagInput{Name: "v1.0.0"})

snapshot, err := client.Specs().VersionTag(ctx, "spec-id", tag.ID)
for _, entry := range snapshot.Entries {
	fmt.Println(entry.Path, entry.Type) // specs.VersionTagEntryTypeFile or ...Folder
}
```

`CreateVersionTag` can return a 409 conflict if a version tag already exists
for the current changelog group; make new changes to the specification to
create a new changelog group before retrying.

## Reference

Source: [`specs/`](https://github.com/grokify/postman-go/tree/main/specs) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/specs)

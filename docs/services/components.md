# Components

Components are reusable pieces of an API definition (schemas, parameters,
responses, and so on) that live in a team's component library, part of
Postman's API Spec Hub. A component has a mutable draft and zero or more
immutable published versions. Reachable via `client.Components()`.

!!! note "Update is limited by the upstream API"
    Postman's OpenAPI spec models the `Update` request body as an untyped
    union (a name-only patch or a status-only patch) that the generated
    client cannot resolve to concrete fields, so it always sends an empty
    JSON object as the body. `Update` therefore cannot presently rename a
    component or change its status; it still performs the request and
    surfaces the component's current state (or any error) from the API.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/components"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List active components, including their latest published version.
list, err := client.Components().List(ctx, &components.ListInput{
	Status:  components.ComponentStatusActive,
	Include: []components.ComponentInclude{components.ComponentIncludeLatestVersion},
})
for _, c := range list.Components {
	fmt.Println(c.ID, c.Name, c.Type)
}

// Create a component and publish its first version.
created, err := client.Components().Create(ctx, &components.CreateInput{
	Name:   "shared-error-schema",
	Type:   components.ComponentTypeOAS3,
	Format: components.ContentFormatJSON,
	Content: `{"type":"object","properties":{"message":{"type":"string"}}}`,
})

version, err := client.Components().CreateVersion(ctx, created.ID, &components.CreateVersionInput{
	Label: "1.0.0",
})
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx, *ListInput)` | List all components in the team's component library. |
| `Create(ctx, *CreateInput)` | Create a new component with an initial draft. |
| `Get(ctx, componentID, *GetInput)` | Get information about a component. |
| `Update(ctx, componentID)` | Rename a component or change its status. See the limitation note above. |
| `Draft(ctx, componentID)` | Get a component's current working draft. |
| `UpdateDraft(ctx, componentID, *UpdateDraftInput)` | Update a component's draft content. |
| `Versions(ctx, componentID, *VersionsInput)` | List a component's published versions. |
| `CreateVersion(ctx, componentID, *CreateVersionInput)` | Publish a new version from the component's current draft. |
| `Version(ctx, componentID, versionID, *VersionInput)` | Get a single published version. |

### List

```go
list, err := client.Components().List(ctx, &components.ListInput{
	Type:        components.ComponentTypeOAS3,
	Status:      components.ComponentStatusActive,
	HasVersions: true,
	Include:     []components.ComponentInclude{components.ComponentIncludeLatestVersion},
})
for _, c := range list.Components {
	fmt.Println(c.ID, c.Name, c.Status)
}
```

A `nil` input returns all components regardless of type or status.

### Create

```go
created, err := client.Components().Create(ctx, &components.CreateInput{
	Name:    "shared-error-schema",
	Type:    components.ComponentTypeOAS3,
	Format:  components.ContentFormatJSON,
	Content: `{"type":"object","properties":{"message":{"type":"string"}}}`,
})
// created.ID
```

`Name` must be unique within the team and up to 60 characters (letters,
digits, hyphens, underscores, and periods). `Content` is limited to 500 KB.

### Get

```go
c, err := client.Components().Get(ctx, "COMPONENT_ID", &components.GetInput{
	Include: []components.ComponentInclude{components.ComponentIncludeLatestVersion},
})
// c.Name, c.Status, c.LatestVersion
```

### Update

```go
updated, err := client.Components().Update(ctx, "COMPONENT_ID")
// updated.ID, updated.Name, updated.Status
```

See the note above — this currently reflects the component's existing
state rather than applying a rename or status change.

### Draft

```go
draft, err := client.Components().Draft(ctx, "COMPONENT_ID")
// draft.Content, draft.Format
```

### UpdateDraft

```go
_, err := client.Components().UpdateDraft(ctx, "COMPONENT_ID", &components.UpdateDraftInput{
	Content: `{"type":"object","properties":{"message":{"type":"string"},"code":{"type":"integer"}}}`,
	Format:  components.ContentFormatJSON,
})
```

Archived components can't be updated.

### Versions

```go
versions, err := client.Components().Versions(ctx, "COMPONENT_ID", &components.VersionsInput{
	Include: []components.VersionInclude{components.VersionIncludeContent},
})
for _, v := range versions.Versions {
	fmt.Println(v.ID, v.Label, v.PublishedAt)
}
```

### CreateVersion

```go
version, err := client.Components().CreateVersion(ctx, "COMPONENT_ID", &components.CreateVersionInput{
	Label:  "1.0.0",
	Source: components.SourceTypeDraft,
})
// version.ID
```

`Source` defaults to the component's current draft. Archived components
must be reactivated before publishing.

### Version

```go
version, err := client.Components().Version(ctx, "COMPONENT_ID", "VERSION_ID", &components.VersionInput{
	Include: []components.VersionInclude{components.VersionIncludeContent},
})
// version.Label, version.URL, version.Content
```

## Reference

Source: [`components/`](https://github.com/grokify/postman-go/tree/main/components) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/components)

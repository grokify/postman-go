# Imports

The Imports service imports an OpenAPI definition into Postman as a new
collection. Reachable via `client.Imports()`.

!!! note "Rate limit and file input"
    `FromOpenAPI` has a rate limit of 10 requests per 10 seconds. The
    Postman web app does not support the `"file"` input method type; only
    JSON and stringified definitions are supported here.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/imports"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

result, err := client.Imports().FromOpenAPI(ctx, &imports.FromOpenAPIInput{
	WorkspaceID: "ws-id",
	Type:        imports.InputTypeString,
	Input:       openAPIYAMLOrJSONString,
})
for _, c := range result.Collections {
	fmt.Println(c.ID, c.Name)
}
```

## Methods

| Method | Description |
|--------|-------------|
| `FromOpenAPI(ctx, *FromOpenAPIInput)` | Import an OpenAPI definition into Postman as a new collection. |

### FromOpenAPI

```go
result, err := client.Imports().FromOpenAPI(ctx, &imports.FromOpenAPIInput{
	WorkspaceID: "ws-id",
	Type:        imports.InputTypeJSON,
	Input:       openAPIDefinitionAsMapOrStruct, // JSON-marshalable when Type is InputTypeJSON
	Options: &imports.CollectionOptions{
		RequestNameSource: imports.RequestNameSourceFallback,
		FolderStrategy:    imports.FolderStrategyTags,
		IndentCharacter:   imports.IndentCharacterSpace,
	},
})
// result.Collections []ImportedCollection, each with ID, Name, UID
```

If `WorkspaceID` is empty, Postman imports into the oldest personal Internal
workspace you own. `Type` selects whether `Input` is a JSON-marshalable
value (`InputTypeJSON`) or a stringified OpenAPI definition
(`InputTypeString`); `Options` is optional and controls request naming,
indentation, folder strategy, and other advanced collection-creation
behavior (field names are case-sensitive on the API side).

## Reference

Source: [`imports/`](https://github.com/grokify/postman-go/tree/main/imports) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/imports)

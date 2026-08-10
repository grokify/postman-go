# Search

The Search service finds Postman resources (workspaces, collections,
requests, environments, flows, and specs) by query text and structured
filters. Reachable via `client.Search()`.

!!! note "Unauthenticated requests"
    If called without an API key, the response only returns
    publicly-available resources.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/search"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

result, err := client.Search().Query(ctx, &search.QueryInput{
	Q:           "auth",
	ElementType: search.ElementTypeCollections,
	Ownership:   search.OwnershipOrganization,
	Limit:       25,
})
for _, r := range result.Results {
	fmt.Println(r.Name, r.URL)
}
```

## Methods

| Method | Description |
|--------|-------------|
| `Query(ctx, *QueryInput)` | Search Postman for resources, filtered by query text, ownership, and structured filters. Paginated. |

### Query

```go
isPrivate := false
result, err := client.Search().Query(ctx, &search.QueryInput{
	Q:           "auth",
	ElementType: search.ElementTypeRequests,
	Ownership:   search.OwnershipAll,
	Filters: []search.Filter{
		{Field: search.FilterFieldMethod, In: []string{"GET", "POST"}},
		{Field: search.FilterFieldVisibility, Eq: string(search.VisibilityPublic)},
		{Field: search.FilterFieldPrivateNetwork, BoolEq: &isPrivate},
	},
	Limit: 50,
})
// result.Results []Result, result.NextCursor, result.Total
// Page through with: Cursor: result.NextCursor
```

`ElementType` is required. `Filters` are ANDed together; each `Filter` sets
`Field` plus the comparison(s) that apply to it:

| Field kind | Fields | Comparisons |
|------------|--------|-------------|
| Boolean | `FilterFieldPrivateNetwork`, `FilterFieldPublisherIsVerified`, `FilterFieldIsGitConnected` | `BoolEq` / `BoolNe` |
| Enum (Eq/Ne only) | `FilterFieldVisibility` (value is a `Visibility`: `internal`, `public`, `partner`) | `Eq` / `Ne` |
| String | `FilterFieldWorkspaceID`, `FilterFieldCollectionID`, `FilterFieldTags`, `FilterFieldMethod`, `FilterFieldRequestID`, `FilterFieldSpecificationID`, `FilterFieldFlowID`, `FilterFieldEnvironmentID`, `FilterFieldCreatedBy`, `FilterFieldOrganizationID`, `FilterFieldTeamID`, `FilterFieldType` | `Eq` / `Ne` / `In` / `Nin` |

Each `Result` includes optional `Team`, `Collection`, `Workspace`,
`Organization`, and `Links` sub-structs that are only populated when
relevant to the matched resource (e.g. `Collection` is set only for request
results).

## Reference

Source: [`search/`](https://github.com/grokify/postman-go/tree/main/search) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/search)

# Secret Scanner

The Secret Scanner detects secrets (API keys, tokens, and other credentials)
exposed in a team's Postman workspaces. Reachable via `client.SecretScanner()`.

!!! note "Requires an add-on"
    The Secret Scanner API requires a Postman Enterprise plan with the
    Advanced Security Administration add-on enabled — see
    [Authentication](../guides/authentication.md#enterprise-add-ons).

## Quick Example

```go
import (
	"context"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/secretscanner"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List supported secret types.
types, err := client.SecretScanner().SecretTypes(ctx)

// Query detected secrets.
result, err := client.SecretScanner().Query(ctx, &secretscanner.QueryInput{
	Limit:        50,
	IncludeTotal: true,
})
for _, s := range result.Secrets {
	fmt.Printf("%s in %s: %s\n", s.SecretType, s.WorkspaceID, s.Resolution)
}
```

## Methods

| Method | Description |
|--------|-------------|
| `Query(ctx, *QueryInput)` | Query detected secrets, filtered by resolution, secret type, workspace, or resource. Paginated. |
| `UpdateResolution(ctx, secretID, *UpdateResolutionInput)` | Mark a detected secret as `FALSE_POSITIVE`, `REVOKED`, or `ACCEPTED_RISK`. |
| `Locations(ctx, secretID, *LocationsInput)` | Find every location (collection, request, environment, ...) where a specific secret was found. |
| `SecretTypes(ctx)` | List the secret types the scanner supports, including custom team regexes. |

### Query

```go
result, err := client.SecretScanner().Query(ctx, &secretscanner.QueryInput{
	Resolved:     false,                                      // only unresolved secrets
	Statuses:     []secretscanner.Resolution{secretscanner.ResolutionActive},
	WorkspaceIDs: []string{"workspace-id"},                    // mutually exclusive with Resources
	Limit:        50,
	IncludeTotal: true,
})
// result.Secrets []DetectedSecret, result.NextCursor, result.Total
```

A `nil` input returns all results. `Resources` and `WorkspaceIDs` are
mutually exclusive filters.

### UpdateResolution

```go
_, err := client.SecretScanner().UpdateResolution(ctx, "SECRET_ID",
	&secretscanner.UpdateResolutionInput{
		Resolution:  secretscanner.ResolutionFalsePositive,
		WorkspaceID: "WORKSPACE_ID",
	})
```

`Resolution` accepts `ResolutionFalsePositive`, `ResolutionRevoked`, or
`ResolutionAcceptedRisk` — `ResolutionActive` cannot be set directly.

### Locations

```go
locations, err := client.SecretScanner().Locations(ctx, "SECRET_ID",
	&secretscanner.LocationsInput{WorkspaceID: "WORKSPACE_ID"})
for _, loc := range locations.Locations {
	fmt.Println(loc.Location, loc.URL, loc.ResourceType)
}
```

### SecretTypes

```go
types, err := client.SecretScanner().SecretTypes(ctx)
for _, t := range types.Types {
	fmt.Println(t.ID, t.Name, t.Origin) // Origin: DEFAULT or TEAM_REGEX
}
```

## Reference

Source: [`secretscanner/`](https://github.com/grokify/postman-go/tree/main/secretscanner) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/secretscanner)

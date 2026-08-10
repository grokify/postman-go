# Quick Start

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/collections"
)

func main() {
	// Reads POSTMAN_API_KEY if WithAPIKey isn't passed.
	client, err := postman.NewClient(postman.WithAPIKey("PMAK-..."))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// List workspaces you have access to.
	workspaces, err := client.Workspaces().List(ctx, nil)
	if err != nil {
		var apiErr *postman.APIError
		if errors.As(err, &apiErr) {
			log.Fatalf("Postman API error (status %d): %s", apiErr.StatusCode, apiErr.Title)
		}
		log.Fatal(err)
	}
	for _, w := range workspaces.Workspaces {
		fmt.Printf("%s  %s  (%s)\n", w.ID, w.Name, w.Type)
	}

	// List collections in the first workspace.
	if len(workspaces.Workspaces) > 0 {
		result, err := client.Collections().List(ctx, &collections.ListInput{
			Workspace: workspaces.Workspaces[0].ID,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d collections in %s\n", len(result.Collections), workspaces.Workspaces[0].Name)
	}
}
```

!!! note
    Every service's `Service` type is reachable from `*postman.Client` (e.g.
    `client.Collections()`), but its input/result types and enums live in that
    service's own subpackage — `collections.ListInput` above, not something
    off the root `postman` package. See each service's own page under
    [Services](../services/analytics.md) for its exact types.

## Next Steps

- [Authentication](../guides/authentication.md) — API keys, EU region, environment variables
- [Error Handling](../guides/errors.md) — `*postman.APIError` and status-code mapping
- [Rate Limiting](../guides/rate-limiting.md) — Postman's request limits and how to handle 429s
- Browse the [Services](../services/analytics.md) section for any of the 31 service packages
- [API Reference](../api-reference.html) — full endpoint-level reference generated from the OpenAPI spec

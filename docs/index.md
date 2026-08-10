# postman-go

postman-go is a Go SDK for the [Postman API](https://learning.postman.com/api-docs/api-reference/),
covering all 31 Postman API service areas (196 endpoints) through a single
`*postman.Client`.

Postman does not publish a downloadable OpenAPI document. The SDK's
[`openapi/openapi.yaml`](https://github.com/grokify/postman-go/blob/main/openapi/openapi.yaml)
is reconstructed from Postman's official
[TypeScript SDK](https://github.com/postmanlabs/postman-api-sdk-ts) and drives
an [ogen](https://github.com/ogen-go/ogen)-generated internal client, with
hand-written, domain-oriented wrapper packages on top — one per service area.

## Features

- **One client, every service** — `client.Collections()`, `client.Workspaces()`,
  `client.SecretScanner()`, and 28 more, all off a single `*postman.Client`.
- **Plain Go types** — no generated `Opt*` wrappers or raw JSON leak into the
  public API; every service returns plain structs and typed enums.
- **Typed errors** — non-2xx responses come back as `*postman.APIError`,
  carrying the RFC 9457 problem-details fields Postman returns.
- **Generated where it should be, hand-written where it matters** — the
  low-level HTTP/JSON plumbing is generated and never hand-edited; the public
  API is hand-written and reviewed like any other Go code.

## Package Structure

| Package | Description |
|---------|-------------|
| root (`postman`) | Top-level `Client`, auth, options |
| `postmanerr` | Shared error type (`APIError`) and mapping helpers |
| `internal/api` | ogen-generated client (not part of the public API) |
| `secretscanner`, `collections`, `workspaces`, ... | One package per Postman service area — see [Services](services/analytics.md) |

## Quick Example

```go
package main

import (
	"context"
	"fmt"
	"log"

	postman "github.com/grokify/postman-go"
)

func main() {
	client, err := postman.NewClient(postman.WithAPIKey("PMAK-..."))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	workspaces, err := client.Workspaces().List(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, w := range workspaces.Workspaces {
		fmt.Println(w.ID, w.Name)
	}
}
```

See [Getting Started](getting-started/installation.md) to install and
authenticate, or jump straight to a [service page](services/analytics.md) for
usage examples.

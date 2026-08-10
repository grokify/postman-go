# postman-go

A Go SDK for the [Postman API](https://learning.postman.com/api-docs/api-reference/).

This SDK follows the standard grokify SDK pipeline: an OpenAPI 3.0 specification
drives an internal client generated with [ogen](https://github.com/ogen-go/ogen),
with hand-written, domain-oriented wrapper packages on top. Postman does not
publish a downloadable OpenAPI document, so the spec in [`openapi/openapi.yaml`](openapi/openapi.yaml)
is reconstructed and maintained in-repo by [`scripts/gen-openapi`](scripts/gen-openapi),
which parses Postman's official [TypeScript SDK](https://github.com/postmanlabs/postman-api-sdk-ts)
(the authoritative machine-readable source, in the absence of a published spec)
and the public API reference.

## Status

All 31 Postman API service areas (196 endpoints) have a hand-written Go wrapper
package with plain Go types, typed errors, and unit tests.

| Service | Package | Endpoints |
|---|---|---|
| Analytics | [`analytics`](analytics/) | 2 |
| API Security | [`apisecurity`](apisecurity/) | 1 |
| Audit Logs | [`auditlogs`](auditlogs/) | 2 |
| Billing | [`billing`](billing/) | 2 |
| Collection Access Keys | [`collectionaccesskeys`](collectionaccesskeys/) | 2 |
| Collection Folders (comments) | [`collectionfolders`](collectionfolders/) | 4 |
| Collection Items | [`collectionitems`](collectionitems/) | 12 |
| Collection Requests (comments) | [`collectionrequests`](collectionrequests/) | 4 |
| Collection Responses (comments) | [`collectionresponses`](collectionresponses/) | 4 |
| Collections | [`collections`](collections/) | 31 |
| Comments | [`comments`](comments/) | 1 |
| Components (API Spec Hub) | [`components`](components/) | 9 |
| Environments | [`environments`](environments/) | 10 |
| Groups | [`groups`](groups/) | 2 |
| Import | [`imports`](imports/) | 1 |
| Mocks | [`mocks`](mocks/) | 13 |
| Monitors | [`monitors`](monitors/) | 8 |
| OAuth 2.0 | [`oauth2`](oauth2/) | 2 |
| Postbot | [`postbot`](postbot/) | 1 |
| Private API Network | [`privateapinetwork`](privateapinetwork/) | 6 |
| Pull Requests | [`pullrequests`](pullrequests/) | 3 |
| SDK Generation | [`sdkgen`](sdkgen/) | 10 |
| Search | [`search`](search/) | 1 |
| Secret Scanner | [`secretscanner`](secretscanner/) | 4 |
| Service Accounts | [`serviceaccounts`](serviceaccounts/) | 1 |
| Specs (API Spec Hub) | [`specs`](specs/) | 22 |
| Tags | [`tags`](tags/) | 5 |
| Teams | [`teams`](teams/) | 10 |
| Users | [`users`](users/) | 3 |
| Webhooks | [`webhooks`](webhooks/) | 1 |
| Workspaces | [`workspaces`](workspaces/) | 19 |

Each service is reachable from a single `*postman.Client` — see [Usage](#usage).

## Installation

```bash
go get github.com/grokify/postman-go
```

## Authentication

All requests authenticate with a [Postman API key](https://learning.postman.com/docs/developer/postman-api/authentication/)
sent in the `X-API-Key` header. Provide it via `WithAPIKey` or the
`POSTMAN_API_KEY` environment variable.

> **Note:** Some services (e.g. Secret Scanner, Private API Network) require a
> Postman Enterprise plan with the relevant add-on enabled.

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/secretscanner"
)

func main() {
	client, err := postman.NewClient(postman.WithAPIKey("PMAK-..."))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// List supported secret types.
	types, err := client.SecretScanner().SecretTypes(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d secret types supported\n", types.Total)

	// Query detected secrets.
	result, err := client.SecretScanner().Query(ctx, &secretscanner.QueryInput{
		Limit:        50,
		IncludeTotal: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range result.Secrets {
		fmt.Printf("%s in %s: %s\n", s.SecretType, s.WorkspaceID, s.Resolution)
	}

	// Mark a secret as a false positive.
	_, err = client.SecretScanner().UpdateResolution(ctx, "SECRET_ID",
		&secretscanner.UpdateResolutionInput{
			Resolution:  secretscanner.ResolutionFalsePositive,
			WorkspaceID: "WORKSPACE_ID",
		})
	if err != nil {
		log.Fatal(err)
	}

	// Every other service follows the same shape, e.g.:
	workspaces, err := client.Workspaces().List(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	_ = workspaces
}
```

See [`examples/`](examples/) for runnable programs.

## Error handling

Non-2xx responses are returned as `*postman.APIError` (an alias for
`postmanerr.APIError`), carrying the RFC 9457 problem-details fields Postman
returns:

```go
result, err := client.SecretScanner().Query(ctx, nil)
if err != nil {
	var apiErr *postman.APIError
	if errors.As(err, &apiErr) {
		fmt.Println(apiErr.StatusCode, apiErr.Title, apiErr.Detail)
	}
}
```

Some Postman error responses are typed unions the source TypeScript SDK cannot
resolve to a concrete shape; in those cases only the HTTP status code is
recoverable (no title/detail body). This is documented per-method where it
applies — see `scripts/gen-openapi/README.md`'s "Known approximations".

## Regenerating the SDK

The OpenAPI spec is reconstructed from the Postman TypeScript SDK, and the
internal client is generated from that spec:

```bash
# 1. Regenerate openapi/openapi.yaml from the TS SDK
node scripts/gen-openapi/index.mjs

# 2. Regenerate internal/api/ from openapi/openapi.yaml
./generate.sh
```

See [`scripts/gen-openapi/README.md`](scripts/gen-openapi/README.md) for how
the reconstruction works and its documented approximations. The hand-written
service packages (`secretscanner/`, `collections/`, etc.) are not generated and
may need updates if the spec's operation IDs or schemas change.

## Layout

```
postman-go/
├── client.go              # top-level Client, options, auth, service accessors
├── openapi/openapi.yaml   # reconstructed OpenAPI 3.0 spec (source of truth)
├── scripts/gen-openapi/   # TS SDK -> OpenAPI spec generator
├── internal/api/          # ogen-generated client (do not edit)
├── postmanerr/            # shared error types (leaf package)
├── secretscanner/         # one wrapper package per Postman service area
├── collections/
├── workspaces/
├── ...
└── examples/              # runnable examples
```

## License

[MIT](LICENSE)

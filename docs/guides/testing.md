# Testing

The SDK has two kinds of tests:

- **Unit tests** run offline against a local `httptest` server. They are the
  default and need no API key.
- **Integration tests** call the live Postman API. They run only when
  `POSTMAN_API_KEY` is set.

## Unit Tests

```bash
go test ./...
```

Each service package tests its wrapper against a local HTTP server that
returns canned JSON. This checks request paths, query parameters, and how
responses decode into Go types, without touching the network.

Some unit tests replay response shapes seen from the live API that the
TypeScript-derived spec got wrong, for example a `null` cursor or a fractional
usage value. They exist so that a future spec regeneration can't quietly
bring the bug back. See
[Regenerating the SDK](https://github.com/grokify/postman-go#regenerating-the-sdk).

## Integration Tests

Integration tests live in the root package and call the real API with your
key:

```bash
export POSTMAN_API_KEY="PMAK-..."
go test -run Integration -v .
```

They cover:

| Test | What it calls |
|------|---------------|
| `TestIntegrationMe` | `Users().Me` |
| `TestIntegrationWorkspacesList` | `Workspaces().List` |
| `TestIntegrationCollectionsList` | `Collections().List` |
| `TestIntegrationTeamsGet` | `Users().Me`, then `Teams().Get` with your team ID |

The tests only read data. They never create, change, or delete anything.

Without `POSTMAN_API_KEY`, each one is skipped, so `go test ./...` stays
offline in CI and on a fresh checkout.

!!! note "Plan-dependent results"
    Some plans have no Teams API access. `TestIntegrationTeamsGet` then gets a
    well-formed `400` error instead of team data. That still passes: the test
    checks that the SDK sends the right request and decodes the reply, not
    which plan you have.

## Testing Your Own Code

To unit test code that uses the SDK, point the client at a local server with
`WithBaseURL`:

```go
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"workspaces": [{"id": "w1", "name": "Demo"}]}`))
}))
defer srv.Close()

client, err := postman.NewClient(
	postman.WithAPIKey("test-key"),
	postman.WithBaseURL(srv.URL),
)
if err != nil {
	t.Fatal(err)
}
```

Return JSON that matches the real API's shape. The generated decoder is
strict, so a wrong type or a missing required field fails the call just like
it would in production.

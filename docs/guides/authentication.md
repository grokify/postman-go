# Authentication

Every request authenticates with a [Postman API key](https://learning.postman.com/docs/developer/postman-api/authentication/)
sent in the `X-API-Key` header. The client attaches it for you — you never
set the header yourself.

## Passing the API Key

```go
client, err := postman.NewClient(postman.WithAPIKey("PMAK-..."))
```

## Environment Variable

If `WithAPIKey` isn't passed, `NewClient` falls back to the
`POSTMAN_API_KEY` environment variable (`postman.EnvAPIKey`):

```bash
export POSTMAN_API_KEY="PMAK-..."
```

```go
client, err := postman.NewClient() // reads POSTMAN_API_KEY
```

If neither is set, `NewClient` returns `postman.ErrNoAPIKey`.

## EU Data Residency

Postman workspaces provisioned in the EU region are served from a different
API host. Point the client at it with `WithBaseURL`:

```go
client, err := postman.NewClient(
	postman.WithAPIKey("PMAK-..."),
	postman.WithBaseURL(postman.EUBaseURL), // https://api.eu.postman.com
)
```

`postman.DefaultBaseURL` (`https://api.postman.com`) is used otherwise.

## Enterprise Add-Ons

Some services require a Postman Enterprise plan with the relevant add-on
enabled — the [Secret Scanner](../services/secretscanner.md) requires
Advanced Security Administration, and [Private API Network](../services/privateapinetwork.md)
requires the Private API Network add-on. Calling those services without the
add-on returns a `403 Forbidden` `*postman.APIError` — see
[Error Handling](errors.md).

## Custom HTTP Client / Timeout

```go
client, err := postman.NewClient(
	postman.WithAPIKey("PMAK-..."),
	postman.WithHTTPClient(myHTTPClient), // takes precedence over WithTimeout
)

// or, without a custom client:
client, err := postman.NewClient(
	postman.WithAPIKey("PMAK-..."),
	postman.WithTimeout(60 * time.Second), // default is 30s
)
```

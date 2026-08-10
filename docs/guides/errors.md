# Error Handling

Every service method returns a plain `error`. Network/transport failures come
back as-is (wrapped `net`/`http` errors); non-2xx API responses come back as
`*postman.APIError`.

## `*postman.APIError`

`postman.APIError` is an alias for `postmanerr.APIError`, so you only need one
type to check regardless of which service produced the error:

```go
type APIError struct {
	StatusCode int    // HTTP status code
	Type       string // RFC 9457 problem "type" URI
	Title      string // short, human-readable summary
	Detail     string // human-readable explanation (may be empty, see below)
	Instance   string // RFC 9457 problem "instance" URI
}
```

```go
result, err := client.SecretScanner().Query(ctx, nil)
if err != nil {
	var apiErr *postman.APIError
	if errors.As(err, &apiErr) {
		fmt.Println(apiErr.StatusCode, apiErr.Title, apiErr.Detail)
	}
	return err
}
```

## A Known Gap: Some 403s Carry No Detail

Postman's official TypeScript SDK models a handful of error responses
(notably many services' `403 Forbidden`) as a union of several possible
shapes. The OpenAPI spec this SDK generates from can't statically resolve
that union to a concrete schema, so those responses decode with only the HTTP
status code — `Title` is a generic `http.StatusText(403)` and `Detail`/`Type`/
`Instance` are empty. This is documented per-service where it applies; see
[`scripts/gen-openapi/README.md`](https://github.com/grokify/postman-go/blob/main/scripts/gen-openapi/README.md)'s
"Known approximations" for the full list of cases this affects.

Everywhere else, `Detail`/`Type`/`Instance` are populated from Postman's
actual response body.

## Common Status Codes

| Status | Meaning |
|--------|---------|
| 400 | Invalid request (bad parameters, conflicting fields) |
| 401 | Missing or invalid API key |
| 403 | Valid API key, but missing plan/add-on or permission |
| 404 | Resource not found |
| 429 | Rate limited — see [Rate Limiting](rate-limiting.md) |
| 500 | Postman-side error |

Not every status applies to every endpoint — check the relevant
[service page](../services/analytics.md) or the
[API Reference](../api-reference.html) for which errors a given operation can
return.

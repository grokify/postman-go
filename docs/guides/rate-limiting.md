# Rate Limiting

Postman's default API rate limit is **300 requests per minute** per API key,
with additional monthly quotas depending on your plan. See
[Postman's rate limit reference](https://learning.postman.com/docs/reference/postman-api/postman-api-rate-limits)
for exact numbers and how they vary by endpoint and plan.

## How Postman Signals Rate Limits

Postman returns rate-limit accounting on every response via headers such as
`RateLimit-Limit`, `RateLimit-Remaining`, and `RateLimit-Reset` (and their
`X-RateLimit-*` equivalents), plus monthly-quota variants
(`RateLimit-Limit-Month`, `RateLimit-Remaining-Month`). When you exceed a
limit, the response is `429 Too Many Requests` with a `RetryAfter` /
`X-RateLimit-RetryAfter` header giving the number of seconds to wait.

## What This SDK Exposes Today

`*postman.APIError` carries the HTTP status code, so you can detect a 429:

```go
result, err := client.SecretScanner().SecretTypes(ctx)
if err != nil {
	var apiErr *postman.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusTooManyRequests {
		// back off and retry
	}
	return err
}
```

`APIError` does **not** currently carry response headers, so the
`RateLimit-*` / `RetryAfter` values above aren't parsed out for you — this is
a known gap, not a deliberate omission. The SDK also has no built-in retry or
backoff logic; every call is a single attempt.

## A Simple Retry Wrapper

Until header access is added, a fixed or exponential backoff keyed off the
429 status code is a reasonable stopgap:

```go
func withRetry(ctx context.Context, maxAttempts int, fn func() error) error {
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err = fn()
		var apiErr *postman.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTooManyRequests {
			return err // success, or an error worth surfacing immediately
		}
		select {
		case <-time.After(time.Duration(attempt+1) * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}
```

```go
err := withRetry(ctx, 3, func() error {
	_, err := client.Workspaces().List(ctx, nil)
	return err
})
```

For latency-sensitive or high-volume use, prefer reading the actual
`Retry-After` value once header access lands, rather than a fixed schedule.

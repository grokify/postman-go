# Service Accounts

The Service Accounts service exchanges a service account API key for a
short-lived access token used to authenticate downstream service-to-service
requests. Reachable via `client.ServiceAccounts()`.

!!! note "Service account API keys only"
    The API key configured on the client must belong to a service account —
    API keys belonging to regular users aren't supported. This endpoint is
    also rate-limited to 10 requests per 10 second window per user.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
)

client, _ := postman.NewClient(postman.WithAPIKey(serviceAccountKey))
ctx := context.Background()

token, err := client.ServiceAccounts().GenerateToken(ctx)
fmt.Println(token.AccessToken)
```

## Methods

| Method | Description |
|--------|-------------|
| `GenerateToken(ctx)` | Exchange the client's service account API key for a short-lived access token. |

### GenerateToken

```go
token, err := client.ServiceAccounts().GenerateToken(ctx)
// token.AccessToken is a JWT, valid for 15 minutes.
```

## Reference

Source: [`serviceaccounts/`](https://github.com/grokify/postman-go/tree/main/serviceaccounts) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/serviceaccounts)

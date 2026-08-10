# OAuth 2.0

Generate and revoke OAuth 2.0 access tokens for client applications using the
`client_credentials` grant type. Reachable via `client.OAuth2()`. Use these
tokens with backend services or bots to authenticate and authorize API
requests without user interaction.

## Quick Example

```go
import (
	"context"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/oauth2"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// Generate an access token.
tok, err := client.OAuth2().Generate(ctx, &oauth2.GenerateInput{
	GrantType:          "client_credentials",
	InstallationAuthID: "installation-id",
	JWT:                "jwt",
})

// Revoke it when no longer needed.
_, err = client.OAuth2().Revoke(ctx, tok.AccessToken)
```

## Methods

| Method | Description |
|--------|-------------|
| `Generate(ctx, *GenerateInput)` | Generate an OAuth 2.0 access token via the `client_credentials` grant type. |
| `Revoke(ctx, token)` | Revoke an active OAuth 2.0 access token. Immediate and can't be undone. |

### Generate

```go
tok, err := client.OAuth2().Generate(ctx, &oauth2.GenerateInput{
	GrantType:          "client_credentials",
	InstallationAuthID: "installation-id",
	JWT:                "signed-jwt",
})
fmt.Println(tok.AccessToken, tok.ExpiresIn, tok.TokenType)
// TokenType is currently always oauth2.TokenTypeBearer.
```

### Revoke

```go
result, err := client.OAuth2().Revoke(ctx, "ACCESS_TOKEN")
fmt.Println(result.Success)
```

!!! note "No authorization required"
    Revoke does not use any authorization — the token itself is the
    credential for the request.

## Reference

Source: [`oauth2/`](https://github.com/grokify/postman-go/tree/main/oauth2) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/oauth2)

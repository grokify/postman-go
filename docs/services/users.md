# Users

Retrieves information about the authenticated user and users on a Postman
team. Reachable via `client.Users()`.

!!! note "Role-dependent response"
    `Me` returns a different response shape for users with the Guest and
    Partner roles, and the `Operations` usage data only returns for users on
    Free plans.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/users"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// Get the authenticated user.
me, err := client.Users().Me(ctx)
fmt.Println(me.User.Username, me.User.Email)

// List all users on the team.
all, err := client.Users().List(ctx, nil)

// Get a single team user by ID.
one, err := client.Users().Get(ctx, all.Users[0].ID)
```

## Methods

| Method | Description |
|--------|-------------|
| `Me(ctx)` | Returns information about the user authenticated by the request's API key. |
| `List(ctx, *ListInput)` | Returns information about all users on the Postman team, optionally filtered by user group. |
| `Get(ctx, userID)` | Returns information about a single user on the Postman team. |

### Me

```go
result, err := client.Users().Me(ctx)
fmt.Println(result.User.ID, result.User.Username, result.User.Email, result.User.TeamName)
for _, op := range result.Operations {
	fmt.Println(op.Name, op.Usage, op.Limit, op.Overage)
}
```

### List

```go
result, err := client.Users().List(ctx, &users.ListInput{
	GroupID: 7, // limit to members of this user group; use the Groups API to get IDs
})
for _, u := range result.Users {
	fmt.Println(u.ID, u.Name, u.Username, u.Email, u.Roles)
}
```

A `nil` input returns all users on the team.

### Get

```go
user, err := client.Users().Get(ctx, 5)
fmt.Println(user.Name, user.Username, user.Email, user.Roles, user.JoinedAt)
```

## Reference

Source: [`users/`](https://github.com/grokify/postman-go/tree/main/users) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/users)

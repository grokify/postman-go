# Groups

Postman user groups let team admins organize users for access control.
Reachable via `client.Groups()`. See
[User groups](https://learning.postman.com/docs/collaborating-in-postman/user-groups/)
for background.

## Quick Example

```go
import (
	"context"

	postman "github.com/grokify/postman-go"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List all of the team's groups.
groups, err := client.Groups().List(ctx)

// Get details for one group.
group, err := client.Groups().Get(ctx, "1")
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx)` | List all of a team's Postman groups. |
| `Get(ctx, groupID)` | Get detailed information about a single Postman group. |

### List

```go
groups, err := client.Groups().List(ctx)
for _, g := range groups {
	fmt.Println(g.ID, g.Name, g.Members, g.Roles)
}
```

### Get

```go
group, err := client.Groups().Get(ctx, "1")
fmt.Println(group.Name, group.Summary)
fmt.Println(group.Members, group.Managers, group.Roles)
```

## Reference

Source: [`groups/`](https://github.com/grokify/postman-go/tree/main/groups) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/groups)

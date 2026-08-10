# Private API Network

The Private API Network lets a team publish and discover internal API
workspaces, and manage requests from members to add their workspaces to it.
Reachable via `client.PrivateAPINetwork()`.

!!! note "Requires an add-on"
    The Private API Network API requires a Postman Enterprise plan with the
    Private API Network add-on enabled — see
    [Authentication](../guides/authentication.md#enterprise-add-ons).

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/privateapinetwork"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List workspaces published to the team's Private API Network.
list, err := client.PrivateAPINetwork().List(ctx, &privateapinetwork.ListInput{Limit: 50})
for _, e := range list.Elements {
	fmt.Println(e.ID, e.Name, e.Type)
}

// Publish a workspace to the network.
elem, err := client.PrivateAPINetwork().Add(ctx, &privateapinetwork.AddInput{
	WorkspaceID: "ws-id",
})

// Review and approve a pending add request.
reqs, err := client.PrivateAPINetwork().ListAddRequests(ctx, &privateapinetwork.ListAddRequestsInput{
	Status: privateapinetwork.RequestStatusPending,
})
for _, r := range reqs.Requests {
	_, err = client.PrivateAPINetwork().RespondAddRequest(ctx, fmt.Sprint(r.ID), &privateapinetwork.RespondInput{
		Status: privateapinetwork.DecisionApproved,
	})
}
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx, *ListInput)` | List the workspaces added to the team's Private API Network. |
| `Add(ctx, *AddInput)` | Publish a workspace to the team's Private API Network. |
| `Remove(ctx, workspaceID)` | Remove a workspace from the Private API Network (does not delete the workspace). |
| `UpdateFolder(ctx, workspaceID, *UpdateFolderInput)` | Move a workspace to a different Private API Network folder. Deprecated by the Postman API. |
| `ListAddRequests(ctx, *ListAddRequestsInput)` | List requests from members to add workspaces to the network. |
| `RespondAddRequest(ctx, requestID, *RespondInput)` | Approve or deny a member's add request. |

### List

```go
list, err := client.PrivateAPINetwork().List(ctx, &privateapinetwork.ListInput{
	Name:      "billing",
	Sort:      privateapinetwork.SortFieldCreatedAt,
	Direction: privateapinetwork.SortDesc,
	Limit:     50,
})
for _, e := range list.Elements {
	fmt.Println(e.ID, e.Name, e.Href)
}
```

`Sort` and `Direction` must both be set to have an effect.

### Add

```go
elem, err := client.PrivateAPINetwork().Add(ctx, &privateapinetwork.AddInput{
	WorkspaceID: "ws-id",
})
// elem.ID, elem.Name, elem.AddedAt
```

### Remove

```go
_, err := client.PrivateAPINetwork().Remove(ctx, "WORKSPACE_ID")
```

This does not delete the workspace; it only removes it from the Private API
Network folder.

### UpdateFolder

```go
err := client.PrivateAPINetwork().UpdateFolder(ctx, "WORKSPACE_ID", &privateapinetwork.UpdateFolderInput{
	ParentFolderID: 42,
})
```

This endpoint is deprecated in the Postman API.

### ListAddRequests

```go
reqs, err := client.PrivateAPINetwork().ListAddRequests(ctx, &privateapinetwork.ListAddRequestsInput{
	Status: privateapinetwork.RequestStatusPending,
	Limit:  50,
})
for _, r := range reqs.Requests {
	fmt.Println(r.ID, r.Element.Name, r.Status)
}
```

### RespondAddRequest

```go
_, err := client.PrivateAPINetwork().RespondAddRequest(ctx, "REQUEST_ID", &privateapinetwork.RespondInput{
	Status:  privateapinetwork.DecisionApproved,
	Message: "Looks good, welcome to the network.",
})
```

Only managers can approve or deny a request; once approved, the workspace
appears in the team's Private API Network.

## Reference

Source: [`privateapinetwork/`](https://github.com/grokify/postman-go/tree/main/privateapinetwork) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/privateapinetwork)

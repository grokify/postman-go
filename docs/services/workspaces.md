# Workspaces

Workspaces group related collections, environments, mocks, monitors, and
specs, and control who can access them. Reachable via `client.Workspaces()`.
See [Postman's guide to workspaces](https://learning.postman.com/docs/collaborating-in-postman/using-workspaces/creating-workspaces/)
for background on visibility and team plans.

!!! note "Known API limitations"
    - `ManagePartnerInvites` sends a free-form JSON body: Postman's schema
      for this endpoint is a union of three request shapes (invite, remove
      from workspace, remove from team) that the generator can't resolve
      statically, so both the request and response are raw JSON.
    - `UpdateRoles` cannot address principals by SCIM ID — Postman's
      `identifierType` header was dropped from the published OpenAPI spec,
      so the generated client has no way to send it.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/workspaces"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List workspaces the caller can access.
list, err := client.Workspaces().List(ctx, nil)
for _, ws := range list.Workspaces {
	fmt.Println(ws.ID, ws.Name, ws.Type)
}

// Fetch full detail, including the collections/environments/specs it contains.
ws, err := client.Workspaces().Get(ctx, list.Workspaces[0].ID, nil)

// Create a new team workspace.
created, err := client.Workspaces().Create(ctx, &workspaces.CreateInput{
	Name: "Platform Team",
	Type: workspaces.WorkspaceTypeTeam,
})
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx, *ListInput)` | List the workspaces you have access to, filtered by type, creator, or element. Paginated. |
| `Create(ctx, *CreateInput)` | Create a new workspace. |
| `Get(ctx, workspaceID, *GetInput)` | Return full detail for a workspace, including its collections, environments, mocks, monitors, and specs. |
| `Update(ctx, workspaceID, *UpdateInput)` | Update a workspace's name, type/visibility, description, or about text. |
| `Delete(ctx, workspaceID)` | Delete a workspace. |
| `AllRoles(ctx)` | List all roles assignable in a workspace, based on the team's plan. |
| `Roles(ctx, workspaceID, *RolesInput)` | List the users, user groups, and partners with roles in a workspace. |
| `UpdateRoles(ctx, workspaceID, *UpdateRolesInput)` | Add or remove roles for users, user groups, or partners in a workspace. |
| `TransferElement(ctx, workspaceID, *TransferElementInput)` | Transfer a collection, environment, mock, monitor, or Flow to another workspace. |
| `TransferToTeam(ctx, workspaceID, *TransferToTeamInput)` | Transfer an entire workspace from one team to another. |
| `GlobalVariables(ctx, workspaceID)` | Return a workspace's global variables. |
| `UpdateGlobalVariables(ctx, workspaceID, *UpdateGlobalVariablesInput)` | Replace all of a workspace's global variables. |
| `ActivityFeed(ctx, workspaceID, *ActivityFeedInput)` | List who added/removed elements or joined/left a workspace. Paginated. |
| `ManagePartnerInvites(ctx, *ManagePartnerInvitesInput)` | Send, remove, or revoke Partner Workspace invitations (raw JSON; see note above). |
| `Updates(ctx, workspaceID, *UpdatesInput)` | List the workspace updates (announcements) posted in a workspace. Paginated. |
| `CreateUpdate(ctx, workspaceID, *CreateUpdateInput)` | Post a new workspace update. |
| `GetUpdate(ctx, workspaceID, updateID)` | Return a single workspace update. |
| `PatchUpdate(ctx, workspaceID, updateID, *PatchUpdateInput)` | Update a workspace update. |
| `DeleteUpdate(ctx, workspaceID, updateID)` | Delete a workspace update. |

### Create, Get, Update, Delete

```go
created, err := client.Workspaces().Create(ctx, &workspaces.CreateInput{
	Name:        "Platform Team",
	Type:        workspaces.WorkspaceTypeTeam,
	Description: "APIs owned by the platform team",
})

ws, err := client.Workspaces().Get(ctx, created.ID, &workspaces.GetInput{
	Include: workspaces.IncludeMocksDeactivated,
})
for _, c := range ws.Collections {
	fmt.Println(c.ID, c.Name)
}

updated, err := client.Workspaces().Update(ctx, created.ID, &workspaces.UpdateInput{
	Description: "Updated description",
})

deleted, err := client.Workspaces().Delete(ctx, created.ID)
```

`WorkspaceType` doubles as visibility (`WorkspaceTypePersonal`, `...Team`,
`...Private`, `...Public`, `...Partner`). `Update` does not support changing
visibility from private to public, public to private, (on Free/Solo plans)
private to personal, or public to personal for team users.

### Roles: AllRoles, Roles, UpdateRoles

```go
all, err := client.Workspaces().AllRoles(ctx)

roles, err := client.Workspaces().Roles(ctx, "workspace-id", &workspaces.RolesInput{IncludeSCIM: false})
for _, r := range roles.Roles {
	fmt.Println(r.DisplayName, r.User)
}

result, err := client.Workspaces().UpdateRoles(ctx, "workspace-id", &workspaces.UpdateRolesInput{
	Operations: []workspaces.RoleOperation{
		{
			Op:   "add",
			Path: workspaces.RolesPathUser,
			Value: []workspaces.RoleChange{
				{ID: "user-id", Role: "viewer-role-id"},
			},
		},
	},
})
```

`UpdateRoles` is restricted to 50 operations per call, each unique per
principal; user groups require an Enterprise plan; it doesn't support the
external Guest role or updating partner and user roles in the same call.

### Transfers: TransferElement, TransferToTeam

```go
moved, err := client.Workspaces().TransferElement(ctx, "workspace-id", &workspaces.TransferElementInput{
	ElementID:   "userId-collection-id",
	ElementType: workspaces.TransferElementTypeCollection,
	To:          "destination-workspace-id",
})

teamMove, err := client.Workspaces().TransferToTeam(ctx, "workspace-id", &workspaces.TransferToTeamInput{
	Source:      "source-team-id",
	Destination: "destination-team-id",
})
```

`TransferElement` supports collections, environments, mocks, monitors, and
Flows modules/actions; transfers from team workspaces to personal workspaces
are not supported. `TransferToTeam` requires Postman Enterprise with Postman
Organizations enabled, and can strip a user's workspace role if they lack a
role in the destination team.

### GlobalVariables, UpdateGlobalVariables

```go
vars, err := client.Workspaces().GlobalVariables(ctx, "workspace-id")

updated, err := client.Workspaces().UpdateGlobalVariables(ctx, "workspace-id", &workspaces.UpdateGlobalVariablesInput{
	Values: []workspaces.GlobalVariable{
		{Key: "baseUrl", Type: workspaces.GlobalVariableTypeDefault, Value: "https://api.example.com", Enabled: true},
		{Key: "apiKey", Type: workspaces.GlobalVariableTypeSecret, Value: "s3cr3t", Enabled: true},
	},
})
```

`UpdateGlobalVariables` replaces the workspace's entire set of global
variables.

### ActivityFeed

```go
feed, err := client.Workspaces().ActivityFeed(ctx, "workspace-id", &workspaces.ActivityFeedInput{Limit: 50})
for _, e := range feed.Entries {
	fmt.Println(e.Action, e.ElementType, e.ElementName, e.User.Username)
}
```

### ManagePartnerInvites

```go
body := []byte(`{"action":"invite_partner","targetEntity":"workspace","targetEntityId":"workspace-id","roleId":"role-id","target":{"emails":["a@example.com"]}}`)
result, err := client.Workspaces().ManagePartnerInvites(ctx, &workspaces.ManagePartnerInvitesInput{Body: body})
```

Requires a Postman Team or Enterprise plan. See the note above for the three
supported body shapes.

### Workspace updates: Updates, CreateUpdate, GetUpdate, PatchUpdate, DeleteUpdate

```go
posted, err := client.Workspaces().CreateUpdate(ctx, "workspace-id", &workspaces.CreateUpdateInput{
	Topic:       "New collection published",
	Description: "The v2 API collection is now available.",
	Category:    workspaces.UpdateCategoryNewFeature,
})

list, err := client.Workspaces().Updates(ctx, "workspace-id", &workspaces.UpdatesInput{
	Category: "new_feature,announcement",
})

one, err := client.Workspaces().GetUpdate(ctx, "workspace-id", posted.ID)

_, err = client.Workspaces().PatchUpdate(ctx, "workspace-id", posted.ID, &workspaces.PatchUpdateInput{
	IsPinned: true,
	Category: workspaces.UpdateCategoryAnnouncement,
})

err = client.Workspaces().DeleteUpdate(ctx, "workspace-id", posted.ID)
```

`UpdateCategory` values are `...Improvement`, `...NewFeature`, `...BugFix`,
`...BreakingChange`, and `...Announcement`.

## Reference

Source: [`workspaces/`](https://github.com/grokify/postman-go/tree/main/workspaces) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/workspaces)

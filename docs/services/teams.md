# Teams

The Teams API manages a Postman team: list and create teams, inspect a
team's membership and settings, and manage access requests and member
roles. Reachable via `client.Teams()`.

!!! note "Admin permissions required"
    Most Teams endpoints require Team Admin (or higher) permissions on the
    target team.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/teams"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List all teams the caller can see.
list, err := client.Teams().List(ctx, nil)
for _, t := range list.Teams {
	fmt.Println(t.ID, t.Name, t.MemberCount)
}

// Get a single team, including its members.
team, err := client.Teams().Get(ctx, "12345", &teams.GetInput{Include: teams.IncludeMembers})

// Approve a pending access request.
_, err = client.Teams().DecideAccessRequest(ctx, "12345", "67890", teams.AccessActionApprove)
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx, *ListInput)` | List all Postman teams in your organization. Paginated. |
| `Create(ctx, *CreateInput)` | Create a Postman team in your organization. |
| `Get(ctx, teamID, *GetInput)` | Get information about a Postman team. |
| `AccessRequests(ctx, teamID, *AccessRequestsInput)` | Get a team's pending access requests. Paginated. |
| `RequestAccess(ctx, teamID, *RequestAccessInput)` | Create an access request for a team (join, upgrade role, or add members). |
| `DecideAccessRequest(ctx, teamID, requestID, action)` | Approve or deny a team's pending access request. |
| `ManageMemberRoles(ctx, teamID, *ManageMemberRolesInput)` | Add or remove member roles in groups, teams, organizations, and for individual users. |
| `RemoveMembers(ctx, teamID, entities)` | Remove entities (users, groups, teams, or organizations) from a Postman team. |
| `Settings(ctx, teamID)` | Get a team's settings. |
| `UpdateSettings(ctx, teamID, *UpdateSettingsInput)` | Update a team's settings. |

### List

```go
result, err := client.Teams().List(ctx, &teams.ListInput{
	Limit:    50,
	Settings: true,
})
// result.Teams []Team, result.NextCursor
```

A `nil` or zero input lists all teams the caller can see.

### Create

```go
team, err := client.Teams().Create(ctx, &teams.CreateInput{
	Name:        "Platform Team",
	Description: "Owns the core platform APIs",
})
```

### Get

```go
team, err := client.Teams().Get(ctx, "TEAM_ID", &teams.GetInput{
	Include: teams.IncludeUserRoles,
})
```

### AccessRequests

```go
result, err := client.Teams().AccessRequests(ctx, "TEAM_ID", &teams.AccessRequestsInput{Limit: 25})
for _, r := range result.Requests {
	fmt.Println(r.ID, r.RequestType, r.Role, r.Status)
}
```

### RequestAccess

```go
result, err := client.Teams().RequestAccess(ctx, "TEAM_ID", &teams.RequestAccessInput{
	Entities: []teams.EntityRef{
		{Type: teams.EntityTypeUser, ID: "98765"},
	},
	Role:        teams.RoleTeamDeveloper,
	Reason:      "Joining the platform team",
	RequestType: teams.RequestTypeJoin,
})
```

If team discovery is enabled, the request is automatically approved.

### DecideAccessRequest

```go
result, err := client.Teams().DecideAccessRequest(ctx, "TEAM_ID", "REQUEST_ID", teams.AccessActionApprove)
fmt.Println(result.Status)
```

### ManageMemberRoles

```go
result, err := client.Teams().ManageMemberRoles(ctx, "TEAM_ID", &teams.ManageMemberRolesInput{
	Add: &teams.MemberRoleChanges{
		Users: []teams.Role{teams.RoleTeamDeveloper},
	},
})
```

!!! note "One implicit target per category"
    The reconstructed OpenAPI spec represents the API's dynamic
    per-entity-ID JSON object (e.g. `{"<userId>": ["TEAM_MANAGER"]}`) with a
    single fixed placeholder key, because the source TypeScript SDK encodes
    that key as a literal template string rather than a real
    `additionalProperties` map. As a result `MemberRoleChanges` can only
    address one implicit target per category (`Users`, `Groups`, `Orgs`,
    `Teams`) — the role slices apply to that category as a whole, not to
    individual entity IDs. Removing a role from a group or team removes that
    role's permissions from all its members.

### RemoveMembers

```go
err := client.Teams().RemoveMembers(ctx, "TEAM_ID", []teams.EntityRef{
	{Type: teams.EntityTypeUser, ID: "98765"},
})
```

### Settings

```go
settings, err := client.Teams().Settings(ctx, "TEAM_ID")
fmt.Println(settings.RfaForAddMember)
```

### UpdateSettings

```go
settings, err := client.Teams().UpdateSettings(ctx, "TEAM_ID", &teams.UpdateSettingsInput{
	RfaForAddMember: teams.RfaEnabled,
})
```

Leave a field empty to leave the corresponding setting unchanged.

## Reference

Source: [`teams/`](https://github.com/grokify/postman-go/tree/main/teams) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/teams)

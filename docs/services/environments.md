# Environments

Environments store variables (plain or secret-backed) that can be applied to
requests within a workspace. This package also covers environment forking:
creating a fork, listing an environment's forks, merging a fork back into its
parent, and pulling changes from a parent into a fork. Reachable via
`client.Environments()`.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/environments"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List environments in a workspace.
list, err := client.Environments().List(ctx, &environments.ListInput{Workspace: "ws-id"})
for _, e := range list.Environments {
	fmt.Println(e.ID, e.Name)
}

// Create an environment with a plain and a secret variable.
created, err := client.Environments().Create(ctx, &environments.CreateInput{
	Workspace: "ws-id",
	Name:      "Staging",
	Values: []environments.Variable{
		{Key: "base_url", Value: "https://staging.example.com", Type: environments.VariableTypeDefault, Enabled: true},
	},
})

// Fetch it back, including its variables.
got, err := client.Environments().Get(ctx, created.ID)
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx, *ListInput)` | List information about all of the caller's environments, optionally filtered by workspace. |
| `Create(ctx, *CreateInput)` | Create an environment with the given name and variables. |
| `Get(ctx, environmentID)` | Get an environment, including its variables. |
| `Replace(ctx, environmentID, *ReplaceInput)` | Replace all of an environment's contents (name and variables). |
| `Patch(ctx, environmentID, *PatchInput)` | Apply JSON Patch-style operations to update specific properties. |
| `Delete(ctx, environmentID)` | Delete an environment. |
| `Forks(ctx, environmentID, *ForksInput)` | List an environment's forked environments. Paginated. |
| `Fork(ctx, environmentID, *ForkInput)` | Create a fork of an environment. |
| `Merge(ctx, environmentID, *MergeInput)` | Merge a forked environment back into its parent. |
| `Pull(ctx, environmentUID, *PullInput)` | Pull changes from a parent environment into a fork. |

### List

```go
list, err := client.Environments().List(ctx, &environments.ListInput{Workspace: "ws-id"})
for _, e := range list.Environments {
	fmt.Println(e.ID, e.Name, e.IsPublic)
}
```

A `nil` input (or an empty `Workspace`) returns all environments visible to
the caller.

### Create

```go
created, err := client.Environments().Create(ctx, &environments.CreateInput{
	Workspace: "ws-id",
	Name:      "Staging",
	Values: []environments.Variable{
		{Key: "base_url", Value: "https://staging.example.com", Type: environments.VariableTypeDefault, Enabled: true},
		{Key: "api_key", Secret: true, Type: environments.VariableTypeSecret, Enabled: true},
	},
})
// created.ID, created.Name, created.UID
```

If `Workspace` is empty, the environment is created in the oldest personal
Internal workspace the caller owns. The request body, including all variable
values, cannot exceed 30MB.

### Get

```go
got, err := client.Environments().Get(ctx, "ENVIRONMENT_ID")
for _, v := range got.Values {
	fmt.Println(v.Key, v.Value, v.Type)
}
```

### Replace

```go
_, err := client.Environments().Replace(ctx, "ENVIRONMENT_ID", &environments.ReplaceInput{
	Name: "Staging v2",
	Values: []environments.Variable{
		{Key: "base_url", Value: "https://staging2.example.com", Enabled: true},
	},
})
```

`Replace` overwrites the entire environment; use `Patch` to update specific
fields instead.

### Patch

```go
_, err := client.Environments().Patch(ctx, "ENVIRONMENT_ID", &environments.PatchInput{
	Ops: []environments.PatchOp{
		{Op: "replace", Path: "/name", Value: "Renamed Environment"},
	},
})
```

Only one type of operation may be performed per call (for example, you
cannot combine an `"add"` and a `"replace"` in the same request).

### Delete

```go
_, err := client.Environments().Delete(ctx, "ENVIRONMENT_ID")
```

### Forks

```go
forks, err := client.Environments().Forks(ctx, "ENVIRONMENT_ID", &environments.ForksInput{
	Limit:     50,
	Direction: environments.DirectionDesc,
	Sort:      environments.SortCreatedAt,
})
for _, f := range forks.Forks {
	fmt.Println(f.ForkID, f.ForkName)
}
```

### Fork

```go
fork, err := client.Environments().Fork(ctx, "ENVIRONMENT_ID", &environments.ForkInput{
	WorkspaceID: "ws-id",
	ForkName:    "my-fork",
})
// fork.UID, fork.ForkName
```

### Merge

```go
_, err := client.Environments().Merge(ctx, "PARENT_ENVIRONMENT_ID", &environments.MergeInput{
	Source:       "FORK_ENVIRONMENT_ID",
	DeleteSource: true,
})
```

### Pull

```go
_, err := client.Environments().Pull(ctx, "FORK_ENVIRONMENT_UID", &environments.PullInput{
	Source: "PARENT_ENVIRONMENT_ID",
})
```

Pulls changes from a parent (source) environment into the forked
(destination) environment identified by `environmentUID`.

## Reference

Source: [`environments/`](https://github.com/grokify/postman-go/tree/main/environments) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/environments)

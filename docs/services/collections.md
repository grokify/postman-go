# Collections

Collections group related API requests together, along with example
responses, scripts, and variables. This package covers collection CRUD,
forking and merging, comments, pull requests, roles, publishing
documentation, and transferring items between collections. Reachable via
`client.Collections()`.

!!! note "Collection bodies are raw JSON"
    Full collection bodies (for `Create`, `Replace`, `Update`, and `Get`) are
    passed and returned as raw JSON in the
    [Postman Collection v2.1.0 format](https://schema.postman.com/collection/json/v2.1.0/draft-07/docs/index.html)
    rather than as hand-modeled Go structs — the item tree (folders,
    requests, responses, auth, scripts) is a large, separately-versioned
    schema that is orthogonal to this service's job of managing collections.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/collections"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// List collections in a workspace.
list, err := client.Collections().List(ctx, &collections.ListInput{Workspace: "workspace-id"})
for _, c := range list.Collections {
	fmt.Println(c.ID, c.Name)
}

// Fetch a collection's full v2.1.0 JSON body.
got, err := client.Collections().Get(ctx, list.Collections[0].ID, nil)
fmt.Println(string(got.Collection))

// Fork it into another workspace.
fork, err := client.Collections().Fork(ctx, list.Collections[0].ID, &collections.ForkInput{
	Workspace: "other-workspace-id",
	Label:     "my fork",
})
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx, *ListInput)` | List the authenticated user's subscribed collections, filtered by workspace or name. Paginated. |
| `Create(ctx, workspaceID, collection []byte)` | Create a collection in a workspace from a Postman Collection v2.1.0 JSON document. |
| `Get(ctx, collectionID, *GetInput)` | Return a collection's contents as raw v2.1.0 JSON. |
| `Replace(ctx, collectionID, collection []byte)` | Replace a collection's entire contents from a v2.1.0 JSON document. |
| `Update(ctx, collectionID, collection []byte)` | Update specific collection information from a partial v2.1.0 JSON document. |
| `Delete(ctx, collectionID)` | Delete a collection. |
| `UpdateStatus(ctx, taskID)` | Get the status of an asynchronous collection update task (see note below). |
| `Fork(ctx, collectionID, *ForkInput)` | Create a fork of an existing collection into a workspace. |
| `ForkedByUser(ctx, *ForkedByUserInput)` | List all of the authenticated user's forked collections. Paginated. |
| `Forks(ctx, collectionID, *ForksInput)` | List a collection's forked collections. Paginated. |
| `MergeFork(ctx, *MergeForkInput)` | Merge a forked collection back into its parent. **Deprecated** — use `MergeForkAsync`. |
| `MergeForkAsync(ctx, *MergeForkAsyncInput)` | Merge or pull a forked collection and its parent asynchronously. |
| `MergeForkAsyncStatus(ctx, taskID)` | Get the status of a task started by `MergeForkAsync`. |
| `Pull(ctx, collectionID)` | Pull changes from a fork's parent collection into the fork. |
| `SourceStatus(ctx, collectionID)` | Check whether a fork's parent collection has changes not yet pulled. |
| `Duplicate(ctx, collectionID, *DuplicateInput)` | Duplicate a collection into another workspace. |
| `DuplicateStatus(ctx, taskID)` | Get the status of a task started by `Duplicate`. |
| `Comments(ctx, collectionID)` | List all comments on a collection. |
| `CreateComment(ctx, collectionID, body, threadID)` | Add a comment, or a reply to an existing thread. |
| `UpdateComment(ctx, collectionID, commentID, body)` | Update a comment. |
| `DeleteComment(ctx, collectionID, commentID)` | Delete a comment (deleting a thread's first comment deletes the thread). |
| `PullRequests(ctx, collectionID)` | List a collection's pull requests. |
| `CreatePullRequest(ctx, collectionID, *CreatePullRequestInput)` | Create a pull request from a forked collection into its parent. |
| `Roles(ctx, collectionID)` | List the users, teams, and groups with access to a collection. |
| `UpdateRoles(ctx, collectionID, []RoleUpdate)` | Update the roles of users, groups, or teams on a collection. |
| `PublishDocs(ctx, collectionID, *PublishDocsInput)` | Publish a collection's documentation publicly. |
| `UnpublishDocs(ctx, collectionID)` | Unpublish a collection's documentation. |
| `TransformToOpenAPI(ctx, collectionID, format)` | Transform a collection into a stringified OpenAPI definition. |
| `TransferFolders(ctx, *TransferInput)` | Copy or move folders into a collection or folder. |
| `TransferRequests(ctx, *TransferInput)` | Copy or move requests into a collection or folder. |
| `TransferResponses(ctx, *TransferInput)` | Copy or move responses into a request. |

### Create, Get, Replace, Update, Delete

`Create`, `Replace`, and `Update` accept a Postman Collection v2.1.0 JSON
document as a `[]byte`; `Get` returns one the same way.

```go
// Create in a specific workspace (empty workspaceID creates it in the
// caller's oldest personal Internal workspace).
created, err := client.Collections().Create(ctx, "workspace-id", collectionJSON)

// Get, restricted to root-level IDs instead of the full item tree.
got, err := client.Collections().Get(ctx, created.ID, &collections.GetInput{
	Model: collections.CollectionModelMinimal,
})

// Update just a few fields via a partial v2.1.0 document.
updated, err := client.Collections().Update(ctx, created.ID, []byte(`{"info":{"name":"New Name"}}`))

// Delete when no longer needed.
deleted, err := client.Collections().Delete(ctx, created.ID)
```

`Replace` overwrites a collection's entire contents — omitting the existing
`id`/`uid` values in the document causes items to be recreated with new IDs.
The maximum accepted document size is 100 MB.

`UpdateStatus(ctx, taskID)` tracks an async `PUT` performed with the
`Prefer: respond-async` header via `GetCollectionUpdatesTasksParams`. This
SDK's `Replace` always runs synchronously (the generated API client does not
model the 202 response or the `Prefer` header), so `UpdateStatus` only helps
if you obtained a task ID some other way (direct API call or another SDK).

### Fork, ForkedByUser, Forks, MergeFork, MergeForkAsync, Pull, SourceStatus

```go
fork, err := client.Collections().Fork(ctx, "collection-id", &collections.ForkInput{
	Workspace: "workspace-id",
	Label:     "my fork",
})
// fork.ID, fork.UID, fork.Fork.From

forks, err := client.Collections().Forks(ctx, "collection-id", &collections.ForksInput{Limit: 25})

mine, err := client.Collections().ForkedByUser(ctx, &collections.ForkedByUserInput{Limit: 25})

// Merge a fork back into its parent (async, recommended).
task, err := client.Collections().MergeForkAsync(ctx, &collections.MergeForkAsyncInput{
	Source:      fork.ID, // parent's ID when pulling into the fork
	Destination: "collection-id",
	Strategy:    collections.MergeStrategyDeleteSource,
})
status, err := client.Collections().MergeForkAsyncStatus(ctx, task.ID)

// Pull the parent's changes into the fork.
pulled, err := client.Collections().Pull(ctx, fork.ID)

// Check whether the parent is ahead before pulling.
src, err := client.Collections().SourceStatus(ctx, fork.ID)
if src.IsSourceAhead {
	// ...
}
```

`MergeFork` (synchronous) is deprecated in favor of `MergeForkAsync`.
`MergeStrategy` accepts `MergeStrategyDefault`,
`MergeStrategyUpdateSourceWithDestination`, or `MergeStrategyDeleteSource`;
`MergeForkAsync` defaults to `MergeStrategyDefault` when empty, while the
deprecated `MergeFork` only accepts the latter two values.

### Duplicate, DuplicateStatus

```go
task, err := client.Collections().Duplicate(ctx, "collection-id", &collections.DuplicateInput{
	Workspace: "workspace-id",
	Suffix:    " (copy)",
})
status, err := client.Collections().DuplicateStatus(ctx, task.ID)
// status.Status: DuplicateTaskProcessing, DuplicateTaskCompleted, or DuplicateTaskFailed
```

### Comments: Comments, CreateComment, UpdateComment, DeleteComment

```go
all, err := client.Collections().Comments(ctx, "collection-id")

created, err := client.Collections().CreateComment(ctx, "collection-id", "Looks good!", 0)
reply, err := client.Collections().CreateComment(ctx, "collection-id", "Thanks!", created.ThreadID)

_, err = client.Collections().UpdateComment(ctx, "collection-id", created.ID, "Looks good, LGTM")
err = client.Collections().DeleteComment(ctx, "collection-id", created.ID)
```

Comment bodies accept a maximum of 10,000 characters. Deleting a thread's
first comment deletes the entire thread.

### Pull Requests: PullRequests, CreatePullRequest

```go
prs, err := client.Collections().PullRequests(ctx, "collection-id")

pr, err := client.Collections().CreatePullRequest(ctx, fork.ID, &collections.CreatePullRequestInput{
	Title:         "Add new endpoints",
	Description:   "Adds the v2 endpoints",
	Reviewers:     []string{"user-id"},
	DestinationID: "parent-collection-id",
})
// pr.Status: PullRequestStatusOpen, ...Approved, ...Declined, or ...Merged
```

### Roles: Roles, UpdateRoles

```go
roles, err := client.Collections().Roles(ctx, "collection-id")
for _, u := range roles.Users {
	fmt.Println(u.ID, u.Role) // collections.RoleViewer or collections.RoleEditor
}

err = client.Collections().UpdateRoles(ctx, "collection-id", []collections.RoleUpdate{
	{
		Path: collections.RolePathUser,
		Values: []collections.RoleAssignment{
			{ID: 12345678, Role: collections.RoleEditor},
		},
	},
})
```

Only collection Editors can call `UpdateRoles`; it does not support the
external Partner or Guest roles.

### Documentation: PublishDocs, UnpublishDocs

```go
docs, err := client.Collections().PublishDocs(ctx, "collection-id", &collections.PublishDocsInput{
	Layout: collections.DocLayoutClassicDoubleColumn,
	CustomColor: collections.DocColorSettings{Highlight: "#FF6C37"},
})
fmt.Println(docs.PublicURL)

err = client.Collections().UnpublishDocs(ctx, "collection-id")
```

Publishing is only supported for collections with HTTP requests, and the
collection must not be attached to an API.

### TransformToOpenAPI

```go
openapi, err := client.Collections().TransformToOpenAPI(ctx, "collection-id", collections.TransformFormatYAML)
```

Transforms a collection into a stringified OpenAPI definition; it does not
create an API resource. `format` defaults to `TransformFormatJSON` when
empty.

### Transfers: TransferFolders, TransferRequests, TransferResponses

```go
result, err := client.Collections().TransferRequests(ctx, &collections.TransferInput{
	IDs:  []string{"request-id-1", "request-id-2"},
	Mode: collections.TransferModeMove,
	Target: collections.TransferTarget{
		ID:    "destination-folder-id",
		Model: collections.TransferTargetModelFolder,
	},
	Location: collections.TransferLocation{
		Position: collections.TransferPositionEnd,
	},
})
```

`TransferFolders` and `TransferRequests` copy or move items into a
collection or folder; `TransferResponses` copies or moves responses into a
request. Leave `Location.ID`/`Location.Model` empty to place items at the
target's root.

## Reference

Source: [`collections/`](https://github.com/grokify/postman-go/tree/main/collections) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/collections)

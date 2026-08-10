# SDK Generation

This package is a client for Postman's own **SDK Generation** feature: it
generates client SDKs (in various languages) from a Postman Collection or
API specification, optionally keeping the generated SDK's Git repository
in sync via a Git connection. Reachable via `client.SDKGen()`.

!!! note "Not to be confused with this Go module"
    "SDK Generation" here refers to Postman's own product feature for
    generating SDKs from your Postman collections/specs — it is unrelated
    to `postman-go` itself, which is a hand-written Go SDK for the Postman
    API.

## Quick Example

```go
import (
	"context"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/sdkgen"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// Start generating a Go SDK from a collection.
err := client.SDKGen().Generate(ctx, &sdkgen.GenerateInput{
	Source:   sdkgen.Source{Type: sdkgen.SourceTypeCollection, ID: "collection-id"},
	Language: sdkgen.LanguageGo,
	GoOptions: &sdkgen.GoOptions{ModuleName: "github.com/example/widgets-sdk"},
})

// List generated SDKs in a workspace.
list, err := client.SDKGen().List(ctx, &sdkgen.ListInput{WorkspaceID: "workspace-id"})
for _, sdk := range list.SDKs {
	fmt.Println(sdk.ID, sdk.Language, sdk.BuildStatus)
}

// Once BuildStatus is BuildStatusSucceeded, download the archive.
dl, err := client.SDKGen().Download(ctx, list.SDKs[0].ID)
fmt.Println(dl.URL, dl.ExpiresAt)
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx, *ListInput)` | List all SDKs the authenticated user has access to. Paginated. |
| `Generate(ctx, *GenerateInput)` | Start an asynchronous generation job for one SDK from a collection or specification. |
| `Status(ctx, sdkID)` | Get information about the SDK, including its current build job status. |
| `Delete(ctx, sdkID)` | Delete an SDK record and its stored archive. |
| `Download(ctx, sdkID)` | Get a short-lived signed URL for the generated SDK archive. |
| `GitConnections(ctx, *GitConnectionsInput)` | Get all Git repository connections the caller has access to in a workspace. Paginated. |
| `ConnectGit(ctx, *ConnectGitInput)` | Connect a source element (collection or specification) to a Git repository for one SDK language. |
| `GitConnection(ctx, connectionID)` | Get information about an SDK's Git connection. |
| `UpdateGitConnection(ctx, connectionID, *UpdateGitConnectionInput)` | Update an SDK Git connection's lifecycle status. |
| `GitConnectionPullRequests(ctx, connectionID, *PullRequestsInput)` | List all SDK-update pull requests for a Git connection. Paginated. |

### List

```go
result, err := client.SDKGen().List(ctx, &sdkgen.ListInput{
	WorkspaceID: "WORKSPACE_ID", // required
	BuildStatus: sdkgen.BuildStatusSucceeded,
	Language:    sdkgen.LanguageGo,
	Limit:       25,
})
// result.SDKs []SDK, result.NextCursor, result.Total
```

### Generate

```go
err := client.SDKGen().Generate(ctx, &sdkgen.GenerateInput{
	Source:   sdkgen.Source{Type: sdkgen.SourceTypeSpec, ID: "SPEC_ID"},
	Language: sdkgen.LanguagePython,
	Version:  "1.0.0",
	Authors:  []sdkgen.Author{{Name: "Platform Team", Email: "platform@example.com"}},
	PythonOptions: &sdkgen.PythonOptions{PypiPackageName: "widgets-sdk"},
	Retry: &sdkgen.RetryOptions{
		Enabled:     true,
		MaxAttempts: 3,
	},
})
```

Only the language-specific options struct matching `Language` is used; the
rest are ignored. Track progress with `Status`; when `BuildStatus` is
`BuildStatusSucceeded`, the SDK is ready to download with `Download`.

### Status

```go
sdk, err := client.SDKGen().Status(ctx, "SDK_ID")
if sdk.BuildStatus == sdkgen.BuildStatusFailed {
	fmt.Println(sdk.Error.Code, sdk.Error.Message)
}
```

### Delete

```go
err := client.SDKGen().Delete(ctx, "SDK_ID")
```

This cannot cancel a generation job that is still in progress.

### Download

```go
dl, err := client.SDKGen().Download(ctx, "SDK_ID")
fmt.Println(dl.URL, dl.ExpiresAt)
```

The URL is created on demand and expires within a few minutes.

### GitConnections

```go
result, err := client.SDKGen().GitConnections(ctx, &sdkgen.GitConnectionsInput{
	WorkspaceID: "WORKSPACE_ID", // required
	Language:    sdkgen.LanguageGo,
	Status:      sdkgen.GitConnectionStatusActive,
})
// result.Connections []GitConnection, result.NextCursor, result.Total
```

### ConnectGit

```go
conn, err := client.SDKGen().ConnectGit(ctx, &sdkgen.ConnectGitInput{
	Source:        sdkgen.Source{Type: sdkgen.SourceTypeCollection, ID: "COLLECTION_ID"},
	Language:      sdkgen.LanguageGo,
	RepositoryURL: "https://github.com/example/widgets-sdk",
	TargetBranch:  "main",
})
```

Each source/language pair maps to a single connection; if one already
exists, this returns a 409 Conflict error. `AutoUpdatePullRequestsEnabled`
is an Enterprise-only feature — always `false` on Team plans.

### GitConnection

```go
conn, err := client.SDKGen().GitConnection(ctx, "CONNECTION_ID")
fmt.Println(conn.Status, conn.RepositoryURL)
```

Includes the SDK currently sent to the target branch and the most recent
SDK-update pull request.

### UpdateGitConnection

```go
conn, err := client.SDKGen().UpdateGitConnection(ctx, "CONNECTION_ID", &sdkgen.UpdateGitConnectionInput{
	Status: sdkgen.GitConnectionStatusDisconnected,
})
```

Only `GitConnectionStatusActive` and `GitConnectionStatusDisconnected` can
be set — `GitConnectionStatusInaccessible` is system-determined. This
action is idempotent: setting fields to their current values is a no-op.

### GitConnectionPullRequests

```go
result, err := client.SDKGen().GitConnectionPullRequests(ctx, "CONNECTION_ID", &sdkgen.PullRequestsInput{
	Status: sdkgen.PRStatusOpen,
	Limit:  25,
})
for _, pr := range result.PullRequests {
	fmt.Println(pr.Number, pr.URL, pr.Status)
}
```

Results are ordered newest first by `updatedAt`.

## Reference

Source: [`sdkgen/`](https://github.com/grokify/postman-go/tree/main/sdkgen) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/sdkgen)

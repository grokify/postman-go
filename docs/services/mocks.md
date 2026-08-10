# Mocks

A mock server simulates the behavior of an existing collection so that
clients can develop and test against realistic responses before the real
API exists (or is available). This package covers managing mock servers
themselves, their call logs, and their server-level (5xx) responses.
Reachable via `client.Mocks()`.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/mocks"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// Create a mock server for a collection.
mock, err := client.Mocks().Create(ctx, &mocks.CreateInput{
	Workspace:  "workspace-id",
	Collection: "collection-id",
	Name:       "My Mock",
})

// List all mock servers visible to the caller.
all, err := client.Mocks().List(ctx, &mocks.ListInput{Workspace: "workspace-id"})
for _, m := range all {
	fmt.Println(m.ID, m.MockURL)
}

// Inspect recent call logs.
logs, err := client.Mocks().CallLogs(ctx, mock.ID, &mocks.CallLogsInput{Limit: 20})
for _, l := range logs.CallLogs {
	fmt.Println(l.ServedAt, l.Request.Method, l.Request.Path)
}
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx, *ListInput)` | List all active mock servers visible to the caller. |
| `Create(ctx, *CreateInput)` | Create a mock server for a collection. |
| `Get(ctx, mockID)` | Get information about a mock server. |
| `Update(ctx, mockID, *UpdateInput)` | Update a mock server's properties, such as its name or collection. |
| `Delete(ctx, mockID)` | Delete a mock server. |
| `CallLogs(ctx, mockID, *CallLogsInput)` | Get a mock server's call logs. Paginated. |
| `Publish(ctx, mockID)` | Publish a mock server, setting its access control to public. |
| `Unpublish(ctx, mockID)` | Unpublish a mock server, setting its access control to private. |
| `ServerResponses(ctx, mockID)` | List all of a mock server's server-level (5xx) responses. |
| `CreateServerResponse(ctx, mockID, *CreateServerResponseInput)` | Create a server-level (5xx) response for a mock server. |
| `ServerResponse(ctx, mockID, serverResponseID)` | Get information about a mock server's server response. |
| `UpdateServerResponse(ctx, mockID, serverResponseID, *UpdateServerResponseInput)` | Update a mock server's server response. |
| `DeleteServerResponse(ctx, mockID, serverResponseID)` | Delete a mock server's server response. |

### List

```go
all, err := client.Mocks().List(ctx, &mocks.ListInput{
	Workspace: "workspace-id", // if both TeamID and Workspace are set, only Workspace is sent
})
```

A `nil` or empty input returns all mock servers you created across all
workspaces.

### Create

```go
mock, err := client.Mocks().Create(ctx, &mocks.CreateInput{
	Workspace:   "workspace-id", // empty creates it in your oldest personal Internal workspace
	Collection:  "collection-id",
	Environment: "environment-id",
	Name:        "My Mock",
	Private:     true,
})
```

### Get

```go
mock, err := client.Mocks().Get(ctx, "MOCK_ID")
fmt.Println(mock.MockURL, mock.Config.MatchBody)
```

### Update

```go
mock, err := client.Mocks().Update(ctx, "MOCK_ID", &mocks.UpdateInput{
	Name:              "Renamed Mock",
	ServerResponseID:  "SERVER_RESPONSE_ID", // sets the mock's active server-level response
})
```

`Update` behaves like a partial update: only non-zero fields are sent.

### Delete

```go
deleted, err := client.Mocks().Delete(ctx, "MOCK_ID")
fmt.Println(deleted.ID, deleted.UID)
```

### CallLogs

```go
result, err := client.Mocks().CallLogs(ctx, "MOCK_ID", &mocks.CallLogsInput{
	Limit:              50,
	Sort:               mocks.CallLogSortServedAt,
	Direction:          mocks.SortDirectionDesc,
	ResponseStatusCode: 200,
})
// result.CallLogs []CallLog, result.NextCursor
```

A single call returns at most 6.5MB or 100 call logs, whichever limit is
reached first.

### Publish

```go
published, err := client.Mocks().Publish(ctx, "MOCK_ID")
```

### Unpublish

```go
unpublished, err := client.Mocks().Unpublish(ctx, "MOCK_ID")
```

### ServerResponses

```go
responses, err := client.Mocks().ServerResponses(ctx, "MOCK_ID")
for _, r := range responses {
	fmt.Println(r.ID, r.Name, r.StatusCode)
}
```

### CreateServerResponse

```go
sr, err := client.Mocks().CreateServerResponse(ctx, "MOCK_ID", &mocks.CreateServerResponseInput{
	Name:       "Service Unavailable",
	StatusCode: 503,
	Language:   mocks.ServerResponseLanguageJSON,
	Body:       `{"error":"unavailable"}`,
})
```

Only one server response can be active at a time; if set as active, all
calls to the mock server return this response.

### ServerResponse

```go
sr, err := client.Mocks().ServerResponse(ctx, "MOCK_ID", "SERVER_RESPONSE_ID")
```

### UpdateServerResponse

```go
sr, err := client.Mocks().UpdateServerResponse(ctx, "MOCK_ID", "SERVER_RESPONSE_ID",
	&mocks.UpdateServerResponseInput{StatusCode: 500})
```

### DeleteServerResponse

```go
_, err := client.Mocks().DeleteServerResponse(ctx, "MOCK_ID", "SERVER_RESPONSE_ID")
```

## Reference

Source: [`mocks/`](https://github.com/grokify/postman-go/tree/main/mocks) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/mocks)

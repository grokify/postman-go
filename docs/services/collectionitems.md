# Collection Items

The Collection Items package manages the individual items (folders,
requests, and responses) nested inside a Postman collection. Each item is
addressed by its own ID plus the ID of the collection that contains it.
Reachable via `client.CollectionItems()`.

!!! note "Limited field coverage"
    This package covers each item's basic properties (name, description,
    URL, method, headers, and similar). It does not model the more elaborate
    Postman Collection Format structures (auth, pre-request and test
    scripts, form-data/GraphQL bodies) — those are left untouched by
    Create/Update calls made through this package. See the
    [Postman Collection Format v2.1.0 schema](https://schema.postman.com/collection/json/v2.1.0/draft-07/docs/index.html)
    for the full format.

## Quick Example

```go
import (
	"context"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/collectionitems"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// Create a folder at the collection root.
folder, err := client.CollectionItems().CreateFolder(ctx, "collection-id",
	&collectionitems.CreateFolderInput{Name: "New Folder"})

// Create a request inside that folder.
req, err := client.CollectionItems().CreateRequest(ctx, "collection-id",
	&collectionitems.CreateRequestInput{
		FolderID: folder.Folder.ID,
		Name:     "Get Widget",
		Method:   "GET",
		URL:      "https://api.example.com/widgets/1",
	})

// Save an example response for that request.
resp, err := client.CollectionItems().CreateResponse(ctx, "collection-id",
	&collectionitems.CreateResponseInput{RequestID: req.Request.ID})
```

## Methods

| Method | Description |
|--------|-------------|
| `CreateFolder(ctx, collectionID, *CreateFolderInput)` | Create a folder in a collection. |
| `GetFolder(ctx, collectionID, folderID, *GetOptions)` | Get information about a folder in a collection. |
| `UpdateFolder(ctx, collectionID, folderID, *UpdateFolderInput)` | Update a folder's name and description. |
| `DeleteFolder(ctx, collectionID, folderID)` | Delete a folder in a collection. |
| `CreateRequest(ctx, collectionID, *CreateRequestInput)` | Create a request in a collection. |
| `GetRequest(ctx, collectionID, requestID, *GetOptions)` | Get information about a request in a collection. |
| `UpdateRequest(ctx, collectionID, requestID, *UpdateRequestInput)` | Update a request's properties. |
| `DeleteRequest(ctx, collectionID, requestID)` | Delete a request in a collection. |
| `CreateResponse(ctx, collectionID, *CreateResponseInput)` | Create a saved example response for a request in a collection. |
| `GetResponse(ctx, collectionID, responseID, *GetOptions)` | Get information about a response in a collection. |
| `UpdateResponse(ctx, collectionID, responseID, *UpdateResponseInput)` | Update a response's properties. |
| `DeleteResponse(ctx, collectionID, responseID)` | Delete a response in a collection. |

### CreateFolder

```go
result, err := client.CollectionItems().CreateFolder(ctx, "COLLECTION_ID",
	&collectionitems.CreateFolderInput{
		Name:   "New Folder",
		Folder: "PARENT_FOLDER_ID", // empty creates it at the collection's root
	})
// result.Folder, result.ModelID, result.Revision
```

It is recommended to set `Name`; otherwise Postman creates the folder with
a blank name.

### GetFolder

```go
result, err := client.CollectionItems().GetFolder(ctx, "COLLECTION_ID", "FOLDER_ID",
	&collectionitems.GetOptions{Populate: true})
fmt.Println(result.Folder.Name, result.Folder.Requests)
```

### UpdateFolder

```go
result, err := client.CollectionItems().UpdateFolder(ctx, "COLLECTION_ID", "FOLDER_ID",
	&collectionitems.UpdateFolderInput{Name: "Renamed Folder"})
```

This behaves like a PATCH: only non-zero fields are sent, and moving the
folder to a different parent is not supported by this endpoint.

### DeleteFolder

```go
deleted, err := client.CollectionItems().DeleteFolder(ctx, "COLLECTION_ID", "FOLDER_ID")
fmt.Println(deleted.ID, deleted.Owner)
```

### CreateRequest

```go
result, err := client.CollectionItems().CreateRequest(ctx, "COLLECTION_ID",
	&collectionitems.CreateRequestInput{
		FolderID: "FOLDER_ID", // empty creates it at the collection's root
		Name:     "Get Widget",
		Method:   "GET",
		URL:      "https://api.example.com/widgets/1",
		Headers: []collectionitems.Header{
			{Key: "Accept", Value: "application/json"},
		},
	})
```

It is recommended to set `Name`; otherwise Postman creates the request with
a blank name.

### GetRequest

```go
result, err := client.CollectionItems().GetRequest(ctx, "COLLECTION_ID", "REQUEST_ID", nil)
fmt.Println(result.Request.Name)
```

### UpdateRequest

```go
result, err := client.CollectionItems().UpdateRequest(ctx, "COLLECTION_ID", "REQUEST_ID",
	&collectionitems.UpdateRequestInput{
		Method: "POST",
		DataMode: "raw",
		RawModeData: `{"name":"updated"}`,
	})
```

This endpoint does not support moving the request to a different folder.

### DeleteRequest

```go
deleted, err := client.CollectionItems().DeleteRequest(ctx, "COLLECTION_ID", "REQUEST_ID")
```

### CreateResponse

```go
result, err := client.CollectionItems().CreateResponse(ctx, "COLLECTION_ID",
	&collectionitems.CreateResponseInput{
		RequestID: "REQUEST_ID", // required
		Name:      "200 OK",
		Status:    "OK",
	})
```

It is recommended to set `Name`; otherwise Postman creates the response with
a blank name.

### GetResponse

```go
result, err := client.CollectionItems().GetResponse(ctx, "COLLECTION_ID", "RESPONSE_ID", nil)
fmt.Println(result.Response.Request) // parent request ID
```

### UpdateResponse

```go
result, err := client.CollectionItems().UpdateResponse(ctx, "COLLECTION_ID", "RESPONSE_ID",
	&collectionitems.UpdateResponseInput{Name: "Renamed Response"})
```

### DeleteResponse

```go
deleted, err := client.CollectionItems().DeleteResponse(ctx, "COLLECTION_ID", "RESPONSE_ID")
```

## Reference

Source: [`collectionitems/`](https://github.com/grokify/postman-go/tree/main/collectionitems) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/collectionitems)

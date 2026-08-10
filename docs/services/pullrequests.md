# Pull Requests

Pull requests let collection collaborators propose, review, and merge changes
between forked collections. Reachable via `client.PullRequests()`. See
[Reviewing pull requests](https://learning.postman.com/docs/collaborating-in-postman/using-version-control/reviewing-pull-requests/)
for background.

## Quick Example

```go
import (
	"context"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/pullrequests"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// Get a pull request's current state.
pr, err := client.PullRequests().Get(ctx, "6")

// Approve it.
_, err = client.PullRequests().Review(ctx, "6", &pullrequests.ReviewInput{
	Action: pullrequests.ReviewActionApprove,
})
```

## Methods

| Method | Description |
|--------|-------------|
| `Get(ctx, pullRequestID)` | Get information about a pull request: source and destination, reviewers, and merge status. |
| `Update(ctx, pullRequestID, *UpdateInput)` | Update an open pull request's title, description, or reviewers. |
| `Review(ctx, pullRequestID, *ReviewInput)` | Approve, decline, merge, or unapprove a pull request. |

### Get

```go
pr, err := client.PullRequests().Get(ctx, "6")
fmt.Println(pr.Title, pr.Status, pr.Merge.Status)
for _, rv := range pr.Reviewers {
	fmt.Println(rv.ID, rv.Status)
}
```

### Update

```go
result, err := client.PullRequests().Update(ctx, "6", &pullrequests.UpdateInput{
	Title:       "Add new auth endpoints",
	Description: "Adds the OAuth 2.0 token endpoints.",
	Reviewers:   []string{"1234", "5678"},
})
```

### Review

```go
result, err := client.PullRequests().Review(ctx, "6", &pullrequests.ReviewInput{
	Action:  pullrequests.ReviewActionMerge,
	Comment: "LGTM, merging.",
})
// Action also accepts ReviewActionApprove, ReviewActionDecline, ReviewActionUnapprove.
```

## Reference

Source: [`pullrequests/`](https://github.com/grokify/postman-go/tree/main/pullrequests) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/pullrequests)

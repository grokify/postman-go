# Analytics

The Analytics service returns metrics and insights for API usage, success,
and workspace/team trends in Postman, along with a catalog describing which
metrics are available for which resources. Reachable via
`client.Analytics()`.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/analytics"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// Discover which metrics are available for the workspace resource.
meta, err := client.Analytics().Metadata(ctx, &analytics.MetadataInput{
	Resources: "workspace",
})

// Fetch active-workspace counts.
data, err := client.Analytics().Data(ctx, &analytics.DataInput{
	Resource: analytics.ResourceWorkspace,
	Metrics:  analytics.MetricActiveWorkspaces,
	Duration: analytics.DurationLast30Days,
})
fmt.Println(string(data.Data))
```

## Methods

| Method | Description |
|--------|-------------|
| `Data(ctx, *DataInput)` | Return analytics data for a resource/metrics combination, across team, internal, public, and Partner Workspaces. |
| `Metadata(ctx, *MetadataInput)` | Return the catalog of analytics resources and their corresponding metrics for use with `Data`. |

### Data

```go
data, err := client.Analytics().Data(ctx, &analytics.DataInput{
	Resource: analytics.ResourceWorkspace,
	Metrics:  analytics.MetricActiveWorkspaces,
	View:     analytics.ViewTrend,
	Duration: analytics.DurationLast180Days,
	Limit:    50,
	Offset:   0,
})
// data.Data is json.RawMessage; its shape depends on Resource/Metrics/View.
```

`Resource` and `Metrics` are required and must be a valid combination — call
`Metadata` to discover valid resource/metric pairs. Several filters
(`UserID`, `RequestID`, `ResponseStatus`, `AttentionType`) only apply to
specific metrics; see the field docs on `DataInput`.

### Metadata

```go
meta, err := client.Analytics().Metadata(ctx, &analytics.MetadataInput{
	Include:   "parameters,response",
	Resources: "workspace,team",
})
fmt.Println(meta.Description)
for _, r := range meta.Resources {
	fmt.Println(string(r)) // json.RawMessage; shape depends on Include
}
```

A `nil` input returns metadata for all available resources and metrics.

## Reference

Source: [`analytics/`](https://github.com/grokify/postman-go/tree/main/analytics) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/analytics)

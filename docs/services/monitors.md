# Monitors

Monitors run a collection on a schedule (optionally from multiple regions)
and report on the results, so teams can be alerted when an API starts
failing or degrading. Reachable via `client.Monitors()`.

!!! note "Two response shapes are approximated"
    `Run`'s response body is a union the generated client cannot resolve to
    a concrete shape, so it collapses to an empty result — call `Get` to
    retrieve the monitor's `LastRun` details after running it. Also, a run
    that exceeds 300 seconds returns an HTTP 202 with a body the client has
    no case for, which surfaces as an error from `Run`; pass `Async: true`
    to avoid triggering it.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/monitors"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

// Create a monitor that runs a collection every hour.
created, err := client.Monitors().Create(ctx, &monitors.CreateInput{
	WorkspaceID: "ws-id",
	Name:        "Nightly health check",
	Collection:  "COLLECTION_ID",
	Active:      true,
	Schedule: monitors.Schedule{
		Cron:     "0 0 * * * *",
		Timezone: "America/Los_Angeles",
	},
})

// List monitors in the workspace.
list, err := client.Monitors().List(ctx, &monitors.ListInput{WorkspaceID: "ws-id"})
for _, m := range list.Monitors {
	fmt.Println(m.ID, m.Name, m.Active)
}

// Trigger a run.
_, err = client.Monitors().Run(ctx, created.ID, &monitors.RunInput{Async: true})
```

## Methods

| Method | Description |
|--------|-------------|
| `List(ctx, *ListInput)` | List the monitors visible to the caller. Paginated. |
| `Create(ctx, *CreateInput)` | Create a monitor. |
| `Get(ctx, monitorID)` | Get full detail about a monitor, including its last run. |
| `Update(ctx, monitorID, *UpdateInput)` | Update a monitor's configuration. |
| `Delete(ctx, monitorID)` | Delete a monitor. |
| `Run(ctx, monitorID, *RunInput)` | Run a monitor. See the limitation note above. |
| `RunnerInstances(ctx, runnerID, *RunnerInstancesInput)` | List instances of a runner polling for monitor runs. Paginated. |
| `RunnerMetrics(ctx, runnerID)` | Get server-side metrics (queue depth, last ping) for a runner instance. |

### List

```go
list, err := client.Monitors().List(ctx, &monitors.ListInput{
	WorkspaceID: "ws-id",
	Active:      true,
	Limit:       25,
})
for _, m := range list.Monitors {
	fmt.Println(m.ID, m.Name, m.CollectionUID)
}
```

### Create

```go
created, err := client.Monitors().Create(ctx, &monitors.CreateInput{
	WorkspaceID: "ws-id",
	Name:        "Nightly health check",
	Collection:  "COLLECTION_ID",
	Environment: "ENVIRONMENT_ID",
	Active:      true,
	Schedule: monitors.Schedule{
		Cron:     "0 0 * * * *",
		Timezone: "America/Los_Angeles",
	},
	Distribution: []monitors.Region{monitors.RegionUsEast, monitors.RegionEuCentral},
	Notifications: &monitors.Notifications{
		OnFailure: []monitors.NotificationEmail{{Email: "oncall@example.com"}},
	},
})
// created.ID, created.UID, created.Active
```

You cannot create monitors for collections added to an API definition. If
`WorkspaceID` is empty, Postman creates the monitor in the oldest personal
Internal workspace you own.

### Get

```go
got, err := client.Monitors().Get(ctx, "MONITOR_ID")
fmt.Println(got.Schedule.Cron, got.LastRun.Status, got.LastRun.Stats.RequestsFailed)
```

### Update

```go
active := true
_, err := client.Monitors().Update(ctx, "MONITOR_ID", &monitors.UpdateInput{
	Active: &active,
	Schedule: &monitors.Schedule{
		Cron:     "0 */6 * * * *",
		Timezone: "America/Los_Angeles",
	},
})
```

Zero-valued fields (and a `nil` `Schedule`/`Retry`/`Options`/`Notifications`)
are left unchanged; use `Active` to explicitly activate or deactivate the
monitor.

### Delete

```go
_, err := client.Monitors().Delete(ctx, "MONITOR_ID")
```

### Run

```go
_, err := client.Monitors().Run(ctx, "MONITOR_ID", &monitors.RunInput{Async: true})
// Then poll for results:
got, err := client.Monitors().Get(ctx, "MONITOR_ID")
fmt.Println(got.LastRun.Status)
```

### RunnerInstances

```go
instances, err := client.Monitors().RunnerInstances(ctx, "RUNNER_ID", &monitors.RunnerInstancesInput{Limit: 25})
for _, ri := range instances.Instances {
	fmt.Println(ri.ID, ri.Hostname, ri.LastPingedAt)
}
```

### RunnerMetrics

```go
metrics, err := client.Monitors().RunnerMetrics(ctx, "RUNNER_ID")
fmt.Println(metrics.QueueDepth, metrics.OldestQueuedRunAgeSeconds)
```

## Reference

Source: [`monitors/`](https://github.com/grokify/postman-go/tree/main/monitors) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/monitors)

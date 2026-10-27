# Exporting Findings to OCSF

The `ocsf` package converts Secret Scanner results into
[OCSF](https://schema.ocsf.io/) (Open Cybersecurity Schema Framework)
**Detection Finding** events (`class_uid` 2004). Use it to send Postman
secret exposures to a SIEM, a data lake, or any tool that reads OCSF.

The package only maps data. It makes no HTTP calls and stores nothing:
you fetch results with `client.SecretScanner()`, convert them with `ocsf`,
and send the events wherever you like. The OCSF types come from
[`ocsf-schema-go`](https://github.com/grokify/ocsf-schema-go).

!!! note "Requires an add-on"
    The Secret Scanner API needs a Postman Enterprise plan with the Advanced
    Security Administration add-on — see
    [Authentication](authentication.md#enterprise-add-ons).

## Quick Example

```go
import (
	"context"
	"encoding/json"
	"os"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/ocsf"
	"github.com/grokify/postman-go/secretscanner"
)

client, err := postman.NewClient()
if err != nil {
	return err
}
ctx := context.Background()

res, err := client.SecretScanner().Query(ctx, &secretscanner.QueryInput{
	Statuses: []secretscanner.Resolution{secretscanner.ResolutionActive},
})
if err != nil {
	return err
}

findings := ocsf.FromQueryResult(res, nil)

enc := json.NewEncoder(os.Stdout)
for _, f := range findings {
	if err := enc.Encode(f); err != nil { // one OCSF event per line (NDJSON)
		return err
	}
}
```

## Choosing a Mapper

| Function | Input | Use when |
|----------|-------|----------|
| `FromQueryResult(res, opts)` | a whole `*QueryResult` | You want one finding per detected secret, quickly. |
| `FromDetectedSecret(s, opts)` | one `DetectedSecret` | Same as above, one secret at a time. |
| `FromDetectedSecretWithLocations(s, locs, opts)` | a secret and its `Locations` | You want `resources[]` to list every place the secret appears. |
| `FromDetectedSecretEnriched(s, locs, enrich, opts)` | a secret, its locations, and lookups | You want resource names, owners, and remediation text. Best for a remediation console. |
| `FromActivityFeed(secretID, entries, opts)` | a secret's resolution history | You want an event for each status change (revoked, false positive, ...). |
| `FromActivityFeedEntry(secretID, e, opts)` | one history entry | Same as above, one entry at a time. |

Every mapper returns plain `*events.DetectionFinding` structs. You can change
any field before you send them.

## Adding Locations

The query endpoint tells you *that* a secret was found in a workspace. The
locations endpoint tells you *where*. Pass both to get one OCSF resource per
location, and a deep link in `finding_info.src_url`:

```go
for _, s := range res.Secrets {
	locs, err := client.SecretScanner().Locations(ctx, s.SecretID,
		&secretscanner.LocationsInput{WorkspaceID: s.WorkspaceID})
	if err != nil {
		return err
	}
	f := ocsf.FromDetectedSecretWithLocations(s, locs.Locations, nil)
	// f.Resources has one entry per location.
	// f.FindingInfo.SrcURL links to the first location.
}
```

## Enrichment

The Secret Scanner returns numeric user IDs and resource IDs, not names. To
put names in the finding, look them up yourself (for example with
`client.Users()` and `client.Collections()`) and pass them in an
`*ocsf.Enrichment`:

```go
enrich := &ocsf.Enrichment{
	// LeakedBy user ID -> resources[].owner
	Users: map[int]ocsf.Owner{
		12345: {Name: "Pat Example", Email: "pat@example.com"},
	},
	// resource ID -> resources[].name
	ResourceNames: map[string]string{
		"COLLECTION_ID": "Payments API",
	},
	// Optional per-type guidance. Return nil to use the default text.
	Remediation: func(secretType string) *ocsf.RemediationInfo {
		if secretType == "GitHub Personal Access Token" {
			return &ocsf.RemediationInfo{
				Desc:       "Revoke the token in GitHub settings, then issue a new one.",
				References: []string{"https://docs.github.com/en/authentication"},
			}
		}
		return nil
	},
}

f := ocsf.FromDetectedSecretEnriched(s, locs.Locations, enrich, nil)
```

Every `Enrichment` field is optional, and a `nil` `*Enrichment` gives the same
result as the plain mappers. Without a `Remediation` function, each finding
gets generic advice: rotate the secret, revoke the exposed value, and move it
into a variable or vault.

## Resolution History

`Locations` also returns the secret's activity feed: each time someone
changed its status. `FromActivityFeed` turns each change into its own event.
All of them use the secret ID as `finding_info.uid`, so they line up with the
original finding:

```go
history := ocsf.FromActivityFeed(s.SecretID, locs.ActivityFeed, nil)
```

A change to `REVOKED`, `FALSE_POSITIVE`, or `ACCEPTED_RISK` becomes activity
**Close**. A change back to `ACTIVE` becomes **Update**. When the entry names
who made the change, it goes in `actor.user`.

## How Fields Are Mapped

**Status** comes from the Postman resolution:

| Postman resolution | OCSF `status_id` |
|--------------------|------------------|
| `ACTIVE` | New (1) |
| `REVOKED` | Resolved (4) |
| `FALSE_POSITIVE` | Suppressed (3) |
| `ACCEPTED_RISK` | Suppressed (3) |

**Severity** is a simple default based on who can see the workspace. Only
active secrets get a severity above Informational:

| Workspace visibility | OCSF `severity_id` |
|----------------------|--------------------|
| `public` | Critical |
| `partner` | High |
| `team` (or unknown) | Medium |
| `personal` / `private` | Low |
| any, once resolved or suppressed | Informational |

If you have your own risk model, overwrite `SeverityID` (and `RiskScore` or
`RiskLevelID`) on the finding before you send it.

**Other fields:**

- `finding_info.uid` is the Postman secret ID.
- `finding_info.tags` holds the secret type, the masked secret, and the
  secret hash. The secret value itself is never included.
- `count` is the number of occurrences.
- Each resource's `namespace` is the workspace ID, and its `labels` include
  `workspace_visibility:<value>`. When no specific resource is known, the
  workspace itself is the resource.
- `time` is the detection time in epoch milliseconds, or `0` when Postman
  gives no time.

## Options

```go
opts := &ocsf.Options{
	ProductName: "Postman Secret Scanner (prod team)", // metadata.product.name
}
findings := ocsf.FromQueryResult(res, opts)
```

| Field | Sets | Default |
|-------|------|---------|
| `ProductName` | `metadata.product.name` | `Postman Secret Scanner` |
| `SchemaVersion` | `metadata.version` | `events.SchemaVersion`, the OCSF version `ocsf-schema-go` was generated from |

A `nil` `*Options` uses all the defaults.

## Reference

Source: [`ocsf/`](https://github.com/grokify/postman-go/tree/main/ocsf) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/ocsf) ·
[OCSF Detection Finding](https://schema.ocsf.io/classes/detection_finding)

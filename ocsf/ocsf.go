// Package ocsf maps Postman Secret Scanner findings to OCSF (Open
// Cybersecurity Schema Framework) Detection Finding events (class_uid 2004).
//
// It is a pure mapping layer: it depends only on the Postman types in
// github.com/grokify/postman-go/secretscanner and the OCSF types in
// github.com/grokify/ocsf-schema-go, and knows nothing about storage. Callers
// persist the returned events however they like — e.g. by passing them to an
// ocsf-lake instance:
//
//	findings := ocsf.FromQueryResult(res, nil)
//	if err := lake.IngestTyped(ctx, toAnySlice(findings)...); err != nil { ... }
//
// The returned *events.DetectionFinding values are plain, fully-mutable
// structs. This package populates a faithful, schema-valid baseline; a caller
// that computes richer risk (for example a separate analytics package) can
// overwrite fields such as SeverityID, RiskScore, and RiskLevelID on the result
// before persisting.
package ocsf

import (
	"strconv"
	"time"

	"github.com/grokify/ocsf-schema-go/ocsf/events"
	"github.com/grokify/ocsf-schema-go/ocsf/objects"
	"github.com/grokify/postman-go/secretscanner"
)

// Default product identity written to metadata.product and finding_info.product.
// ocsf-lake derives a stored event's Source from metadata.product.name, so this
// is also the default lake source for mapped findings.
const (
	defaultProductName = "Postman Secret Scanner"
	productVendorName  = "Postman"
	productFeatureName = "Secret Scanner"
)

// Options configures the mapping. A nil *Options uses the defaults.
type Options struct {
	// ProductName overrides metadata.product.name (default
	// "Postman Secret Scanner"). It becomes the stored event's lake Source.
	ProductName string
	// SchemaVersion overrides metadata.version (default events.SchemaVersion,
	// the OCSF schema version the vendored ocsf-schema-go was generated from).
	SchemaVersion string
}

func (o *Options) withDefaults() Options {
	out := Options{ProductName: defaultProductName, SchemaVersion: events.SchemaVersion}
	if o != nil {
		if o.ProductName != "" {
			out.ProductName = o.ProductName
		}
		if o.SchemaVersion != "" {
			out.SchemaVersion = o.SchemaVersion
		}
	}
	return out
}

// FromQueryResult maps every secret in a Secret Scanner query result to an OCSF
// Detection Finding. It is a convenience wrapper over FromDetectedSecret.
func FromQueryResult(res *secretscanner.QueryResult, opts *Options) []*events.DetectionFinding {
	if res == nil {
		return nil
	}
	out := make([]*events.DetectionFinding, 0, len(res.Secrets))
	for _, s := range res.Secrets {
		out = append(out, FromDetectedSecret(s, opts))
	}
	return out
}

// FromDetectedSecret maps a single detected secret to an OCSF Detection Finding
// with activity "Create" (a point-in-time report that the finding exists as of
// the sync). The finding carries one resource, derived from the secret's own
// resource reference; use FromDetectedSecretWithLocations to attach the full
// set of locations where the secret was found.
func FromDetectedSecret(s secretscanner.DetectedSecret, opts *Options) *events.DetectionFinding {
	return fromDetectedSecret(s, nil, nil, opts)
}

// FromDetectedSecretWithLocations maps a detected secret to an OCSF Detection
// Finding whose resources[] enumerate every location where the secret was
// found (as returned by the Secret Scanner locations endpoint). When locs is
// empty it behaves like FromDetectedSecret.
func FromDetectedSecretWithLocations(s secretscanner.DetectedSecret, locs []secretscanner.SecretLocation, opts *Options) *events.DetectionFinding {
	return fromDetectedSecret(s, locs, nil, opts)
}

// FromDetectedSecretEnriched maps a detected secret and its locations to an OCSF
// Detection Finding with remediation context resolved by the caller: human
// resource names, deep-link URLs, and the leaking user's name/email. See
// Enrichment. This is the recommended entry point for a remediation console.
func FromDetectedSecretEnriched(s secretscanner.DetectedSecret, locs []secretscanner.SecretLocation, enrich *Enrichment, opts *Options) *events.DetectionFinding {
	return fromDetectedSecret(s, locs, enrich, opts)
}

func fromDetectedSecret(s secretscanner.DetectedSecret, locs []secretscanner.SecretLocation, enrich *Enrichment, opts *Options) *events.DetectionFinding {
	o := opts.withDefaults()
	detected := parseTimeMillis(s.DetectedAt)
	product := newProduct(o.ProductName)

	fi := newFindingInfo(s, detected, product)
	// The primary deep link: the first location URL, surfaced as the finding's
	// source URL so a console can link straight to the exposed resource.
	for _, loc := range locs {
		if loc.URL != "" {
			fi.SrcURL = loc.URL
			break
		}
	}

	f := &events.DetectionFinding{
		ActivityID:  events.DetectionFindingActivityIDCreate,
		CategoryUID: events.CategoryUIDFindings,
		ClassUID:    events.ClassUIDDetectionFinding,
		TypeUID:     int64(events.ClassUIDDetectionFinding)*100 + events.DetectionFindingActivityIDCreate,
		Time:        detected,
		SeverityID:  severityIDFor(s.Resolution, s.WorkspaceVisibility),
		StatusID:    statusIDFor(s.Resolution),
		Status:      statusFor(s.Resolution),
		Count:       s.Occurrences,
		Metadata: &objects.Metadata{
			Version: o.SchemaVersion,
			Product: product,
		},
		FindingInfo: fi,
		Resources:   resourcesFor(s, locs, enrich),
		Remediation: remediationFor(s.SecretType, enrich),
	}
	return f
}

// FromActivityFeed maps a secret's resolution-history entries to OCSF Detection
// Findings, one per entry. Each models a resolution change as its own
// write-once event: activity "Close" when the entry moves the finding to a
// resolved/suppressed state, otherwise "Update". All entries share the secret's
// UID in finding_info.uid so they correlate with the base finding.
func FromActivityFeed(secretID string, entries []secretscanner.ActivityFeedEntry, opts *Options) []*events.DetectionFinding {
	if len(entries) == 0 {
		return nil
	}
	out := make([]*events.DetectionFinding, 0, len(entries))
	for _, e := range entries {
		out = append(out, FromActivityFeedEntry(secretID, e, opts))
	}
	return out
}

// FromActivityFeedEntry maps a single resolution-history entry to an OCSF
// Detection Finding. The entry carries only the resolved status, time, and
// actor, so the resulting finding is sparse by design — it records the
// transition, correlated to the secret by finding_info.uid.
func FromActivityFeedEntry(secretID string, e secretscanner.ActivityFeedEntry, opts *Options) *events.DetectionFinding {
	o := opts.withDefaults()
	occurred := parseTimeMillis(e.ResolvedAt)
	product := newProduct(o.ProductName)
	activityID := activityIDFor(e.Status)

	f := &events.DetectionFinding{
		ActivityID:  activityID,
		CategoryUID: events.CategoryUIDFindings,
		ClassUID:    events.ClassUIDDetectionFinding,
		TypeUID:     int64(events.ClassUIDDetectionFinding)*100 + int64(activityID),
		Time:        occurred,
		SeverityID:  severityIDFor(e.Status, ""),
		StatusID:    statusIDFor(e.Status),
		Status:      statusFor(e.Status),
		Metadata: &objects.Metadata{
			Version: o.SchemaVersion,
			Product: product,
		},
		FindingInfo: &objects.FindingInfo{
			UID:          secretID,
			Title:        "Postman secret resolution updated",
			Product:      product,
			DataSources:  []string{defaultProductName},
			ModifiedTime: occurred,
		},
	}
	if e.ResolvedBy != 0 {
		// OCSF's user object requires one of {uid, name, account}; a Postman
		// user is a numeric ID, so set both the string uid and uid_numeric.
		f.Actor = &objects.Actor{User: &objects.User{
			UID:        strconv.Itoa(e.ResolvedBy),
			UIDNumeric: int64(e.ResolvedBy),
		}}
	}
	return f
}

// activityIDFor chooses the OCSF activity for a resolution change: a move to a
// resolved/suppressed state is a Close; anything else (e.g. back to active) is
// an Update.
func activityIDFor(r secretscanner.Resolution) int {
	if r == secretscanner.ResolutionActive || r == "" {
		return events.DetectionFindingActivityIDUpdate
	}
	return events.DetectionFindingActivityIDClose
}

// newFindingInfo builds the required finding_info object for a detected secret.
func newFindingInfo(s secretscanner.DetectedSecret, detected int64, product *objects.Product) *objects.FindingInfo {
	fi := &objects.FindingInfo{
		UID:           s.SecretID,
		Title:         findingTitle(s),
		Desc:          findingDesc(s),
		Types:         []string{"Exposed Secret"},
		Product:       product,
		DataSources:   []string{defaultProductName},
		CreatedTime:   detected,
		FirstSeenTime: detected,
	}
	if s.SecretType != "" {
		fi.Tags = append(fi.Tags, objects.KeyValueObject{Name: "secret_type", Value: s.SecretType})
	}
	if s.ObfuscatedSecret != "" {
		fi.Tags = append(fi.Tags, objects.KeyValueObject{Name: "obfuscated_secret", Value: s.ObfuscatedSecret})
	}
	if s.SecretHash != "" {
		fi.Tags = append(fi.Tags, objects.KeyValueObject{Name: "secret_hash", Value: s.SecretHash})
	}
	return fi
}

// resourcesFor builds the resources[] for a detected secret. When locations are
// supplied each becomes its own resource; otherwise the secret's single
// resource reference is used. enrich (may be nil) supplies human resource names
// and resolved owners.
func resourcesFor(s secretscanner.DetectedSecret, locs []secretscanner.SecretLocation, enrich *Enrichment) []objects.ResourceDetails {
	if len(locs) == 0 {
		r := resourceFrom(s.ResourceType, s.ResourceID, s.WorkspaceID, s.WorkspaceVisibility, nil)
		applyEnrichment(&r, s.ResourceID, 0, enrich)
		return []objects.ResourceDetails{r}
	}
	out := make([]objects.ResourceDetails, 0, len(locs))
	for _, loc := range locs {
		data := map[string]any{}
		if loc.URL != "" {
			data["url"] = loc.URL
		}
		if loc.IsResourceDeleted {
			data["is_resource_deleted"] = true
		}
		r := resourceFrom(loc.ResourceType, loc.ResourceID, s.WorkspaceID, s.WorkspaceVisibility, data)
		applyEnrichment(&r, loc.ResourceID, loc.LeakedBy, enrich)
		out = append(out, r)
	}
	return out
}

// applyEnrichment overlays caller-resolved context onto a resource: a human
// name (resources[].name) and the leaking user as the resource owner
// (resources[].owner). Nil-safe.
func applyEnrichment(r *objects.ResourceDetails, resourceID string, leakedBy int, enrich *Enrichment) {
	if enrich == nil {
		return
	}
	if name, ok := enrich.ResourceNames[resourceID]; ok && name != "" {
		r.Name = name
	}
	if leakedBy != 0 {
		if o, ok := enrich.Users[leakedBy]; ok && (o.Name != "" || o.Email != "") {
			r.Owner = &objects.User{
				UID:       strconv.Itoa(leakedBy),
				FullName:  o.Name,
				EmailAddr: o.Email,
			}
		}
	}
}

// resourceFrom builds one OCSF resource. The Postman workspace is modeled as
// the resource namespace (its container), and workspace visibility as a label.
//
// The Secret Scanner query endpoint groups results by workspace and omits the
// per-resource type/ID (those come from the locations endpoint), so when no
// specific resource is identified the workspace itself is used as the resource.
// OCSF requires every resource to carry at least one of {name, uid}; this
// guarantees a uid is always present.
func resourceFrom(resourceType, resourceID, workspaceID, visibility string, data map[string]any) objects.ResourceDetails {
	r := objects.ResourceDetails{
		Type:      resourceType,
		UID:       resourceID,
		Namespace: workspaceID,
	}
	if r.UID == "" {
		// No specific resource identified: the workspace is the resource.
		r.UID = workspaceID
		if r.Type == "" {
			r.Type = "workspace"
		}
		r.Namespace = ""
	}
	if visibility != "" {
		r.Labels = []string{"workspace_visibility:" + visibility}
	}
	if len(data) > 0 {
		r.Data = data
	}
	return r
}

func newProduct(name string) *objects.Product {
	return &objects.Product{
		Name:       name,
		VendorName: productVendorName,
		Feature:    &objects.Feature{Name: productFeatureName},
	}
}

// statusIDFor maps a Postman resolution to an OCSF finding status_id.
//
//	ACTIVE        -> New         (1) — open, unremediated
//	REVOKED       -> Resolved    (4) — secret rotated/revoked
//	FALSE_POSITIVE-> Suppressed  (3) — not a real secret
//	ACCEPTED_RISK -> Suppressed  (3) — risk acknowledged, not remediated
func statusIDFor(r secretscanner.Resolution) int {
	switch r {
	case secretscanner.ResolutionRevoked:
		return events.DetectionFindingStatusIDResolved
	case secretscanner.ResolutionFalsePositive, secretscanner.ResolutionAcceptedRisk:
		return events.DetectionFindingStatusIDSuppressed
	case secretscanner.ResolutionActive:
		return events.DetectionFindingStatusIDNew
	default:
		return events.DetectionFindingStatusIDNew
	}
}

// statusFor returns the human-readable status string paired with statusIDFor.
func statusFor(r secretscanner.Resolution) string {
	return events.DetectionFindingStatusIDCaptions[statusIDFor(r)]
}

// severityIDFor returns a coarse default severity for an active finding, scaled
// by how broadly the containing workspace is exposed: a live secret in a public
// workspace is Critical, a partner workspace High, a team workspace Medium, and
// a personal/private workspace Low. Resolved or suppressed findings are
// Informational. This is intentionally simple — callers with a real risk model
// should overwrite SeverityID (and RiskScore/RiskLevelID) on the returned
// finding.
func severityIDFor(r secretscanner.Resolution, visibility string) int {
	if r != secretscanner.ResolutionActive && r != "" {
		return events.DetectionFindingSeverityIDInformational
	}
	switch visibility {
	case "public":
		return events.DetectionFindingSeverityIDCritical
	case "partner":
		return events.DetectionFindingSeverityIDHigh
	case "personal", "private":
		return events.DetectionFindingSeverityIDLow
	default: // team, unknown, empty
		return events.DetectionFindingSeverityIDMedium
	}
}

func findingTitle(s secretscanner.DetectedSecret) string {
	name := s.SecretType
	if name == "" {
		name = "Secret"
	}
	if s.ResourceType != "" {
		return name + " exposed in Postman " + s.ResourceType
	}
	return name + " exposed in Postman"
}

func findingDesc(s secretscanner.DetectedSecret) string {
	name := s.SecretType
	if name == "" {
		name = "A secret"
	}
	return "Postman Secret Scanner detected " + name + " in a Postman workspace resource."
}

// parseTimeMillis parses an RFC 3339 timestamp to epoch milliseconds, returning
// 0 when the input is empty or unparseable (the OCSF time field is always
// present; 0 signals an unknown time rather than omitting it).
func parseTimeMillis(s string) int64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}

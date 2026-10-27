package ocsf_test

import (
	"encoding/json"
	"testing"

	"github.com/grokify/ocsf-schema-go/ocsf/events"
	"github.com/grokify/ocsf-schema-go/ocsf/ocsfdata"
	"github.com/grokify/ocsf-schema-go/validate"
	"github.com/grokify/postman-go/ocsf"
	"github.com/grokify/postman-go/secretscanner"
)

// sampleSecret mirrors the shape of a /detected-secrets-queries row, with
// synthetic IDs.
func sampleSecret() secretscanner.DetectedSecret {
	return secretscanner.DetectedSecret{ //nolint:gosec // G101: synthetic test fixture; ObfuscatedSecret is a masked public sample prefix, not a credential
		SecretID:            "c2VjcmV0LTE=",
		SecretType:          "GitHub Personal Access Token",
		SecretHash:          "0000000000000000000000000000000000000000000000000000000000000001",
		ObfuscatedSecret:    "ghp_16C7e4******",
		Resolution:          secretscanner.ResolutionActive,
		WorkspaceID:         "11111111-1111-4111-8111-111111111111",
		WorkspaceVisibility: "team",
		ResourceType:        "collection",
		ResourceID:          "22222222-2222-4222-8222-222222222222",
		DetectedAt:          "2026-08-12T15:36:51.000Z",
		Occurrences:         1,
	}
}

// validateFinding marshals a finding and validates it against the embedded OCSF
// schema — the same schema ocsf-lake enforces on ingest.
func validateFinding(t *testing.T, f *events.DetectionFinding) {
	t.Helper()
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal finding: %v", err)
	}
	v := validate.New(ocsfdata.MustSchema())
	res := v.ValidateEvent(data)
	if !res.Valid() {
		for _, iss := range res.Errors() {
			t.Errorf("schema issue: %s", iss.String())
		}
		t.Fatalf("finding failed OCSF schema validation\npayload: %s", data)
	}
}

func TestFromDetectedSecret_SchemaValid(t *testing.T) {
	f := ocsf.FromDetectedSecret(sampleSecret(), nil)

	if f.ClassUID != events.ClassUIDDetectionFinding {
		t.Errorf("ClassUID = %d, want %d", f.ClassUID, events.ClassUIDDetectionFinding)
	}
	if f.CategoryUID != events.CategoryUIDFindings {
		t.Errorf("CategoryUID = %d, want %d", f.CategoryUID, events.CategoryUIDFindings)
	}
	if f.ActivityID != events.DetectionFindingActivityIDCreate {
		t.Errorf("ActivityID = %d, want Create", f.ActivityID)
	}
	wantType := int64(events.ClassUIDDetectionFinding)*100 + events.DetectionFindingActivityIDCreate
	if f.TypeUID != wantType {
		t.Errorf("TypeUID = %d, want %d", f.TypeUID, wantType)
	}
	if f.StatusID != events.DetectionFindingStatusIDNew {
		t.Errorf("StatusID = %d, want New", f.StatusID)
	}
	if f.SeverityID != events.DetectionFindingSeverityIDMedium {
		t.Errorf("SeverityID = %d, want Medium", f.SeverityID)
	}
	if f.FindingInfo == nil || f.FindingInfo.UID != "c2VjcmV0LTE=" {
		t.Errorf("FindingInfo.UID mismatch: %+v", f.FindingInfo)
	}
	if f.Metadata == nil || f.Metadata.Product == nil || f.Metadata.Product.Name != "Postman Secret Scanner" {
		t.Errorf("Metadata.Product.Name mismatch: %+v", f.Metadata)
	}
	if len(f.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(f.Resources))
	}
	if r := f.Resources[0]; r.UID != "22222222-2222-4222-8222-222222222222" || r.Type != "collection" || r.Namespace != "11111111-1111-4111-8111-111111111111" {
		t.Errorf("resource mismatch: %+v", r)
	}
	if f.Count != 1 {
		t.Errorf("Count = %d, want 1", f.Count)
	}

	validateFinding(t, f)
}

// The query endpoint groups by workspace and omits resourceType/resourceID, so
// the resource must fall back to the workspace to satisfy OCSF's
// resources[]: one of {name, uid} constraint.
func TestFromDetectedSecret_NoResourceFallsBackToWorkspace(t *testing.T) {
	s := sampleSecret()
	s.ResourceType = ""
	s.ResourceID = ""
	f := ocsf.FromDetectedSecret(s, nil)
	if len(f.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(f.Resources))
	}
	r := f.Resources[0]
	if r.UID != s.WorkspaceID {
		t.Errorf("resource UID = %q, want workspace ID %q", r.UID, s.WorkspaceID)
	}
	if r.Type != "workspace" {
		t.Errorf("resource Type = %q, want workspace", r.Type)
	}
	validateFinding(t, f)
}

func TestResolutionMapping(t *testing.T) {
	cases := []struct {
		res        secretscanner.Resolution
		wantStatus int
		wantSev    int
	}{
		{secretscanner.ResolutionActive, events.DetectionFindingStatusIDNew, events.DetectionFindingSeverityIDMedium},
		{secretscanner.ResolutionRevoked, events.DetectionFindingStatusIDResolved, events.DetectionFindingSeverityIDInformational},
		{secretscanner.ResolutionFalsePositive, events.DetectionFindingStatusIDSuppressed, events.DetectionFindingSeverityIDInformational},
		{secretscanner.ResolutionAcceptedRisk, events.DetectionFindingStatusIDSuppressed, events.DetectionFindingSeverityIDInformational},
	}
	for _, c := range cases {
		s := sampleSecret()
		s.Resolution = c.res
		f := ocsf.FromDetectedSecret(s, nil)
		if f.StatusID != c.wantStatus {
			t.Errorf("%s: StatusID = %d, want %d", c.res, f.StatusID, c.wantStatus)
		}
		if f.SeverityID != c.wantSev {
			t.Errorf("%s: SeverityID = %d, want %d", c.res, f.SeverityID, c.wantSev)
		}
		validateFinding(t, f)
	}
}

func TestFromDetectedSecretWithLocations(t *testing.T) {
	locs := []secretscanner.SecretLocation{
		{ResourceType: "collection", ResourceID: "col-1", URL: "https://go.postman.co/x", Occurrences: 1},
		{ResourceType: "environment", ResourceID: "env-2", IsResourceDeleted: true, Occurrences: 2},
	}
	f := ocsf.FromDetectedSecretWithLocations(sampleSecret(), locs, nil)
	if len(f.Resources) != 2 {
		t.Fatalf("Resources len = %d, want 2", len(f.Resources))
	}
	if f.Resources[0].UID != "col-1" || f.Resources[1].UID != "env-2" {
		t.Errorf("resource UIDs mismatch: %+v", f.Resources)
	}
	validateFinding(t, f)
}

func TestFromQueryResult(t *testing.T) {
	res := &secretscanner.QueryResult{
		Secrets: []secretscanner.DetectedSecret{sampleSecret(), sampleSecret()},
	}
	findings := ocsf.FromQueryResult(res, nil)
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2", len(findings))
	}
	for _, f := range findings {
		validateFinding(t, f)
	}

	if got := ocsf.FromQueryResult(nil, nil); got != nil {
		t.Errorf("FromQueryResult(nil) = %v, want nil", got)
	}
}

func TestFromActivityFeed(t *testing.T) {
	entries := []secretscanner.ActivityFeedEntry{
		{ResolvedAt: "2026-08-13T10:00:00.000Z", ResolvedBy: 57569581, Status: secretscanner.ResolutionRevoked},
		{ResolvedAt: "2026-08-14T10:00:00.000Z", ResolvedBy: 57569581, Status: secretscanner.ResolutionActive},
	}
	findings := ocsf.FromActivityFeed("c2VjcmV0LTE=", entries, nil)
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2", len(findings))
	}
	if findings[0].ActivityID != events.DetectionFindingActivityIDClose {
		t.Errorf("revoked entry ActivityID = %d, want Close", findings[0].ActivityID)
	}
	if findings[1].ActivityID != events.DetectionFindingActivityIDUpdate {
		t.Errorf("active entry ActivityID = %d, want Update", findings[1].ActivityID)
	}
	for _, f := range findings {
		if f.FindingInfo.UID != "c2VjcmV0LTE=" {
			t.Errorf("FindingInfo.UID = %q, want secret UID", f.FindingInfo.UID)
		}
		if f.Actor == nil || f.Actor.User == nil || f.Actor.User.UID != "57569581" || f.Actor.User.UIDNumeric != 57569581 {
			t.Errorf("actor mismatch: %+v", f.Actor)
		}
		validateFinding(t, f)
	}
}

func TestOptionsOverride(t *testing.T) {
	f := ocsf.FromDetectedSecret(sampleSecret(), &ocsf.Options{ProductName: "psa", SchemaVersion: "1.3.0"})
	if f.Metadata.Product.Name != "psa" {
		t.Errorf("product name = %q, want psa", f.Metadata.Product.Name)
	}
	if f.Metadata.Version != "1.3.0" {
		t.Errorf("schema version = %q, want 1.3.0", f.Metadata.Version)
	}
}

func TestFromDetectedSecretEnriched(t *testing.T) {
	s := sampleSecret()
	s.WorkspaceVisibility = "public" // should escalate severity to Critical

	locs := []secretscanner.SecretLocation{
		{
			ResourceType: "collection",
			ResourceID:   "22222222-2222-4222-8222-222222222222",
			URL:          "https://go.postman.co/workspace/11111111/collection/22222222",
			LeakedBy:     57569581,
			Occurrences:  1,
		},
	}
	enrich := &ocsf.Enrichment{
		Users:         map[int]ocsf.Owner{57569581: {Name: "John Wang", Email: "jwang@example.com"}},
		ResourceNames: map[string]string{"22222222-2222-4222-8222-222222222222": "Example API"},
	}

	f := ocsf.FromDetectedSecretEnriched(s, locs, enrich, nil)

	if f.SeverityID != events.DetectionFindingSeverityIDCritical {
		t.Errorf("SeverityID = %d, want Critical (public workspace)", f.SeverityID)
	}
	if f.FindingInfo.SrcURL != "https://go.postman.co/workspace/11111111/collection/22222222" {
		t.Errorf("SrcURL = %q, want the location deep link", f.FindingInfo.SrcURL)
	}
	if f.Remediation == nil || f.Remediation.Desc == "" {
		t.Error("Remediation missing")
	}
	if len(f.Resources) != 1 {
		t.Fatalf("Resources len = %d, want 1", len(f.Resources))
	}
	r := f.Resources[0]
	if r.Name != "Example API" {
		t.Errorf("resource Name = %q, want enriched collection name", r.Name)
	}
	if r.Owner == nil || r.Owner.FullName != "John Wang" || r.Owner.EmailAddr != "jwang@example.com" || r.Owner.UID != "57569581" {
		t.Errorf("resource Owner mismatch: %+v", r.Owner)
	}
	data, _ := r.Data.(map[string]any)
	if data["url"] != "https://go.postman.co/workspace/11111111/collection/22222222" {
		t.Errorf("resource data.url mismatch: %+v", r.Data)
	}
	validateFinding(t, f)
}

func TestSeverityByVisibility(t *testing.T) {
	cases := map[string]int{
		"public":   events.DetectionFindingSeverityIDCritical,
		"partner":  events.DetectionFindingSeverityIDHigh,
		"team":     events.DetectionFindingSeverityIDMedium,
		"personal": events.DetectionFindingSeverityIDLow,
		"private":  events.DetectionFindingSeverityIDLow,
	}
	for vis, want := range cases {
		s := sampleSecret()
		s.WorkspaceVisibility = vis
		if got := ocsf.FromDetectedSecret(s, nil); got.SeverityID != want {
			t.Errorf("visibility %q: severity = %d, want %d", vis, got.SeverityID, want)
		}
	}
}

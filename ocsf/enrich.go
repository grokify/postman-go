package ocsf

import "github.com/grokify/ocsf-schema-go/ocsf/objects"

// Enrichment carries caller-resolved remediation context that the Postman
// Secret Scanner API does not return directly. The Secret Scanner gives numeric
// user IDs and resource IDs; the caller resolves them (via the Postman Users and
// Collections APIs) and passes the results here, where they populate native OCSF
// fields: resources[].owner and resources[].name. Keeping resolution in the
// caller preserves this package as a pure, HTTP-free mapper.
//
// All fields are optional; a nil *Enrichment yields the same result as the
// unenriched mappers.
type Enrichment struct {
	// Users maps a Postman user ID (a location's LeakedBy) to display info,
	// written to resources[].owner as the user who exposed the secret.
	Users map[int]Owner
	// ResourceNames maps a resource ID to a human name (e.g. a collection name),
	// written to resources[].name.
	ResourceNames map[string]string
	// Remediation, when set, returns guidance for a secret type. When it returns
	// nil, or is itself nil, a generic default is used.
	Remediation func(secretType string) *RemediationInfo
}

// Owner is a resolved Postman user for resources[].owner.
type Owner struct {
	Name  string
	Email string
}

// RemediationInfo is caller-supplied remediation guidance for a secret type.
type RemediationInfo struct {
	Desc       string
	References []string
}

// remediationFor returns the OCSF remediation for a secret type, preferring
// caller-supplied guidance and falling back to a generic default.
func remediationFor(secretType string, enrich *Enrichment) *objects.Remediation {
	if enrich != nil && enrich.Remediation != nil {
		if r := enrich.Remediation(secretType); r != nil {
			return &objects.Remediation{Desc: r.Desc, References: r.References}
		}
	}
	return defaultRemediation(secretType)
}

// defaultRemediation is the generic rotate/revoke/remove guidance applied when
// no per-type guidance is supplied.
func defaultRemediation(secretType string) *objects.Remediation {
	name := secretType
	if name == "" {
		name = "the exposed secret"
	}
	return &objects.Remediation{
		Desc: "Rotate " + name + " at its provider, revoke the exposed value, and remove it from the Postman resource — " +
			"use an environment variable or a vault reference instead of a hard-coded secret.",
		References: []string{"https://learning.postman.com/docs/administration/secret-scanner/"},
	}
}

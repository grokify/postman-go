# API Security

The API Security service analyzes an API definition against your team's
configured API governance rulesets — including Postman's OWASP security
rules, if enabled — and reports any violations found. Reachable via
`client.APISecurity()`.

!!! note "Requires OWASP rules to be enabled"
    You must import and enable Postman's OWASP security rules for `Validate`
    to return security rule violations. It can be integrated into CI/CD to
    automate schema validation.

## Quick Example

```go
import (
	"context"
	"encoding/json"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/apisecurity"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

result, err := client.APISecurity().Validate(ctx, &apisecurity.ValidateInput{
	Type:     apisecurity.SchemaTypeOpenAPI3,
	Language: apisecurity.SchemaLanguageJSON,
	Schema:   openapiJSON,
})
for _, w := range result.Warnings {
	fmt.Println(string(w))
}
```

## Methods

| Method | Description |
|--------|-------------|
| `Validate(ctx, *ValidateInput)` | Analyze an API definition and return any issues found based on the team's configured governance rulesets. |

### Validate

```go
result, err := client.APISecurity().Validate(ctx, &apisecurity.ValidateInput{
	Type:     apisecurity.SchemaTypeOpenAPI3,
	Language: apisecurity.SchemaLanguageYaml,
	Schema:   openapiYAML, // stringified definition, max 10 MB
})
for _, w := range result.Warnings {
	var issue struct {
		Severity        string `json:"severity"`
		Category        string `json:"category"`
		PossibleFixURL  string `json:"possibleFixUrl"`
	}
	if err := json.Unmarshal(w, &issue); err != nil {
		// handle error
	}
	fmt.Println(issue.Severity, issue.Category)
}
```

All `ValidateInput` fields are optional; an empty input validates an empty
definition. `Type` accepts `SchemaTypeOpenAPI3` or `SchemaTypeOpenAPI2`;
`Language` accepts `SchemaLanguageJSON` or `SchemaLanguageYaml`. Each
`Warnings` entry is raw JSON — the schema isn't fixed, so unmarshal only the
fields you need (severity, category, location, data paths, and an optional
`possibleFixUrl`).

## Reference

Source: [`apisecurity/`](https://github.com/grokify/postman-go/tree/main/apisecurity) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/apisecurity)

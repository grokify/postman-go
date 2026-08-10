# Postbot

The Postbot service generates AI agent tool code from a public collection
request in Postman's Public API Network. Reachable via `client.Postbot()`.

!!! warning "Deprecated upstream"
    `GenerateTool` is deprecated by Postman. It only supports public Postman
    collections and requests, is rate-limited to 300 calls every 3 hours,
    and does not accrue Postbot usage. Access requires no special plan.

## Quick Example

```go
import (
	"context"
	"fmt"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/postbot"
)

client, _ := postman.NewClient(postman.WithAPIKey(key))
ctx := context.Background()

result, err := client.Postbot().GenerateTool(ctx, &postbot.GenerateToolInput{
	RequestID:      "req-id",
	CollectionID:   "collection-id",
	Language:       postbot.LanguagePython,
	AgentFramework: postbot.AgentFrameworkOpenAI,
})
fmt.Println(result.Text)
```

## Methods

| Method | Description |
|--------|-------------|
| `GenerateTool(ctx, *GenerateToolInput)` | Generate AI agent tool code from a public collection request. Deprecated upstream by Postman. |

### GenerateTool

```go
result, err := client.Postbot().GenerateTool(ctx, &postbot.GenerateToolInput{
	RequestID:      "req-id",
	CollectionID:   "collection-id",
	Language:       postbot.LanguageTypeScript,
	AgentFramework: postbot.AgentFrameworkLangChain,
})
// result.Text is the generated tool code.
```

`Language` accepts `LanguageJavaScript`, `LanguageTypeScript`, or
`LanguagePython`. `AgentFramework` accepts `AgentFrameworkOpenAI`,
`AgentFrameworkMistral`, `AgentFrameworkGemini`, `AgentFrameworkAnthropic`,
`AgentFrameworkLangChain`, or `AgentFrameworkAutogen` — note that
`AgentFrameworkAutogen` only supports `LanguagePython`.

## Reference

Source: [`postbot/`](https://github.com/grokify/postman-go/tree/main/postbot) ·
[GoDoc](https://pkg.go.dev/github.com/grokify/postman-go/postbot)

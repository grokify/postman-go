# Installation

Add postman-go as a dependency to your Go project:

```bash
go get github.com/grokify/postman-go
```

Import the top-level package plus any service packages whose input/result
types or enums you reference directly:

```go
import (
	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/secretscanner" // only if you use secretscanner types
)
```

You only need to import a service's subpackage when you construct its
`*Input` types or reference its enums directly — `client.Collections()`,
`client.Workspaces()`, and so on are all available straight off
`*postman.Client` without any additional imports.

## Requirements

- Go 1.25 or later
- A [Postman API key](https://learning.postman.com/docs/developer/postman-api/authentication/)

See [Authentication](../guides/authentication.md) for how to configure the
client, then [Quick Start](quickstart.md) for a full working example.

// Command secretscanner demonstrates the Postman Secret Scanner SDK.
//
// Usage:
//
//	POSTMAN_API_KEY=PMAK-... go run ./examples/secretscanner
package main

import (
	"context"
	"fmt"
	"log"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/secretscanner"
)

func main() {
	client, err := postman.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// List the secret types supported by the scanner.
	types, err := client.SecretScanner().SecretTypes(ctx)
	if err != nil {
		log.Fatalf("SecretTypes: %v", err)
	}
	fmt.Printf("Supported secret types: %d\n", types.Total)
	for _, t := range types.Types {
		fmt.Printf("  - %s (%s, %s)\n", t.Name, t.ID, t.Origin)
	}

	// Query all detected secrets, including the total count.
	result, err := client.SecretScanner().Query(ctx, &secretscanner.QueryInput{
		Limit:        50,
		IncludeTotal: true,
	})
	if err != nil {
		log.Fatalf("Query: %v", err)
	}
	fmt.Printf("\nDetected secrets (total %d):\n", result.Total)
	for _, s := range result.Secrets {
		fmt.Printf("  - %s [%s] in workspace %s (%d occurrences)\n",
			s.SecretType, s.Resolution, s.WorkspaceID, s.Occurrences)
	}
}

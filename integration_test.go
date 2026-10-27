package postman_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	postman "github.com/grokify/postman-go"
	"github.com/grokify/postman-go/postmanerr"
)

// newIntegrationClient returns a client backed by the live Postman API using
// the POSTMAN_API_KEY environment variable. Tests using it are skipped when
// the variable is unset, so `go test ./...` stays offline by default.
func newIntegrationClient(t *testing.T) *postman.Client {
	t.Helper()
	if _, err := postman.NewClient(); errors.Is(err, postman.ErrNoAPIKey) {
		t.Skip("POSTMAN_API_KEY not set; skipping integration test")
	}
	c, err := postman.NewClient()
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// TestIntegrationMe exercises GET /me against the live API.
func TestIntegrationMe(t *testing.T) {
	c := newIntegrationClient(t)

	me, err := c.Users().Me(context.Background())
	if err != nil {
		t.Fatalf("Users().Me: %v", err)
	}
	if me.User.ID == 0 {
		t.Error("User.ID = 0, want nonzero")
	}
	if me.User.TeamID == 0 {
		t.Error("User.TeamID = 0, want nonzero")
	}
}

// TestIntegrationWorkspacesList exercises GET /workspaces against the live API.
func TestIntegrationWorkspacesList(t *testing.T) {
	c := newIntegrationClient(t)

	list, err := c.Workspaces().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("Workspaces().List: %v", err)
	}
	t.Logf("got %d workspaces", len(list.Workspaces))
}

// TestIntegrationCollectionsList exercises GET /collections against the live API.
func TestIntegrationCollectionsList(t *testing.T) {
	c := newIntegrationClient(t)

	list, err := c.Collections().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("Collections().List: %v", err)
	}
	t.Logf("got %d collections", len(list.Collections))
}

// TestIntegrationTeamsGet exercises the me -> teamID -> Teams().Get flow: the
// team ID is only discoverable via /me, since Teams().Get has no "get my own
// team" shortcut. Some plans don't have Teams API access at all, in which case
// the API returns a well-formed 400 APIError rather than team data — both
// outcomes are valid here, since this test verifies the SDK plumbs the ID
// through and decodes the response correctly either way.
func TestIntegrationTeamsGet(t *testing.T) {
	c := newIntegrationClient(t)
	ctx := context.Background()

	me, err := c.Users().Me(ctx)
	if err != nil {
		t.Fatalf("Users().Me: %v", err)
	}

	team, err := c.Teams().Get(ctx, strconv.Itoa(me.User.TeamID), nil)
	if err != nil {
		var apiErr *postmanerr.APIError
		if errors.As(err, &apiErr) {
			t.Logf("Teams().Get returned APIError (status %d): %s", apiErr.StatusCode, apiErr.Detail)
			return
		}
		t.Fatalf("Teams().Get: unexpected error type: %v", err)
	}
	if team.ID == 0 {
		t.Error("Team.ID = 0, want nonzero")
	}
}

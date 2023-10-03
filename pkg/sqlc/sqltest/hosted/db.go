package hosted

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/tkcrm/pgxgen/pkg/sqlc/quickdb"
	pb "github.com/tkcrm/pgxgen/pkg/sqlc/quickdb/v1"
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/sqlpath"
)

var client pb.QuickClient
var once sync.Once

func initClient() error {
	projectID := os.Getenv("CI_SQLC_PROJECT_ID")
	authToken := os.Getenv("CI_SQLC_AUTH_TOKEN")
	if projectID == "" || authToken == "" {
		return fmt.Errorf("missing project id or auth token")
	}
	c, err := quickdb.NewClient(projectID, authToken)
	if err != nil {
		return err
	}
	client = c
	return nil
}

func PostgreSQL(t *testing.T, migrations []string) string {
	ctx := context.Background()
	t.Helper()

	once.Do(func() {
		if err := initClient(); err != nil {
			t.Log(err)
		}
	})

	if client == nil {
		t.Skip("client init failed")
	}

	var seed []string
	files, err := sqlpath.Glob(migrations)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		blob, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		seed = append(seed, string(blob))
	}

	resp, err := client.CreateEphemeralDatabase(ctx, &pb.CreateEphemeralDatabaseRequest{
		Engine:     "postgresql",
		Region:     quickdb.GetClosestRegion(),
		Migrations: seed,
	})
	if err != nil {
		t.Fatalf("region %s: %s", quickdb.GetClosestRegion(), err)
	}

	t.Cleanup(func() {
		_, err = client.DropEphemeralDatabase(ctx, &pb.DropEphemeralDatabaseRequest{
			DatabaseId: resp.DatabaseId,
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	return resp.Uri
}

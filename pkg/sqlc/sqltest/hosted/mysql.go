package hosted

import (
	"context"
	"os"
	"testing"

	"github.com/tkcrm/pgxgen/pkg/sqlc/quickdb"
	pb "github.com/tkcrm/pgxgen/pkg/sqlc/quickdb/v1"
	"github.com/tkcrm/pgxgen/pkg/sqlc/sql/sqlpath"
)

func MySQL(t *testing.T, migrations []string) string {
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
		Engine:     "mysql",
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

	uri, err := quickdb.MySQLReformatURI(resp.Uri)
	if err != nil {
		t.Fatalf("uri error: %s", err)
	}

	return uri
}

package hosted

import (
	"fmt"
	"os"
	"sync"

	"github.com/tkcrm/pgxgen/pkg/sqlc/quickdb"
	pb "github.com/tkcrm/pgxgen/pkg/sqlc/quickdb/v1"
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

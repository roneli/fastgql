package execution

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/roneli/fastgql/pkg/schema"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupPostgres creates an instance of the postgres container type
func setupPostgres(ctx context.Context) (testcontainers.Container, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:14-alpine",
		Env:          map[string]string{},
		ExposedPorts: []string{"5432/tcp"},
		Cmd:          []string{"postgres", "-c", "fsync=off"},
		WaitingFor:   wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}
	return container, nil
}

func cleanupTestDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".go" { // if the file has a .go extension
			err := os.Remove(path) // delete the file
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func TestPostgresGraph(t *testing.T) {
	require.Nil(t, cleanupTestDirectory("."))
	// first generate the graphQL server code
	err := schema.Generate("gqlgen.yml", true)

	require.Nil(t, err)

}

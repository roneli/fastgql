package mongo

import (
	"context"
	"log"

	"github.com/roneli/fastgql/pkg/execution/builders"
	mb "github.com/roneli/fastgql/pkg/execution/builders/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go.mongodb.org/mongo-driver/mongo"
)

// Driver is a dialect.Driver implementation for SQL based databases.
type Driver struct {
	client  *mongo.Client
	builder mb.Builder
	dialect string
}

// NewDriver creates a new Driver with the given Conn and dialect.
func NewDriver(dialect string, cfg *builders.Config, uri string) *Driver {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	return &Driver{dialect: dialect, builder: mb.NewBuilder(cfg), client: client}
}

func (d Driver) Scan(ctx context.Context, model interface{}) error {
	field := builders.CollectFields(ctx, d.builder.Schema)
	table := field.Table()
	db := d.client.Database(table.Schema)
	collection := db.Collection(table.Name)
	pipeline, err := d.builder.Query(field)
	if err != nil {
		return err
	}
	cur, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = cur.Close(ctx)
	}()
	return cur.All(ctx, model)
}

func (d Driver) Close() error {
	return d.client.Disconnect(context.Background())
}

func (d Driver) Dialect() string {
	return d.dialect
}

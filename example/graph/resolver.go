package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/roneli/fastgql/builders"
)

type Resolver struct {
	Cfg *builders.Config
	Sql *pgxpool.Pool
}

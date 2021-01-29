package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"context"
	"fastgql/builders"
	pgx "github.com/jackc/pgx/v4"
)

type Resolver struct{
	Cfg *builders.Config
	Sql SqlRepo
}

type SqlRepo interface {
	Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
}

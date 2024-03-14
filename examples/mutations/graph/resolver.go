package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/roneli/fastgql/pkg/execution/builders"
)

type Resolver struct {
	Cfg      *builders.Config
	Executor pgxscan.Querier
}

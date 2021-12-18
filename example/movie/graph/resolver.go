package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"github.com/roneli/fastgql/builders"
	"github.com/roneli/fastgql/execution"
)

type Resolver struct {
	Cfg      *builders.Config
	Executor execution.Querier
}

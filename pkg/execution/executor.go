package execution

import (
	"context"
	"github.com/roneli/fastgql/pkg/execution/builders"
)

type Executor interface {
	Execute(ctx context.Context, field builders.Field, dst any) error
}

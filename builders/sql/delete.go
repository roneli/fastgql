package sql

import (
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
)

type deleteHelper struct {
	*goqu.DeleteDataset
	table exp.AliasedExpression
	alias string
}

func (d deleteHelper) Table() tableHelper {
	return tableHelper{
		table: d.table,
		alias: d.alias,
	}
}

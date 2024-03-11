package schema

type ArgName string

const (
	GroupBy     ArgName = "groupBy"
	FilterInput ArgName = "filter"
	OrderBy     ArgName = "orderBy"
)

const (
	generateDirectiveName     = "generate"
	skipGenerateDirectiveName = "skipGenerate"
	tableDirectiveName        = "table"
	relationDirectiveName     = "relation"
)

package schema


type LogicalOperator string

const (
	LogicalOperatorAND LogicalOperator = "AND"
	LogicalOperatorOR LogicalOperator = "OR"
	LogicalOperatorNot LogicalOperator = "NOT"
)

type BooleanOperator string

const (
	EqualOperator BooleanOperator = "eq"
	NotEqualOperator = "neq"
	InOperator = "in"
	NotInOperator = "notIn"
	NotNull = "notNull"
	Exists = "exists"
)
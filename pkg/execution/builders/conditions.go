package builders

type LogicalOperator string

const (
	LogicalOperatorAND LogicalOperator = "AND"
	LogicalOperatorOR  LogicalOperator = "OR"
	LogicalOperatorNot LogicalOperator = "NOT"
)

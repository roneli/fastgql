package sql

import (
	"fmt"
	"regexp"
)

// pathValidationRegex validates JSON paths with support for multiple array indices
var pathValidationRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\[[0-9]+\])*(\.[a-zA-Z_][a-zA-Z0-9_]*(\[[0-9]+\])*)*$`)

// JSONPathOp represents a JSONPath operator
type JSONPathOp string

// JSONPath comparison operators
const (
	JSONPathEq  JSONPathOp = "=="
	JSONPathNeq JSONPathOp = "!="
	JSONPathGt  JSONPathOp = ">"
	JSONPathGte JSONPathOp = ">="
	JSONPathLt  JSONPathOp = "<"
	JSONPathLte JSONPathOp = "<="
)

// JSONPath regex operator
const JSONPathLikeRegex JSONPathOp = "like_regex"

// JSONPath logical operators
const (
	JSONPathAnd JSONPathOp = "&&"
	JSONPathOr  JSONPathOp = "||"
)

// graphqlToJSONPathOp maps GraphQL operators to JSONPath operators
var graphqlToJSONPathOp = map[string]JSONPathOp{
	"eq":       JSONPathEq,
	"neq":      JSONPathNeq,
	"gt":       JSONPathGt,
	"gte":      JSONPathGte,
	"lt":       JSONPathLt,
	"lte":      JSONPathLte,
	"like":     JSONPathLikeRegex,
	"prefix":   JSONPathLikeRegex,
	"suffix":   JSONPathLikeRegex,
	"ilike":    JSONPathLikeRegex,
	"contains": JSONPathLikeRegex,
}

// invertedJSONPathOp maps operators to their logical inverses for 'all' array filter
var invertedJSONPathOp = map[JSONPathOp]JSONPathOp{
	JSONPathEq:  JSONPathNeq,
	JSONPathNeq: JSONPathEq,
	JSONPathGt:  JSONPathLte,
	JSONPathGte: JSONPathLt,
	JSONPathLt:  JSONPathGte,
	JSONPathLte: JSONPathGt,
}

// knownOperators detects if a map contains operators vs nested field filters
var knownOperators = map[string]bool{
	"eq": true, "neq": true, "gt": true, "gte": true,
	"lt": true, "lte": true, "like": true,
	"ilike":    true,
	"prefix":   true,
	"suffix":   true,
	"contains": true,
	"isNull":   true,
	"any":      true,
	"all":      true,
}

// validatePath validates a JSON path
func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if !pathValidationRegex.MatchString(path) {
		return fmt.Errorf("invalid path format: %s", path)
	}
	return nil
}

// isOperatorMap checks if a map contains operators vs nested field filters
func isOperatorMap(m map[string]any) bool {
	for k := range m {
		if knownOperators[k] {
			return true
		}
	}
	return false
}

// escapeRegexPattern escapes special regex characters
func escapeRegexPattern(pattern string) string {
	return regexp.QuoteMeta(pattern)
}

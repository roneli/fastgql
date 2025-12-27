# JSON Filter Code Analysis

## Test Failures

### 1. `typed_json_filter_with_NOT` Test Failure

**Expected:**
- JSONPath: `$ ? (!(@.color == $v0))` (negation inside JSONPath)
- SQL: `jsonb_path_exists("sq0"."attributes", $1::jsonpath, $2::jsonb)` (no SQL NOT wrapper)

**Actual:**
- JSONPath: `$ ? (@.color == $v0)` (no negation)
- SQL: `NOT(jsonb_path_exists(...))` (SQL NOT wrapper)

**Root Cause:** In `json_convert.go`, the NOT operator is handled by wrapping the expression with SQL `NOT()` via `BuildLogicalFilter`, but it should instead negate the condition inside the JSONPath expression string.

**Location:** `json_convert.go:165-182` - NOT handling wraps expression instead of negating JSONPath condition

### 2. `typed_json_filter_with_nested_logical_operators` Test Failure

**Expected:**
- Single `jsonb_path_exists` call with complex JSONPath: `$ ? (@.color == $v0 && (@.size > $v2 || @.size < $v1))`
- Single variable map: `{"v0":"red","v1":10,"v2":5}`

**Actual:**
- Multiple `jsonb_path_exists` calls combined with SQL AND/OR
- Multiple variable maps: `{"v0":"red"}`, `{"v0":10,"v1":5}`

**Root Cause:** The code creates separate expressions for each logical group instead of combining nested AND/OR/NOT into a single JSONPath expression.

**Location:** `json_convert.go:49-102` (AND handling) and `104-163` (OR handling) - creates separate expressions instead of combining into one JSONPath

## Code Redundancy

### 1. `processFieldOperatorsV2` Function (Unused)

**Location:** `json_convert.go:385-498`

**Issue:** This function appears to be completely unused. It's similar to `categorizeFieldOperators` but processes operators differently. It should be removed if not needed.

**Evidence:** No references found in codebase via grep.

### 2. Duplication Between `categorizeFieldOperators` and `processFieldOperatorsV2`

Both functions:
- Process field operators
- Handle `any` and `all` array filters
- Handle standard operators (eq, neq, etc.)
- Handle `isNull` checks

**Difference:** `processFieldOperatorsV2` creates individual filters for each operator, while `categorizeFieldOperators` separates simple vs complex operators for optimization.

**Recommendation:** Remove `processFieldOperatorsV2` if unused, or merge logic if both are needed.

### 3. Redundant Filter Creation in `convertFilterMapWithPrefix`

**Location:** `json_convert.go:36-37` and multiple places

**Issue:** Creates `combinedFilter` at the start but also creates individual filters in various places. The logic for combining simple conditions is duplicated.

### 4. `extractSimpleConditions` Complexity

**Location:** `json_convert.go:247-294`

**Issue:** This function attempts to extract simple conditions but has complex logic that might be simplified. It's used in AND/OR optimization but could be more straightforward.

## Simplification Opportunities

### 1. NOT Operator Handling

**Current:** Wraps entire expression with SQL NOT
**Should:** Negate condition inside JSONPath string

**Fix:** Modify `JSONPathConditionExpr.ToJSONPathString()` to support negation, or create a negated version of the condition string when processing NOT.

### 2. Nested Logical Operators

**Current:** Creates separate `jsonb_path_exists` calls for each logical group
**Should:** Combine all conditions into a single JSONPath expression with proper parentheses

**Fix:** When processing AND/OR/NOT, instead of creating separate expressions, build a single `JSONPathFilterExpr` with all conditions and proper logic operators.

### 3. Variable Name Management

**Current:** Each filter manages its own variable names (v0, v1, etc.)
**Issue:** When combining multiple filters, variable names can conflict or be inefficient

**Fix:** Use a shared variable counter or better variable management when combining filters.

### 4. Expression Building Pattern

**Current:** Multiple places create `NewJSONPathFilter`, add conditions, then call `Expression()`
**Simplification:** Could create a helper function to reduce repetition

**Example:**
```go
func buildJSONPathFilter(col exp.IdentifierExpression, conditions []*JSONPathConditionExpr, logic LogicType, dialect Dialect) (exp.Expression, error) {
    filter := NewJSONPathFilter(col, dialect)
    filter.SetLogic(logic)
    for _, cond := range conditions {
        filter.AddCondition(cond)
    }
    return filter.Expression()
}
```

### 5. Remove Unused Code

- `processFieldOperatorsV2` function (if confirmed unused)
- Any other dead code paths

## Additional Findings

### 5. `JSONLogicalExpr` Usage Pattern

**Location:** `json_expr.go:353-401` and used in `json_builder.go` and `json_convert.go`

**Issue:** `JSONLogicalExpr` is used to combine SQL expressions (wrapping `jsonb_path_exists` calls), but for JSON filters, we should combine conditions inside JSONPath expressions instead.

**Current Pattern:**
- Creates separate `jsonb_path_exists` calls
- Combines them with SQL AND/OR using `JSONLogicalExpr`
- Results in: `(jsonb_path_exists(...) AND jsonb_path_exists(...))`

**Should Be:**
- Combines conditions inside single JSONPath
- Results in: `jsonb_path_exists(..., "$ ? (cond1 && cond2)", ...)`

### 6. `wrapORInPar` Flag Complexity

**Location:** `json_expr.go:107, 136, 176`

**Issue:** The `wrapORInPar` flag adds extra parentheses for OR conditions. This seems like a workaround for a specific issue. The logic could be simplified if we understand why this is needed.

**Current:** `$ ? ((cond1 || cond2))` (double parentheses for OR)
**Normal:** `$ ? (cond1 || cond2)` (single parentheses)

### 7. Variable Name Collision Risk

**Location:** Multiple places create variable names (v0, v1, etc.)

**Issue:** When combining filters from different sources (AND/OR branches), variable names could collide. Each `JSONPathFilterExpr` starts from v0, which could cause conflicts when combining.

**Example:** Two filters both use `v0`, `v1` - when combined, they should use `v0-v3` or similar.

## Code Structure Issues

### 8. Mixed Abstraction Levels

The code mixes:
- Low-level JSONPath string building (`ToJSONPathString`)
- High-level filter map conversion (`ConvertFilterMapToExpression`)
- Intermediate expression building (`JSONPathFilterExpr`, `JSONArrayFilterExpr`)

This makes it hard to understand the flow and fix issues.

### 9. Duplicate Logic in AND/OR Handling

**Location:** `json_convert.go:49-102` (AND) and `104-163` (OR)

**Issue:** Both AND and OR handling have nearly identical code:
- Extract simple conditions
- Check if all are simple
- Combine simple conditions into one filter
- Handle complex operators separately

This could be refactored into a shared function.

## Recommended Refactoring Steps

1. **Fix NOT operator:** Modify to negate inside JSONPath instead of SQL wrapper
   - Change `json_convert.go:165-182` to build negated JSONPath condition
   - Update `JSONPathConditionExpr.ToJSONPathString()` or create negated version

2. **Fix nested logical operators:** Combine into single JSONPath expression
   - Modify AND/OR handling to build single `JSONPathFilterExpr` with all conditions
   - Use proper JSONPath syntax: `$ ? (cond1 && (cond2 || cond3))`

3. **Remove `processFieldOperatorsV2`** - confirmed unused (only definition, no calls)

4. **Refactor AND/OR handling** - extract common logic into shared function

5. **Simplify variable management** - use shared counter or better naming when combining filters

6. **Clarify `wrapORInPar`** - document why double parentheses are needed, or remove if unnecessary

7. **Add helper functions** to reduce repetition:
   ```go
   func buildJSONPathFilter(col exp.IdentifierExpression, conditions []*JSONPathConditionExpr, logic LogicType, dialect Dialect) (exp.Expression, error)
   ```

8. **Consider refactoring** to separate concerns:
   - JSONPath string building (low-level)
   - Condition combination (mid-level)  
   - Filter map conversion (high-level)


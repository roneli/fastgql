# JSON Filter Code Review Summary

## Test Failures (2)

### 1. `typed_json_filter_with_NOT`
- **Problem:** NOT wraps SQL function instead of negating JSONPath condition
- **Expected:** `$ ? (!(@.color == $v0))` in single `jsonb_path_exists` call
- **Actual:** `$ ? (@.color == $v0)` wrapped with SQL `NOT()`
- **Fix:** Negate condition inside JSONPath string, not SQL wrapper

### 2. `typed_json_filter_with_nested_logical_operators`
- **Problem:** Creates multiple `jsonb_path_exists` calls instead of one combined
- **Expected:** Single call with `$ ? (@.color == $v0 && (@.size > $v2 || @.size < $v1))`
- **Actual:** Multiple calls combined with SQL AND/OR
- **Fix:** Combine all conditions into single JSONPath expression

## Redundant Code

1. **`processFieldOperatorsV2`** (lines 385-498 in `json_convert.go`)
   - **Status:** Completely unused (only definition, no calls)
   - **Action:** DELETE

2. **Duplicate AND/OR logic** (lines 49-102 and 104-163 in `json_convert.go`)
   - **Issue:** Nearly identical code for AND and OR handling
   - **Action:** Extract to shared function

3. **Old function references**
   - `BuildJsonFilterFromOperatorMap` - replaced by `ConvertFilterMapToExpression`
   - `ParseMapComparator` + `BuildMapFilter` - replaced by `ConvertMapComparatorToExpression`
   - **Status:** Comments indicate replacement, old code removed ✓

## Simplification Opportunities

1. **Variable name management**
   - Each filter starts from `v0`, risks collision when combining
   - **Fix:** Use shared counter or better naming strategy

2. **Filter creation pattern**
   - Repeated pattern: `NewJSONPathFilter` → `AddCondition` → `Expression()`
   - **Fix:** Create helper function to reduce repetition

3. **`wrapORInPar` flag**
   - Adds double parentheses for OR: `$ ? ((cond1 || cond2))`
   - **Action:** Document why needed or simplify

4. **Mixed abstraction levels**
   - Low-level JSONPath strings mixed with high-level filter maps
   - **Suggestion:** Consider clearer separation of concerns

## Key Files

- `json_expr.go` (401 lines) - Expression types and JSONPath building
- `json_convert.go` (680 lines) - Filter map to expression conversion
- `json_builder.go` (299 lines) - Fluent API builder (separate concern)
- `json.go` (104 lines) - Helper functions and operator maps

## Priority Fixes

1. **HIGH:** Fix NOT operator (negate in JSONPath, not SQL)
2. **HIGH:** Fix nested logical operators (combine into single JSONPath)
3. **MEDIUM:** Remove unused `processFieldOperatorsV2`
4. **MEDIUM:** Refactor duplicate AND/OR logic
5. **LOW:** Add helper functions for common patterns
6. **LOW:** Improve variable name management



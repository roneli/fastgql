package schema

type GenerateDirective struct {
	Pagination bool `json:"pagination"`
	Ordering   bool `json:"ordering"`
	Filtering  bool `json:"filtering"`
	Recursive  bool `json:"recursive"`
	Aggregate  bool `json:"aggregate"`
}

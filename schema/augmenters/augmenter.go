package augmenters

import "github.com/vektah/gqlparser/v2/ast"

// Augmenter is a late source injector adding logic into our schema after it's loaded
type Augmenter interface {
	// DirectiveName is the name of the directive this augmenter adds to the Schema, calling this augmenter on every
	// occurrence of this directive.
	DirectiveName() string

	Augment(s *ast.Schema)
}



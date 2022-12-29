package augmenters

import "github.com/vektah/gqlparser/v2/ast"

// Augmenter is a late source injector adding logic into our schema after it's loaded
type Augmenter interface {
	// Name of the augmenter
	Name() string
	// DirectiveName Name is the name of the directive this augmenter adds to the Schema, calling this augmenter on every
	// occurrence of this directive.
	DirectiveName() string
	// Augment is the actual method that gets called to augment the schema
	Augment(s *ast.Schema) error
}

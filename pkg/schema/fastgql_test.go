package schema

import (
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/codegen/config"
	"github.com/stretchr/testify/assert"
)

func Test_Generate(t *testing.T) {
	fgqlPlugin := NewFastGQLPlugin("")
	cfg, err := config.LoadConfig("./test/no_fastgql_gqlgen.yml")
	assert.Nil(t, err)
	assert.Nil(t, cfg.LoadSchema())
	srcs, err := fgqlPlugin.CreateAugmented(cfg.Schema)
	assert.Nil(t, cfg.LoadSchema())
	fmt.Println(srcs)
}

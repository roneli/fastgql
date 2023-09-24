package cmd

import (
	"fmt"
	"os"

	"github.com/roneli/fastgql/pkg/schema"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "generate a graphql server based on schema",
	Long:  `Generate and augment a graphql server based on a schema`,
	Run: func(cmd *cobra.Command, args []string) {
		generateAPI()
	},
}

func generateAPI() {
	if err := schema.Generate(configPath, false, false); err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config", err.Error())
		os.Exit(2)
	}
}

package cmd

import (
	"github.com/roneli/fastgql/schema"
	"fmt"
	"github.com/spf13/cobra"
	"os"
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
	if err := schema.Generate(configPath, false); err != nil{
		fmt.Fprintln(os.Stderr, "failed to load config", err.Error())
		os.Exit(2)
	}
}
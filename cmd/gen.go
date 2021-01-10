package cmd

import (
	"fastgql/schema"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var configPath string

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate fake graphql server from SDL",
	Long:  `Generate a fake graphql server from given SDL`,
	Run: func(cmd *cobra.Command, args []string) {
		generateAPI()
	},
}

func generateAPI() {
	if err := schema.Generate(configPath); err != nil{
		fmt.Fprintln(os.Stderr, "failed to load config", err.Error())
		os.Exit(2)
	}
}
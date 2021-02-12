package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose        bool
	configPath     string
	configFilename string
	schemaFilename string
	serverFilename string

	rootCmd = &cobra.Command{
		Use:   "fastgql",
		Short: "Blazing fast, instant realtime & extendable GraphQL APIs powered by gqlgen",
	}
)

func init() {
	rootCmd.AddCommand(generateCmd, versionCmd, initCmd)
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to server config")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "turns fastgql build log on")
	initCmd.Flags().StringVarP(&schemaFilename, "schemaName", "s", "schema.graphql", "name of schema file")
	initCmd.Flags().StringVarP(&configFilename, "configName", "n", "gqlgen.yml", "name of config file")
	initCmd.Flags().StringVarP(&serverFilename, "serverName", "g", "server.go", "name of server file")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

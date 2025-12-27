package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configPath     string
	configFilename string
	schemaFilename string
	serverFilename string
	saveFiles      bool

	rootCmd = &cobra.Command{
		Use:   "fastgql",
		Short: "Blazing fast, instant realtime & extendable GraphQL APIs powered by gqlgen",
	}
)

func init() {
	rootCmd.AddCommand(generateCmd, versionCmd, initCmd, importCmd)
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "path to server config")
	initCmd.Flags().StringVarP(&schemaFilename, "schemaName", "s", "schema.graphql", "name of schema file")
	initCmd.Flags().StringVarP(&configFilename, "configName", "n", "gqlgen.yml", "name of config file")
	initCmd.Flags().StringVarP(&serverFilename, "serverName", "g", "server.go", "name of server file")
	generateCmd.Flags().BoolVarP(&saveFiles, "saveFiles", "f", false, "save generated files to disk")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

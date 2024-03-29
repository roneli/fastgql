package cmd

import (
	"fmt"

	"github.com/roneli/fastgql/pkg"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "fastgql version",
	Long:  `print the version string`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(pkg.Version)
	},
}

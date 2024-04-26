package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dockem",
	Short: "Use this application to build docker images only when a change has taken place",
	Long: `This tool can be used to look at the build location (or other files) and see
if anything has changed since the last build. If nothing has changed, it will 
copy the tag across to the new repository without having to push. Otherwise, 
it'll trigger a rebuild.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {}

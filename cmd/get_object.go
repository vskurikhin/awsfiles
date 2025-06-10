package cmd

import (
	"github.com/spf13/cobra"

	"github.com/vskurikhin/awsfiles/internal/config"
	"github.com/vskurikhin/awsfiles/internal/object"
)

// getObjectCmd represents the base command when called without any subcommands
var getObjectCmd = &cobra.Command{
	Use:   "get-object",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		setSlogDebug(cmd)
		mergeCobraAndViper(cmd)
		slogInfoVerbose(cmd)
		cfg := config.MakeConfig(cmd)
		object.GetObject(cfg)
	},
}

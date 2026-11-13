package cmd

import (
	"github.com/checkship-deploy/checkship/cmd/config"
	"github.com/checkship-deploy/checkship/cmd/serve"
	"github.com/checkship-deploy/checkship/cmd/version"
	"github.com/checkship-deploy/checkship/pkg/log"
	"github.com/spf13/cobra"
)

// Build the root command where everything start!
func Root() *cobra.Command {
	var (
		conf = config.Default()
		path string
	)

	// Instantiate the logger here since it should be available for sub-commands.
	logger, loggerErr := log.NewLogger()

	rootCmd := &cobra.Command{
		Use:          "checkship",
		SilenceUsage: true,
		Version:      version.Current(),
		Short:        "Painless self-hosting in a single binary.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if loggerErr != nil {
				return loggerErr
			}

			return conf.Initialize(logger, path)
		},
	}

	rootCmd.PersistentFlags().StringVarP(&path, "config", "c", config.DefaultConfigPath, "config file to use")

	// Add sub-commands
	rootCmd.AddCommand(serve.Root(conf, logger))

	return rootCmd
}

package cmd

import (
	"fmt"
	syncer "imagesync/cmd/syncher"
	"os"

	"github.com/spf13/cobra"
)

func getConfig(cmd *cobra.Command) syncer.ProgramCfg {
	logPath, _ := cmd.Flags().GetString("log")
	interval, _ := cmd.Flags().GetDuration("interval")
	root, _ := cmd.Flags().GetString("path")
	maxSizeInKb, _ := cmd.Flags().GetInt64("size")
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			fmt.Printf("Error: %s", err)
			os.Exit(1)
		}
	}
	return syncer.ProgramCfg{
		Interval:    interval,
		LogPath:     logPath,
		Root:        root,
		MaxFileSize: maxSizeInKb * 1024,
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "imagesync",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		cfg := getConfig(cmd)
		p := syncer.NewProgram(cfg)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %s", err)
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.imagesync.yaml)")
	rootCmd.PersistentFlags().StringP("path", "p", "", "path to the watched directory (default is the working directory)")
	rootCmd.PersistentFlags().StringP("log", "l", "", "path to the log file")
	rootCmd.PersistentFlags().Int64("size", 1024, "maximum file size in KB")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

package cmd

import (
	"fmt"
	"time"

	syncer "github.com/flytaly/linksyncer/cmd/syncher"
	"github.com/spf13/cobra"
)

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch for changes in the current directory and update links in Markdown and HTML files",
	Long:  `Watch for changes in the current directory and update links in Markdown and HTML files`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := getConfig(cmd)
		p := syncer.NewProgram(cfg)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error: %s", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// watchCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	watchCmd.Flags().DurationP("interval", "i", 500*time.Millisecond, "poll interval duration (e.g. 1s, 500ms...)")
}

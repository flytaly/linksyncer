package cmd

import (
	"fmt"
	syncer "imagesync/cmd/syncher"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		root, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error: %s", err)
			os.Exit(1)
		}
		// isync := imagesync.New(os.DirFS(root), root)
		// isync.ProcessFiles()
		// isync.Watch(time.Millisecond * 500)

		p := syncer.NewProgram(root, time.Millisecond*500)
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
	// watchCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

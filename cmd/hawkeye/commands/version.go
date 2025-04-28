package commands

import (
	"fmt"
	"runtime"

	"github.com/nemuizzz/hawkeye/pkg/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show hawkeye version information",
	Long:  `Display version information for hawkeye, including build date and git commit.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Hawkeye v%s\n", version.Version)
		fmt.Printf("Build Date: %s\n", version.BuildDate)
		fmt.Printf("Git Commit: %s\n", version.GitCommit)
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	// No flags needed for version command
}

package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	// Flags for list command
	listFormat string
	listGroup  string

	// listCmd represents the list command
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List monitored URLs",
		Long: `List all URLs currently being monitored.
Shows information about monitoring status, groups, and more.`,
		Run: func(cmd *cobra.Command, args []string) {
			configDir, err := getConfigDir()
			if err != nil {
				fmt.Printf("Error getting config directory: %s\n", err)
				return
			}

			configFile := filepath.Join(configDir, "monitors.json")
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				fmt.Println("No monitors found. Use 'hawkeye watch' to add monitors.")
				return
			}

			data, err := os.ReadFile(configFile)
			if err != nil {
				fmt.Printf("Error reading config file: %s\n", err)
				return
			}

			var monitors map[string]MonitorConfig
			if err := json.Unmarshal(data, &monitors); err != nil {
				fmt.Printf("Error parsing config file: %s\n", err)
				return
			}

			if len(monitors) == 0 {
				fmt.Println("No monitors found. Use 'hawkeye watch' to add monitors.")
				return
			}

			fmt.Printf("Found %d monitored URLs:\n\n", len(monitors))

			for url, config := range monitors {
				// Skip if filtering by group and doesn't match
				if listGroup != "" && config.Group != listGroup {
					continue
				}

				if listFormat == "json" {
					jsonOutput, _ := json.MarshalIndent(config, "", "  ")
					fmt.Printf("%s\n", jsonOutput)
				} else {
					fmt.Printf("URL: %s\n", url)
					fmt.Printf("  Interval: %s\n", config.Interval)
					if config.Group != "" {
						fmt.Printf("  Group: %s\n", config.Group)
					}
					if len(config.Headers) > 0 {
						fmt.Printf("  Headers: %v\n", config.Headers)
					}
					if len(config.Ignore) > 0 {
						fmt.Printf("  Ignore: %v\n", config.Ignore)
					}
					if config.NormalizeWhitespace {
						fmt.Printf("  Normalize Whitespace: true\n")
					}
					if config.IgnoreTimestamps {
						fmt.Printf("  Ignore Timestamps: true\n")
					}
					if config.CreatedAt != "" {
						fmt.Printf("  Added: %s\n", config.CreatedAt)
					}
					fmt.Println()
				}
			}

			// List groups if no specific group was requested
			if listGroup == "" {
				groups := make(map[string]int)
				for _, config := range monitors {
					if config.Group != "" {
						groups[config.Group]++
					}
				}

				if len(groups) > 0 {
					fmt.Println("Groups:")
					for group, count := range groups {
						fmt.Printf("  %s: %d URLs\n", group, count)
					}
				}
			}
		},
	}
)

func init() {
	listCmd.Flags().StringVarP(&listFormat, "format", "f", "text", "Output format (text/json)")
	listCmd.Flags().StringVarP(&listGroup, "group", "g", "", "Filter by group name")
}

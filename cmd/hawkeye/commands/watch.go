package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nemuizzz/hawkeye/pkg/monitor"
	"github.com/spf13/cobra"
)

var (
	// Flag variables
	interval            string
	timeout             string
	format              string
	headers             []string
	ignore              []string
	output              string
	group               string
	retryCount          int
	retryInterval       string
	normalizeWhitespace bool
	ignoreTimestamps    bool

	// watchCmd represents the watch command
	watchCmd = &cobra.Command{
		Use:   "watch [URLs...]",
		Short: "Monitor URLs for changes",
		Long: `Watch one or more URLs for changes and report when content changes.
Example:
  hawkeye watch https://example.com --interval 5m`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Error: at least one URL is required")
				cmd.Help()
				os.Exit(1)
			}

			// Parse durations
			intervalDuration, err := time.ParseDuration(interval)
			if err != nil {
				fmt.Printf("Invalid interval: %s\n", err)
				os.Exit(1)
			}

			timeoutDuration, err := time.ParseDuration(timeout)
			if err != nil {
				fmt.Printf("Invalid timeout: %s\n", err)
				os.Exit(1)
			}

			retryIntervalDuration, err := time.ParseDuration(retryInterval)
			if err != nil {
				fmt.Printf("Invalid retry interval: %s\n", err)
				os.Exit(1)
			}

			// Parse headers
			headerMap := make(map[string]string)
			for _, h := range headers {
				// Parse header in format "key:value"
				parts := strings.SplitN(h, ":", 2)
				if len(parts) != 2 {
					fmt.Printf("Warning: invalid header format: %s (expected 'key:value')\n", h)
					continue
				}
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				headerMap[key] = value
			}

			// Create manager for handling multiple URLs
			manager := monitor.NewManager()

			// Create and add monitors for each URL
			for _, url := range args {
				config := &monitor.Config{
					URL:                 url,
					Interval:            intervalDuration,
					Timeout:             timeoutDuration,
					Headers:             headerMap,
					IgnoreSelectors:     ignore,
					Method:              monitor.MethodHash,
					RetryCount:          retryCount,
					RetryInterval:       retryIntervalDuration,
					FollowRedirects:     true,
					NormalizeWhitespace: normalizeWhitespace,
					IgnoreTimestamps:    ignoreTimestamps,
				}

				_, err := manager.AddMonitorWithConfig(config)
				if err != nil {
					fmt.Printf("Error setting up monitor for %s: %s\n", url, err)
					continue
				}

				fmt.Printf("Monitoring %s every %s\n", url, interval)
			}

			// If a group is specified, create it
			if group != "" {
				_, err := manager.CreateGroup(group, "Created via CLI")
				if err != nil {
					fmt.Printf("Error creating group '%s': %s\n", group, err)
				} else {
					// Add all URLs to the group
					for _, url := range args {
						err := manager.AddToGroup(url, group)
						if err != nil {
							fmt.Printf("Error adding %s to group '%s': %s\n", url, group, err)
						}
					}
					fmt.Printf("Added URLs to group: %s\n", group)
				}
			}

			// Save the monitor configurations to a file
			if err := saveMonitors(args, headerMap); err != nil {
				fmt.Printf("Warning: Failed to save monitor configuration: %s\n", err)
			}

			// Start monitoring
			changes := manager.Start()
			fmt.Println("Monitoring started. Press Ctrl+C to stop.")

			// Open output file if specified
			var outputFile *os.File
			if output != "" {
				var err error
				outputFile, err = os.Create(output)
				if err != nil {
					fmt.Printf("Error creating output file: %s\n", err)
					os.Exit(1)
				}
				defer outputFile.Close()
				fmt.Printf("Writing output to file: %s\n", output)
			}

			// Process changes
			for change := range changes {
				if change.Error != "" {
					if format == "json" {
						jsonOutput, _ := json.Marshal(change)
						outputString := string(jsonOutput) + "\n"

						if outputFile != nil {
							outputFile.WriteString(outputString)
						} else {
							fmt.Print(outputString)
						}
					} else {
						outputString := fmt.Sprintf("[ERROR] %s: %s\n", change.URL, change.Error)

						if outputFile != nil {
							outputFile.WriteString(outputString)
						} else {
							fmt.Print(outputString)
						}
					}
					continue
				}

				if change.HasChanged {
					if format == "json" {
						jsonOutput, _ := json.Marshal(change)
						outputString := string(jsonOutput) + "\n"

						if outputFile != nil {
							outputFile.WriteString(outputString)
						} else {
							fmt.Print(outputString)
						}
					} else {
						outputString := fmt.Sprintf("[CHANGED] %s at %s\n", change.URL, change.Timestamp.Format(time.RFC3339))

						if outputFile != nil {
							outputFile.WriteString(outputString)
						} else {
							fmt.Print(outputString)
						}

						if change.Details != "" {
							detailsString := fmt.Sprintf("  Details: %s\n", change.Details)

							if outputFile != nil {
								outputFile.WriteString(detailsString)
							} else {
								fmt.Print(detailsString)
							}
						}

						if change.ContentType != "" {
							typeString := fmt.Sprintf("  Content-Type: %s\n", change.ContentType)

							if outputFile != nil {
								outputFile.WriteString(typeString)
							} else {
								fmt.Print(typeString)
							}
						}

						if change.StatusCode > 0 {
							codeString := fmt.Sprintf("  Status Code: %d\n", change.StatusCode)

							if outputFile != nil {
								outputFile.WriteString(codeString)
							} else {
								fmt.Print(codeString)
							}
						}
					}
				}
			}
		},
	}
)

func init() {
	watchCmd.Flags().StringVarP(&interval, "interval", "i", "5m", "Check interval (e.g., 5m, 1h)")
	watchCmd.Flags().StringVarP(&timeout, "timeout", "t", "30s", "Request timeout")
	watchCmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text/json)")
	watchCmd.Flags().StringArrayVarP(&headers, "header", "H", []string{}, "Custom HTTP headers (key:value)")
	watchCmd.Flags().StringArrayVarP(&ignore, "ignore", "I", []string{}, "CSS selectors to ignore")
	watchCmd.Flags().StringVarP(&output, "output", "o", "", "Output file")
	watchCmd.Flags().StringVarP(&group, "group", "g", "", "Group name for URLs")
	watchCmd.Flags().IntVarP(&retryCount, "retries", "r", 3, "Number of retry attempts")
	watchCmd.Flags().StringVarP(&retryInterval, "retry-interval", "R", "10s", "Time between retries")
	watchCmd.Flags().BoolVarP(&normalizeWhitespace, "normalize", "n", false, "Normalize whitespace to ignore insignificant changes")
	watchCmd.Flags().BoolVarP(&ignoreTimestamps, "ignore-timestamps", "T", false, "Ignore timestamps when comparing content")
}

// saveMonitors saves the monitor configurations to a file
func saveMonitors(urls []string, headers map[string]string) error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	// Create the config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(configDir, "monitors.json")

	// Load existing monitors if the file exists
	var monitors map[string]MonitorConfig
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(data, &monitors); err != nil {
			// If the file is corrupted, start with an empty map
			monitors = make(map[string]MonitorConfig)
		}
	} else {
		monitors = make(map[string]MonitorConfig)
	}

	// Add or update monitors
	for _, url := range urls {
		monitors[url] = MonitorConfig{
			URL:                 url,
			Interval:            interval,
			Group:               group,
			Headers:             headers,
			Ignore:              ignore,
			CreatedAt:           time.Now().Format(time.RFC3339),
			NormalizeWhitespace: normalizeWhitespace,
			IgnoreTimestamps:    ignoreTimestamps,
		}
	}

	// Save to file
	data, err := json.MarshalIndent(monitors, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

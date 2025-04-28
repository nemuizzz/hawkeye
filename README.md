# Hawkeye

<p align="left">
  <img src="assets/logo.png" alt="Hawkeye Logo" width="100" height="100">
</p>

A simple yet powerful URL monitoring tool that helps you track changes in web content. Use it as a Go package in your code or as a command-line tool. Monitor multiple URLs simultaneously with a single command.

## What Can Hawkeye Do?

- Watch multiple websites for changes at once
- Alert you when content changes on any monitored URL
- Run as a program or use in your Go code
- Customize how and what to monitor for each URL

## Quick Start

### Install

```bash
# Install the command-line tool
go install github.com/nemuizzz/hawkeye/cmd/hawkeye@latest

# Or use as a Go package
go get github.com/nemuizzz/hawkeye
```

### Basic Usage

```bash
# Watch a single website
hawkeye watch https://example.com

# Watch multiple websites
hawkeye watch https://example1.com https://example2.com https://example3.com

# Watch with custom settings
hawkeye watch https://example.com --interval 1m --ignore ".ads,#footer"
```

### Use in Go Code

```go
package main

import (
    "fmt"
    "github.com/nemuizzz/hawkeye"
    "time"
    "context"
)

func main() {
    // List of URLs to monitor
    urls := []string{
        "https://example1.com",
        "https://example2.com",
        "https://example3.com",
    }

    // Create a cancellable context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel() // Ensure we cancel when done

    // Monitor each URL
    for _, url := range urls {
        // Check every 5 minutes
        monitor := hawkeye.NewMonitor(url, time.Minute*5)

        // Start monitoring in a goroutine
        go func(m *hawkeye.Monitor, url string) {
            // Channel to receive change notifications
            changes := m.Start()

            // Watch for changes
            for {
                select {
                case change := <-changes:
                    if change.HasChanged {
                        fmt.Printf("Change detected on %s at %s\n",
                            url,
                            change.Timestamp.Format(time.RFC3339))
                    }
                case <-ctx.Done():
                    // Stop monitoring when context is cancelled
                    fmt.Printf("Stopped monitoring %s\n", url)
                    return
                }
            }
        }(monitor, url)
    }

    // Simple user interaction
    fmt.Println("Monitoring started. Press Enter to stop...")
    fmt.Scanln()
}
```

**Simplified Example (Single URL)**:

```go
package main

import (
    "fmt"
    "github.com/nemuizzz/hawkeye"
    "time"
)

func main() {
    // Monitor a single URL every 5 minutes
    monitor := hawkeye.NewMonitor("https://example.com", time.Minute*5)

    // Start monitoring and display changes
    changes := monitor.Start()
    fmt.Println("Monitoring started. Press Ctrl+C to exit...")

    for change := range changes {
        if change.HasChanged {
            fmt.Printf("Change detected at %s\n",
                change.Timestamp.Format(time.RFC3339))

            // You can access more details about the change
            if change.StatusCode > 0 {
                fmt.Printf("  Status code: %d\n", change.StatusCode)
            }
            if change.Details != "" {
                fmt.Printf("  Details: %s\n", change.Details)
            }
        }
    }
}
```

## Features

### Core Features

- Watch multiple URLs simultaneously
- Set how often to check (1 second to 24 hours)
- Get alerts when content changes on any URL
- Choose output format (text or JSON)

### Advanced Features

- Ignore specific parts of the page
- Add custom headers (like authentication)
- Save results to a file
- Different settings for each URL
- Group URLs for easier management

## Command Line Options

```bash
hawkeye watch [URLs...] [options]

Options:
  -i, --interval     How often to check (default: 5m)
  -f, --format      Output format (text/json)
  -t, --timeout     How long to wait for response
  -h, --header      Add custom headers
  -ig, --ignore     Parts of page to ignore
  -o, --output      Save results to file
  -g, --group       Group name for URLs
  -r, --retries     Number of retry attempts
  -R, --retry-interval Time between retries
      --help        Show help

hawkeye list [options]

Options:
  -f, --format      Output format (text/json)
  -g, --group       Filter by group name
```

## Examples

### Watch Multiple News Sites

```bash
# Check multiple news sites every minute
hawkeye watch \
    https://news1.example.com \
    https://news2.example.com \
    https://news3.example.com \
    --interval 1m \
    --ignore ".ads" \
    --group "news-sites"
```

### Watch Different Types of Sites

```bash
# Watch news and API endpoints
hawkeye watch \
    --group "news" \
    https://news.example.com \
    --interval 1m \
    --ignore ".ads" \
    --group "api" \
    https://api.example.com/data \
    --header "Authorization: Bearer token" \
    --interval 5m
```

### Save Results for Multiple Sites

```bash
# Watch multiple pages and save results
hawkeye watch \
    https://example.com/page1 \
    https://example.com/page2 \
    --output changes.json \
    --format json
```

## Development

### Setup

1. Get the code: `git clone https://github.com/nemuizzz/hawkeye.git`
2. Install: `go mod download`
3. Test: `go test ./...`
4. Build: `go build ./...`

### Project Structure

```
hawkeye/
├── .github/           # GitHub Actions workflows
├── cmd/               # Command-line interface
│   └── hawkeye/       # Hawkeye CLI implementation
│       ├── commands/  # Command implementations
│       └── main.go    # Entry point
├── pkg/               # Public packages
│   ├── http/          # HTTP utilities
│   ├── monitor/       # Core monitoring functionality
│   ├── utils/         # Common utilities
│   └── version/       # Version information
└── internal/          # Private implementation details
```

### Building and Releasing

```bash
# Build from source with version information
make build

# Run tests
make test

# Install locally
make install

# Create a new release (tag must be in vX.Y.Z format)
make release TAG=v0.1.0
```

The project uses GitHub Actions for CI/CD:

- Automatic testing and building on every pull request
- Automatic releases when a new tag is pushed
- Cross-platform binaries built for Linux, macOS, and Windows

### Contributing

1. Fork the repo
2. Make your changes
3. Run tests
4. Send a pull request

## License

MIT License - see [LICENSE](LICENSE) file

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/maximseshuk/snapr/internal/app"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "openapi" {
		if err := app.DumpOpenAPI(os.Stdout, version); err != nil {
			fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
			os.Exit(1)
		}
		return
	}

	configFile := flag.String("config", "", "Path to configuration file")
	configFileAlias := flag.String("c", "", "Path to configuration file (shorthand)")
	showVersion := flag.Bool("version", false, "Show version information")
	showVersionAlias := flag.Bool("v", false, "Show version information (shorthand)")

	flag.Parse()

	if *showVersion || *showVersionAlias {
		fmt.Printf("snapr\n")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("Commit:  %s\n", commit)
		fmt.Printf("Built:   %s\n", date)
		os.Exit(0)
	}

	configPath := *configFile
	if configPath == "" {
		configPath = *configFileAlias
	}

	if err := app.Run(configPath, version); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

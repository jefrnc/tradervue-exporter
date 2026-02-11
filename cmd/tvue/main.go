package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jefrnc/tradervue-utils/internal/api"
	"github.com/jefrnc/tradervue-utils/internal/config"
	"github.com/jefrnc/tradervue-utils/internal/exporter"
	"github.com/jefrnc/tradervue-utils/internal/summary"
)

// version is set at build time via ldflags in the release pipeline.
var version = "dev"

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "export":
		runExport(os.Args[2:])
	case "summary":
		runSummary(os.Args[2:])
	case "version":
		fmt.Printf("tvue v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)

	username := fs.String("username", "", "Tradervue username")
	password := fs.String("password", "", "Tradervue password")
	dataDir := fs.String("data-dir", "", "Data directory (default: ./data)")
	fromDate := fs.String("from", "", "Start date (yyyy-mm-dd)")
	toDate := fs.String("to", "", "End date (yyyy-mm-dd)")
	withExecs := fs.Bool("with-executions", false, "Fetch individual executions per trade (slower)")
	force := fs.Bool("force", false, "Re-export existing dates")

	// Short aliases
	fs.StringVar(username, "u", "", "")
	fs.StringVar(password, "p", "", "")
	fs.StringVar(dataDir, "d", "", "")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tvue export [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	cfg, err := config.Load(*username, *password, *dataDir)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	client := api.NewClient(cfg.Username, cfg.Password, cfg.UserAgent)
	exp := exporter.New(client, cfg.DataDir)

	opts := exporter.Options{
		WithExecutions: *withExecs,
		FromDate:       *fromDate,
		ToDate:         *toDate,
		Force:          *force,
	}

	if err := exp.Run(opts); err != nil {
		log.Fatalf("Export failed: %v", err)
	}
}

func runSummary(args []string) {
	fs := flag.NewFlagSet("summary", flag.ExitOnError)

	dataDir := fs.String("data-dir", "./data", "Data directory")
	fromDate := fs.String("from", "", "Start date filter (yyyy-mm-dd)")
	toDate := fs.String("to", "", "End date filter (yyyy-mm-dd)")
	csvOutput := fs.Bool("csv", false, "Output as CSV")
	outputFile := fs.String("output", "", "Output file (default: stdout)")

	// Short aliases
	fs.StringVar(dataDir, "d", "./data", "")
	fs.StringVar(outputFile, "o", "", "")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: tvue summary [options]\n\nOptions:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	gen := summary.NewGenerator(*dataDir)

	summaries, err := gen.Generate(*fromDate, *toDate)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if len(summaries) == 0 {
		log.Println("No exported data found. Run 'tvue export' first.")
		return
	}

	// Determine output writer
	var w *os.File
	if *outputFile != "" {
		f, err := os.Create(*outputFile)
		if err != nil {
			log.Fatalf("Error creating output file: %v", err)
		}
		defer f.Close()
		w = f
	} else {
		w = os.Stdout
	}

	if *csvOutput {
		if err := gen.ExportCSV(w, summaries); err != nil {
			log.Fatalf("Error writing CSV: %v", err)
		}
	} else {
		gen.PrintTable(w, summaries)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `tvue v%s - Tradervue Trade Exporter & Analyzer

Export all your trades from Tradervue incrementally and generate daily summaries.

Usage:
  tvue <command> [options]

Commands:
  export    Export trades from Tradervue API
  summary   Show daily trade summaries from exported data
  version   Print version
  help      Show this help

Examples:
  tvue export -u myuser -p mypass          # First run (full export)
  tvue export                              # Incremental (uses .env)
  tvue export --from 2025-01-01 --force    # Re-export range
  tvue summary                             # Show all summaries
  tvue summary --from 2025-01-01 --csv     # CSV output

Configuration:
  Credentials via flags (--username, --password) or .env file:
    TRADERVUE_USERNAME=your_username
    TRADERVUE_PASSWORD=your_password

`, version)
}

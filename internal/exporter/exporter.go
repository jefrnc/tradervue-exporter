package exporter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jefrnc/tradervue-utils/internal/api"
	"github.com/jefrnc/tradervue-utils/internal/models"
)

const (
	stateFile  = "state.json"
	tradesDir  = "trades"
	tvDateFmt  = "01/02/2006" // Tradervue API date format (mm/dd/yyyy)
	fileDateFmt = "2006-01-02" // File naming format (yyyy-mm-dd)
)

// Options controls the export behavior.
type Options struct {
	WithExecutions bool
	FromDate       string // yyyy-mm-dd override
	ToDate         string // yyyy-mm-dd override
	Force          bool
}

// Exporter orchestrates the trade export from Tradervue.
type Exporter struct {
	client  *api.Client
	dataDir string
}

// New creates a new Exporter.
func New(client *api.Client, dataDir string) *Exporter {
	return &Exporter{client: client, dataDir: dataDir}
}

// Run executes the export process.
func (e *Exporter) Run(opts Options) error {
	// Ensure data directories exist
	tradesPath := filepath.Join(e.dataDir, tradesDir)
	if err := os.MkdirAll(tradesPath, 0755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	state, _ := e.loadState()

	var startDate, endDate time.Time

	// Determine date range
	if opts.FromDate != "" {
		var err error
		startDate, err = time.Parse(fileDateFmt, opts.FromDate)
		if err != nil {
			return fmt.Errorf("invalid --from date %q (use yyyy-mm-dd): %w", opts.FromDate, err)
		}
	} else if state != nil && state.LastExportDate != "" && !opts.Force {
		last, err := time.Parse(fileDateFmt, state.LastExportDate)
		if err != nil {
			return fmt.Errorf("corrupt state file: %w", err)
		}
		startDate = last.AddDate(0, 0, 1) // day after last export
	} else {
		// First run: discover first trade date
		log.Println("First run: discovering first trade date...")
		first, err := e.discoverFirstTradeDate()
		if err != nil {
			return err
		}
		startDate = first
		log.Printf("First trade found on: %s", startDate.Format(fileDateFmt))
	}

	if opts.ToDate != "" {
		var err error
		endDate, err = time.Parse(fileDateFmt, opts.ToDate)
		if err != nil {
			return fmt.Errorf("invalid --to date %q (use yyyy-mm-dd): %w", opts.ToDate, err)
		}
	} else {
		endDate = time.Now()
	}

	if startDate.After(endDate) {
		log.Println("Already up to date. No new trades to export.")
		return nil
	}

	log.Printf("Exporting trades from %s to %s...", startDate.Format(fileDateFmt), endDate.Format(fileDateFmt))

	// Fetch all trades in the date range
	allTrades, err := e.fetchAllTrades(startDate, endDate)
	if err != nil {
		return err
	}

	if len(allTrades) == 0 {
		log.Println("No trades found in the date range.")
		return nil
	}

	// Group trades by date
	byDate := e.groupTradesByDate(allTrades)
	dates := sortedKeys(byDate)

	totalTrades := 0
	for _, date := range dates {
		trades := byDate[date]
		totalTrades += len(trades)

		dayExport := &models.DayExport{
			Date:       date,
			Trades:     trades,
			ExportedAt: time.Now(),
		}

		// Optionally fetch executions
		if opts.WithExecutions {
			execs, err := e.fetchExecutionsForTrades(trades)
			if err != nil {
				log.Printf("Warning: failed to fetch executions for %s: %v", date, err)
			} else {
				dayExport.Executions = execs
			}
		}

		if err := e.saveDayExport(dayExport); err != nil {
			return fmt.Errorf("saving %s: %w", date, err)
		}

		// Build symbol summary for log
		symbols := summarizeSymbols(trades)
		log.Printf("  %s: %d trades [%s]", date, len(trades), symbols)
	}

	// Update state
	if state == nil {
		state = &models.ExportState{}
	}
	if state.FirstTradeDate == "" || dates[0] < state.FirstTradeDate {
		state.FirstTradeDate = dates[0]
	}
	lastDate := dates[len(dates)-1]
	if lastDate > state.LastExportDate {
		state.LastExportDate = lastDate
	}
	state.TotalTrades += totalTrades
	state.TotalDays += len(dates)
	state.LastRunAt = time.Now()

	if err := e.saveState(state); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	log.Printf("Export complete: %d days, %d trades", len(dates), totalTrades)
	return nil
}

// discoverFirstTradeDate finds the oldest trade in the account.
func (e *Exporter) discoverFirstTradeDate() (time.Time, error) {
	// Fetch trades without date filter to get the total count.
	// Tradervue returns newest first, so we paginate to the last page.
	page := 1
	var oldest models.Trade
	var found bool

	for {
		trades, err := e.client.ListTrades("01/01/2010", "", page)
		if err != nil {
			return time.Time{}, fmt.Errorf("discovering first trade: %w", err)
		}
		if len(trades) == 0 {
			break
		}
		// The last item on the last page is the oldest trade
		oldest = trades[len(trades)-1]
		found = true

		if len(trades) < 100 {
			// This was the last page
			break
		}
		page++
		log.Printf("  Scanning page %d...", page)
	}

	if !found {
		return time.Time{}, fmt.Errorf("no trades found in your Tradervue account")
	}

	// Parse the start_datetime to extract the date
	return parseTradeDate(oldest.StartDatetime)
}

// fetchAllTrades retrieves all trades in a date range with pagination.
func (e *Exporter) fetchAllTrades(start, end time.Time) ([]models.Trade, error) {
	startStr := start.Format(tvDateFmt)
	endStr := end.Format(tvDateFmt)

	var all []models.Trade
	page := 1

	for {
		trades, err := e.client.ListTrades(startStr, endStr, page)
		if err != nil {
			return nil, fmt.Errorf("fetching trades page %d: %w", page, err)
		}
		if len(trades) == 0 {
			break
		}
		all = append(all, trades...)

		if len(trades) < 100 {
			break
		}
		page++
	}

	return all, nil
}

// groupTradesByDate groups trades by the date portion of their StartDatetime.
func (e *Exporter) groupTradesByDate(trades []models.Trade) map[string][]models.Trade {
	byDate := make(map[string][]models.Trade)

	for _, t := range trades {
		date, err := parseTradeDate(t.StartDatetime)
		if err != nil {
			log.Printf("Warning: skipping trade %d with unparseable date %q", t.ID, t.StartDatetime)
			continue
		}
		key := date.Format(fileDateFmt)
		byDate[key] = append(byDate[key], t)
	}

	return byDate
}

// fetchExecutionsForTrades fetches executions for each trade.
func (e *Exporter) fetchExecutionsForTrades(trades []models.Trade) (map[int][]models.Execution, error) {
	result := make(map[int][]models.Execution)

	for _, t := range trades {
		execs, err := e.client.GetExecutions(t.ID)
		if err != nil {
			return nil, fmt.Errorf("fetching executions for trade %d: %w", t.ID, err)
		}
		if len(execs) > 0 {
			result[t.ID] = execs
		}
	}

	return result, nil
}

// saveDayExport writes a day's export to a JSON file.
func (e *Exporter) saveDayExport(day *models.DayExport) error {
	path := filepath.Join(e.dataDir, tradesDir, day.Date+".json")

	data, err := json.MarshalIndent(day, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// loadState reads the export state file.
func (e *Exporter) loadState() (*models.ExportState, error) {
	path := filepath.Join(e.dataDir, stateFile)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state models.ExportState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// saveState writes the export state file.
func (e *Exporter) saveState(state *models.ExportState) error {
	path := filepath.Join(e.dataDir, stateFile)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// parseTradeDate extracts a time.Time from a Tradervue datetime string.
// Tradervue returns ISO 8601 format like "2025-01-15T09:30:00-05:00".
func parseTradeDate(datetime string) (time.Time, error) {
	// Try full ISO 8601
	t, err := time.Parse(time.RFC3339, datetime)
	if err == nil {
		// Load Eastern timezone for consistent date grouping
		loc, locErr := time.LoadLocation("America/New_York")
		if locErr == nil {
			t = t.In(loc)
		}
		return t, nil
	}

	// Try date-only format
	t, err = time.Parse(fileDateFmt, datetime)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("cannot parse date %q", datetime)
}

// summarizeSymbols creates a compact string like "SNGX(L) MULN(S)"
func summarizeSymbols(trades []models.Trade) string {
	seen := make(map[string]string) // symbol -> side
	var order []string

	for _, t := range trades {
		if _, ok := seen[t.Symbol]; !ok {
			order = append(order, t.Symbol)
		}
		existing := seen[t.Symbol]
		if existing == "" {
			seen[t.Symbol] = t.Side
		} else if existing != t.Side {
			seen[t.Symbol] = "L/S"
		}
	}

	var parts []string
	for _, sym := range order {
		parts = append(parts, fmt.Sprintf("%s(%s)", sym, seen[sym]))
	}
	return strings.Join(parts, " ")
}

// sortedKeys returns map keys sorted alphabetically.
func sortedKeys(m map[string][]models.Trade) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

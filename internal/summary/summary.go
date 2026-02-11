package summary

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/jefrnc/tradervue-utils/internal/models"
)

// Generator reads exported day files and produces summaries.
type Generator struct {
	dataDir string
}

// NewGenerator creates a new summary generator.
func NewGenerator(dataDir string) *Generator {
	return &Generator{dataDir: dataDir}
}

// Generate produces daily summaries for the given date range.
// Dates should be in yyyy-mm-dd format. Empty strings mean no filter.
func (g *Generator) Generate(fromDate, toDate string) ([]models.DailySummary, error) {
	tradesPath := filepath.Join(g.dataDir, "trades")

	entries, err := os.ReadDir(tradesPath)
	if err != nil {
		return nil, fmt.Errorf("reading trades directory: %w", err)
	}

	var summaries []models.DailySummary

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		date := strings.TrimSuffix(entry.Name(), ".json")

		// Apply date filters
		if fromDate != "" && date < fromDate {
			continue
		}
		if toDate != "" && date > toDate {
			continue
		}

		dayExport, err := g.loadDayExport(filepath.Join(tradesPath, entry.Name()))
		if err != nil {
			continue
		}

		summary := buildDailySummary(date, dayExport.Trades)
		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Date < summaries[j].Date
	})

	return summaries, nil
}

// PrintTable prints summaries as a formatted ASCII table.
func (g *Generator) PrintTable(w io.Writer, summaries []models.DailySummary) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	fmt.Fprintf(tw, "DATE\tTRADES\tGROSS P&L\tNET P&L\tWIN%%\tVOLUME\tSYMBOLS\n")
	fmt.Fprintf(tw, "────\t──────\t─────────\t───────\t────\t──────\t───────\n")

	var totGross, totNet, totComm, totFees float64
	var totTrades, totVol, totWin, totLoss int

	for _, s := range summaries {
		symbols := formatSymbols(s.Symbols)
		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%.0f%%\t%d\t%s\n",
			s.Date,
			s.TradeCount,
			formatPL(s.GrossPL),
			formatPL(s.NetPL),
			s.WinRate,
			s.TotalVolume,
			symbols,
		)

		totGross += s.GrossPL
		totNet += s.NetPL
		totComm += s.Commission
		totFees += s.Fees
		totTrades += s.TradeCount
		totVol += s.TotalVolume
		totWin += s.Winners
		totLoss += s.Losers
	}

	fmt.Fprintf(tw, "────\t──────\t─────────\t───────\t────\t──────\t───────\n")

	winRate := 0.0
	if totWin+totLoss > 0 {
		winRate = float64(totWin) / float64(totWin+totLoss) * 100
	}

	fmt.Fprintf(tw, "TOTAL\t%d\t%s\t%s\t%.0f%%\t%d\t%d days\n",
		totTrades,
		formatPL(totGross),
		formatPL(totNet),
		winRate,
		totVol,
		len(summaries),
	)

	tw.Flush()
}

// ExportCSV writes summaries as CSV.
func (g *Generator) ExportCSV(w io.Writer, summaries []models.DailySummary) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header
	if err := cw.Write([]string{
		"date", "trades", "gross_pl", "net_pl", "commission", "fees",
		"win_rate", "winners", "losers", "volume", "symbols",
	}); err != nil {
		return err
	}

	for _, s := range summaries {
		symbols := formatSymbolsCSV(s.Symbols)
		if err := cw.Write([]string{
			s.Date,
			fmt.Sprintf("%d", s.TradeCount),
			fmt.Sprintf("%.2f", s.GrossPL),
			fmt.Sprintf("%.2f", s.NetPL),
			fmt.Sprintf("%.2f", s.Commission),
			fmt.Sprintf("%.2f", s.Fees),
			fmt.Sprintf("%.1f", s.WinRate),
			fmt.Sprintf("%d", s.Winners),
			fmt.Sprintf("%d", s.Losers),
			fmt.Sprintf("%d", s.TotalVolume),
			symbols,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) loadDayExport(path string) (*models.DayExport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var day models.DayExport
	if err := json.Unmarshal(data, &day); err != nil {
		return nil, err
	}

	return &day, nil
}

func buildDailySummary(date string, trades []models.Trade) models.DailySummary {
	s := models.DailySummary{
		Date:       date,
		TradeCount: len(trades),
	}

	// Track symbols
	type symAgg struct {
		sides   map[string]bool
		grossPL float64
		volume  int
		count   int
	}
	syms := make(map[string]*symAgg)
	var symOrder []string

	for _, t := range trades {
		s.GrossPL += t.GrossPL
		s.Commission += t.Commission
		s.Fees += t.Fees
		s.TotalVolume += t.Volume

		if t.GrossPL > 0 {
			s.Winners++
		} else {
			s.Losers++
		}

		agg, ok := syms[t.Symbol]
		if !ok {
			agg = &symAgg{sides: make(map[string]bool)}
			syms[t.Symbol] = agg
			symOrder = append(symOrder, t.Symbol)
		}
		agg.sides[t.Side] = true
		agg.grossPL += t.GrossPL
		agg.volume += t.Volume
		agg.count++
	}

	s.NetPL = s.GrossPL - s.Commission - s.Fees

	if s.Winners+s.Losers > 0 {
		s.WinRate = float64(s.Winners) / float64(s.Winners+s.Losers) * 100
	}

	for _, sym := range symOrder {
		agg := syms[sym]
		side := sideFromMap(agg.sides)
		s.Symbols = append(s.Symbols, models.SymbolSummary{
			Symbol:  sym,
			Side:    side,
			GrossPL: agg.grossPL,
			Volume:  agg.volume,
			Count:   agg.count,
		})
	}

	return s
}

func sideFromMap(sides map[string]bool) string {
	hasLong := sides["L"]
	hasShort := sides["S"]
	if hasLong && hasShort {
		return "L/S"
	}
	if hasShort {
		return "S"
	}
	return "L"
}

func formatPL(v float64) string {
	if v >= 0 {
		return fmt.Sprintf("+$%.2f", v)
	}
	return fmt.Sprintf("-$%.2f", -v)
}

func formatSymbols(syms []models.SymbolSummary) string {
	var parts []string
	for _, s := range syms {
		parts = append(parts, fmt.Sprintf("%s(%s)%s", s.Symbol, s.Side, formatPL(s.GrossPL)))
	}
	return strings.Join(parts, " ")
}

func formatSymbolsCSV(syms []models.SymbolSummary) string {
	var parts []string
	for _, s := range syms {
		parts = append(parts, fmt.Sprintf("%s(%s)", s.Symbol, s.Side))
	}
	return strings.Join(parts, " ")
}

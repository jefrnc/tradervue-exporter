package models

import "time"

// Trade represents a single Tradervue trade.
type Trade struct {
	ID             int      `json:"id"`
	Symbol         string   `json:"symbol"`
	Volume         int      `json:"volume"`
	Open           bool     `json:"open"`
	Side           string   `json:"side"` // "L" = Long, "S" = Short
	EntryPrice     float64  `json:"entry_price"`
	ExitPrice      *float64 `json:"exit_price,omitempty"`
	GrossPL        float64  `json:"gross_pl"`
	NativePL       *float64 `json:"native_pl,omitempty"`
	NativeCurrency *string  `json:"native_currency,omitempty"`
	Commission     float64  `json:"commission"`
	Fees           float64  `json:"fees"`
	StartDatetime  string   `json:"start_datetime"`
	EndDatetime    *string  `json:"end_datetime,omitempty"`
	Duration       string   `json:"duration"` // "I" = Intraday, "M" = Multi-day
	Notes          string   `json:"notes"`
	NotesExcerpt   string   `json:"notes_excerpt"`
	Tags           []string `json:"tags"`
	Shared         bool     `json:"shared"`
	InitialRisk    *float64 `json:"initial_risk,omitempty"`
	ExecCount      int      `json:"exec_count"`
	CommentCount   int      `json:"comment_count"`

	// MFE/MAE fields
	PositionMFE         *float64 `json:"position_mfe,omitempty"`
	PositionMFEDatetime *string  `json:"position_mfe_datetime,omitempty"`
	PositionMAE         *float64 `json:"position_mae,omitempty"`
	PositionMAEDatetime *string  `json:"position_mae_datetime,omitempty"`
	PriceMFE            *float64 `json:"price_mfe,omitempty"`
	PriceMFEDatetime    *string  `json:"price_mfe_datetime,omitempty"`
	PriceMAE            *float64 `json:"price_mae,omitempty"`
	PriceMAEDatetime    *string  `json:"price_mae_datetime,omitempty"`
	BestExitPL          *float64 `json:"best_exit_pl,omitempty"`
	BestExitPLDatetime  *string  `json:"best_exit_pl_datetime,omitempty"`
}

// Execution represents a single fill/execution within a trade.
type Execution struct {
	ID         int     `json:"id"`
	Datetime   string  `json:"datetime"`
	Symbol     string  `json:"symbol"`
	Quantity   int     `json:"quantity"` // positive = buy, negative = sell
	Price      float64 `json:"price"`
	Commission float64 `json:"commission"`
	TransFee   float64 `json:"trans_fee"`
	ECNFee     float64 `json:"ecn_fee"`
}

// JournalEntry represents a Tradervue daily journal entry.
type JournalEntry struct {
	ID           int     `json:"id"`
	Date         string  `json:"date"`
	Notes        string  `json:"notes"`
	CommentCount int     `json:"comment_count"`
	TradeCount   int     `json:"trade_count"`
	TotalVolume  int     `json:"total_volume"`
	GrossPL      float64 `json:"gross_pl"`
	CommFees     float64 `json:"commfees"`
	TradeIDs     []int   `json:"trade_ids"`
}

// DayExport holds all exported data for a single trading day.
type DayExport struct {
	Date       string                `json:"date"`
	Trades     []Trade               `json:"trades"`
	Executions map[int][]Execution   `json:"executions,omitempty"`
	Journal    *JournalEntry         `json:"journal,omitempty"`
	ExportedAt time.Time             `json:"exported_at"`
}

// ExportState tracks incremental export progress.
type ExportState struct {
	LastExportDate string    `json:"last_export_date"`
	FirstTradeDate string    `json:"first_trade_date"`
	TotalTrades    int       `json:"total_trades"`
	TotalDays      int       `json:"total_days"`
	LastRunAt      time.Time `json:"last_run_at"`
}

// DailySummary is a computed summary for display.
type DailySummary struct {
	Date        string          `json:"date"`
	TradeCount  int             `json:"trade_count"`
	Symbols     []SymbolSummary `json:"symbols"`
	GrossPL     float64         `json:"gross_pl"`
	NetPL       float64         `json:"net_pl"`
	Commission  float64         `json:"commission"`
	Fees        float64         `json:"fees"`
	TotalVolume int             `json:"total_volume"`
	Winners     int             `json:"winners"`
	Losers      int             `json:"losers"`
	WinRate     float64         `json:"win_rate"`
}

// SymbolSummary groups trades by symbol within a day.
type SymbolSummary struct {
	Symbol  string  `json:"symbol"`
	Side    string  `json:"side"` // L, S, or "L/S" if both
	GrossPL float64 `json:"gross_pl"`
	Volume  int     `json:"volume"`
	Count   int     `json:"count"`
}

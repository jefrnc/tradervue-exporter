package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/jefrnc/tradervue-utils/internal/models"
)

const (
	baseURL       = "https://app.tradervue.com/api/v1"
	maxPerPage    = 100
	requestDelay  = 200 * time.Millisecond
	maxRetries    = 3
)

// Client is the Tradervue API client.
type Client struct {
	username   string
	password   string
	userAgent  string
	httpClient *http.Client
	lastReq    time.Time
}

// NewClient creates a new Tradervue API client.
func NewClient(username, password, userAgent string) *Client {
	return &Client{
		username:  username,
		password:  password,
		userAgent: userAgent,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// tradesResponse wraps the API response for /trades.
type tradesResponse struct {
	Trades []models.Trade `json:"trades"`
}

// executionsResponse wraps the API response for /trades/{id}/executions.
type executionsResponse struct {
	Executions []models.Execution `json:"executions"`
}

// journalResponse wraps the API response for /journal.
type journalResponse struct {
	JournalEntries []models.JournalEntry `json:"journal_entries"`
}

// ListTrades fetches a page of trades with optional date filters.
// Dates should be in mm/dd/yyyy format as required by Tradervue.
func (c *Client) ListTrades(startDate, endDate string, page int) ([]models.Trade, error) {
	url := fmt.Sprintf("%s/trades?count=%d&page=%d", baseURL, maxPerPage, page)
	if startDate != "" {
		url += "&startdate=" + startDate
	}
	if endDate != "" {
		url += "&enddate=" + endDate
	}

	var resp tradesResponse
	if err := c.doGet(url, &resp); err != nil {
		return nil, err
	}
	return resp.Trades, nil
}

// GetExecutions fetches all executions for a given trade ID.
func (c *Client) GetExecutions(tradeID int) ([]models.Execution, error) {
	url := fmt.Sprintf("%s/trades/%d/executions", baseURL, tradeID)

	var resp executionsResponse
	if err := c.doGet(url, &resp); err != nil {
		return nil, err
	}
	return resp.Executions, nil
}

// ListJournal fetches a page of journal entries with optional date filters.
func (c *Client) ListJournal(startDate, endDate string, page int) ([]models.JournalEntry, error) {
	url := fmt.Sprintf("%s/journal?count=%d&page=%d", baseURL, maxPerPage, page)
	if startDate != "" {
		url += "&startdate=" + startDate
	}
	if endDate != "" {
		url += "&enddate=" + endDate
	}

	var resp journalResponse
	if err := c.doGet(url, &resp); err != nil {
		return nil, err
	}
	return resp.JournalEntries, nil
}

// doGet performs an authenticated GET request with retry and rate limiting.
func (c *Client) doGet(url string, result interface{}) error {
	c.rateLimit()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(wait)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("reading response: %w", readErr)
			continue
		}

		switch {
		case resp.StatusCode == 401:
			return fmt.Errorf("authentication failed (HTTP 401): check your username and password")
		case resp.StatusCode == 400:
			return fmt.Errorf("bad request (HTTP 400): %s", string(body))
		case resp.StatusCode >= 500:
			lastErr = fmt.Errorf("server error (HTTP %d): %s", resp.StatusCode, string(body))
			continue
		case resp.StatusCode != 200:
			return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
		}

		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
		return nil
	}

	return fmt.Errorf("request failed after %d attempts: %w", maxRetries, lastErr)
}

// rateLimit enforces a minimum delay between API requests.
func (c *Client) rateLimit() {
	if !c.lastReq.IsZero() {
		elapsed := time.Since(c.lastReq)
		if elapsed < requestDelay {
			time.Sleep(requestDelay - elapsed)
		}
	}
	c.lastReq = time.Now()
}

package plot

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/raykavin/backnrun/pkg/exchange"
)

// handleHealth handles health check requests
func (c *Chart) handleHealth(w http.ResponseWriter, _ *http.Request) {
	// Consider unhealthy if no updates in more than an hour
	if time.Since(c.lastUpdate) > time.Hour+10*time.Minute {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, err := w.Write([]byte(c.lastUpdate.String()))
		if err != nil {
			c.log.Error("Failed to write health status: ", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleIndex handles the main page request
func (c *Chart) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Get all available pairs
	c.Lock()
	var pairs = make([]string, 0, len(c.candles))
	for pair := range c.candles {
		pairs = append(pairs, pair)
	}
	c.Unlock()

	sort.Strings(pairs)

	// Get requested pair or redirect to first available pair
	pair := r.URL.Query().Get("pair")
	if pair == "" && len(pairs) > 0 {
		http.Redirect(w, r, fmt.Sprintf("/?pair=%s", pairs[0]), http.StatusFound)
		return
	}

	// Render the template
	w.Header().Set("Content-Type", "text/html")
	err := c.indexHTML.Execute(w, map[string]interface{}{
		"pair":  pair,
		"pairs": pairs,
	})
	if err != nil {
		c.log.Error("Template execution failed: ", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleData handles chart data requests
func (c *Chart) handleData(w http.ResponseWriter, r *http.Request) {
	pair := r.URL.Query().Get("pair")
	if pair == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get maximum drawdown information if available
	var maxDrawdown *drawdown
	if c.paperWallet != nil {
		value, start, end := c.paperWallet.MaxDrawdown()
		maxDrawdown = &drawdown{
			Start: start,
			End:   end,
			Value: fmt.Sprintf("%.1f", value*100),
		}
	}

	// Split pair into asset and quote
	asset, quote := exchange.SplitAssetQuote(pair)

	// Get asset and equity values
	assetValues, equityValues := c.equityValuesByPair(pair)

	c.Lock()
	// Encode response as JSON
	response := map[string]interface{}{
		"candles":       c.candlesByPair(pair),
		"indicators":    c.indicatorsByPair(pair),
		"shapes":        c.shapesByPair(pair),
		"asset_values":  assetValues,
		"equity_values": equityValues,
		"quote":         quote,
		"asset":         asset,
		"max_drawdown":  maxDrawdown,
	}
	c.Unlock()

	if err := json.NewEncoder(w).Encode(response); err != nil {
		c.log.Error("JSON encoding failed: ", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleTradingHistoryData handles CSV export of trading history
func (c *Chart) handleTradingHistoryData(w http.ResponseWriter, r *http.Request) {
	pair := r.URL.Query().Get("pair")
	if pair == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Set headers for CSV download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=history_"+pair+".csv")
	w.Header().Set("Transfer-Encoding", "chunked")

	c.Lock()
	// Get order data
	orders := c.orderStringByPair(pair)
	c.Unlock()

	// Create CSV in memory
	buffer := bytes.NewBuffer(nil)
	csvWriter := csv.NewWriter(buffer)

	// Write header
	if err := csvWriter.Write([]string{
		"created_at", "status", "side", "id", "type",
		"quantity", "price", "total", "profit",
	}); err != nil {
		c.log.Error("Failed writing CSV header: ", err)
		http.Error(w, "Failed to generate CSV", http.StatusInternalServerError)
		return
	}

	// Write data rows
	if err := csvWriter.WriteAll(orders); err != nil {
		c.log.Error("Failed writing CSV data: ", err)
		http.Error(w, "Failed to generate CSV", http.StatusInternalServerError)
		return
	}
	csvWriter.Flush()

	// Send the CSV
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(buffer.Bytes()); err != nil {
		c.log.Error("Failed writing CSV response: ", err)
	}
}

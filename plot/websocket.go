package plot

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// WebSocketManager handles WebSocket connections
type WebSocketManager struct {
	sync.RWMutex
	clients       map[*websocket.Conn]string // map of connection to pair
	upgrader      websocket.Upgrader
	broadcastChan chan WebSocketMessage
	log           core.Logger
	chart         *Chart // Reference to the chart instance
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(log core.Logger, chart *Chart) *WebSocketManager {
	manager := &WebSocketManager{
		clients: make(map[*websocket.Conn]string),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		broadcastChan: make(chan WebSocketMessage, 100),
		log:           log,
		chart:         chart,
	}

	// Start broadcast handler
	go manager.handleBroadcasts()

	return manager
}

// handleBroadcasts processes messages from the broadcast channel
func (m *WebSocketManager) handleBroadcasts() {
	for msg := range m.broadcastChan {
		m.RLock()
		for conn, pair := range m.clients {
			// If message has a specific pair, only send to clients subscribed to that pair
			if msgPair, ok := msg.Payload.(map[string]any)["pair"].(string); ok && msgPair != "" {
				if pair != msgPair {
					continue
				}
			}

			err := conn.WriteJSON(msg)
			if err != nil {
				m.log.Error("Error sending WebSocket message: ", err)
				conn.Close()
				// We can't remove from the map while iterating with the read lock
				// The connection will be removed when the client handler detects the closed connection
			}
		}
		m.RUnlock()
	}
}

// HandleWebSocket handles WebSocket connections
func (m *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract pair from query parameters
	pair := r.URL.Query().Get("pair")
	if pair == "" {
		http.Error(w, "Missing pair parameter", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.log.Error("Failed to upgrade connection to WebSocket: ", err)
		return
	}

	// Register client
	m.Lock()
	m.clients[conn] = pair
	m.log.Info("New WebSocket client connected for pair: ", pair)
	clientCount := len(m.clients)
	m.Unlock()

	m.log.Info("Total WebSocket clients: ", clientCount)

	// Send initial data
	go m.sendInitialData(conn, pair)

	// Handle client messages
	go m.handleClient(conn)
}

// handleClient processes messages from a client
func (m *WebSocketManager) handleClient(conn *websocket.Conn) {
	defer func() {
		// Remove client on disconnect
		m.Lock()
		delete(m.clients, conn)
		m.log.Info("WebSocket client disconnected, remaining: ", len(m.clients))
		m.Unlock()
		conn.Close()
	}()

	// Keep connection alive with ping/pong
	conn.SetPingHandler(func(appData string) error {
		return conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(10*time.Second))
	})

	// Read messages (we don't expect any, but need to handle disconnects)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				m.log.Error("WebSocket read error: ", err)
			}
			break
		}
	}
}

// sendInitialData sends initial chart data to a new client
func (m *WebSocketManager) sendInitialData(conn *websocket.Conn, pair string) {
	// Get the chart instance
	m.chart.Lock()
	// Split pair into asset and quote
	asset, quote := exchange.SplitAssetQuote(pair)

	// Get asset and equity values
	assetValues, equityValues := m.chart.equityValuesByPair(pair)

	m.Lock()
	// Prepare response
	response := map[string]any{
		"candles":       m.chart.candlesByPair(pair),
		"indicators":    m.chart.indicatorsByPair(pair),
		"shapes":        m.chart.shapesByPair(pair),
		"asset_values":  assetValues,
		"equity_values": equityValues,
		"quote":         quote,
		"asset":         asset,
		"pair":          pair,
	}
	m.Unlock()

	// Add max drawdown if available
	if m.chart.paperWallet != nil {
		value, start, end := m.chart.paperWallet.MaxDrawdown()
		response["max_drawdown"] = &drawdown{
			Start: start,
			End:   end,
			Value: fmt.Sprintf("%.1f", value*100),
		}
	}
	m.chart.Unlock()

	// Send initial data
	msg := WebSocketMessage{
		Type:    "initialData",
		Payload: response,
	}

	err := conn.WriteJSON(msg)
	if err != nil {
		m.log.Error("Error sending initial data: ", err)
	}
}

// BroadcastCandle broadcasts a new candle to all clients
func (m *WebSocketManager) BroadcastCandle(candle core.Candle, pair string) {
	// Create a candle object with the complete flag
	candleData := map[string]any{
		"time":     candle.Time,
		"open":     candle.Open,
		"high":     candle.High,
		"low":      candle.Low,
		"close":    candle.Close,
		"volume":   candle.Volume,
		"complete": candle.Complete,
	}

	m.broadcastChan <- WebSocketMessage{
		Type: "newCandle",
		Payload: map[string]any{
			"candle": candleData,
			"pair":   pair,
		},
	}
}

// BroadcastOrder broadcasts a new order to all clients
func (m *WebSocketManager) BroadcastOrder(order core.Order) {
	m.broadcastChan <- WebSocketMessage{
		Type: "newOrder",
		Payload: map[string]any{
			"order": order,
			"pair":  order.Pair,
		},
	}
}

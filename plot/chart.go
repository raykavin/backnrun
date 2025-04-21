package plot

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/StudioSol/set"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange"
)

// Static assets embedded in the binary
var (
	//go:embed assets
	staticFiles embed.FS
)

// Chart handles the visualization of trading data
type Chart struct {
	sync.Mutex
	port               int
	debug              bool
	candles            map[string][]Candle
	dataframe          map[string]*core.Dataframe
	ordersIDsByPair    map[string]*set.LinkedHashSetINT64
	orderByID          map[int64]core.Order
	indicators         []Indicator
	paperWallet        *exchange.PaperWallet
	scriptContent      string
	indexHTML          *template.Template
	strategy           core.Strategy
	lastUpdate         time.Time
	log                core.Logger
	wsManager          *WebSocketManager
	simulationInterval time.Duration
}

// Option defines a function type for configuring a Chart instance
type Option func(*Chart)

// WithPort sets the HTTP server port
func WithPort(port int) Option {
	return func(chart *Chart) {
		chart.port = port
	}
}

// WithStrategyIndicators sets the strategy for indicators
func WithStrategyIndicators(strategy core.Strategy) Option {
	return func(chart *Chart) {
		chart.strategy = strategy
	}
}

// WithPaperWallet sets the paper wallet for the chart
func WithPaperWallet(paperWallet *exchange.PaperWallet) Option {
	return func(chart *Chart) {
		chart.paperWallet = paperWallet
	}
}

// WithDebug enables debug mode (disables minification)
func WithDebug() Option {
	return func(chart *Chart) {
		chart.debug = true
	}
}

// WithCustomIndicators adds custom indicators to the chart
func WithCustomIndicators(indicators ...Indicator) Option {
	return func(chart *Chart) {
		chart.indicators = indicators
	}
}

// WithSimulation enables real-time candle simulation for testing
func WithSimulation(interval time.Duration) Option {
	return func(chart *Chart) {
		chart.simulationInterval = interval
	}
}

// NewChart creates a new chart instance with the provided options
func NewChart(log core.Logger, options ...Option) (*Chart, error) {
	chart := &Chart{
		port:            8080,
		log:             log,
		candles:         make(map[string][]Candle),
		dataframe:       make(map[string]*core.Dataframe),
		ordersIDsByPair: make(map[string]*set.LinkedHashSetINT64),
		orderByID:       make(map[int64]core.Order),
	}

	// Apply all options
	for _, option := range options {
		option(chart)
	}

	// Parse chart HTML template
	var err error
	chart.indexHTML, err = template.ParseFS(staticFiles, "assets/index.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse chart template: %w", err)
	}

	// Read and transpile chart JavaScript
	chartJS, err := staticFiles.ReadFile("assets/js/main.js")
	if err != nil {
		return nil, fmt.Errorf("failed to read main.js: %w", err)
	}

	transpileChartJS := api.Transform(string(chartJS), api.TransformOptions{
		Loader:            api.LoaderJS,
		Target:            api.ES2015,
		MinifySyntax:      !chart.debug,
		MinifyIdentifiers: !chart.debug,
		MinifyWhitespace:  !chart.debug,
	})

	if len(transpileChartJS.Errors) > 0 {
		return nil, fmt.Errorf("chart script failed with: %v", transpileChartJS.Errors)
	}

	chart.scriptContent = string(transpileChartJS.Code)

	// Create WebSocket manager
	chart.wsManager = NewWebSocketManager(log, chart)

	return chart, nil
}

// GetPort returns the configured port
func (c *Chart) GetPort() int {
	return c.port
}

// GetWSManager returns the WebSocket manager
func (c *Chart) GetWSManager() *WebSocketManager {
	return c.wsManager
}

// GetSimulationInterval returns the simulation interval
func (c *Chart) GetSimulationInterval() time.Duration {
	return c.simulationInterval
}

// GetFirstAvailablePair returns the first available pair or empty string
func (c *Chart) GetFirstAvailablePair() string {
	c.Lock()
	defer c.Unlock()

	for p := range c.candles {
		return p
	}

	return ""
}

// RegisterHandlers registers all necessary handlers on the HTTP server
func (c *Chart) RegisterHandlers(server HTTPServer) {
	// Register static file handler
	server.RegisterFileServer("/assets/", http.FS(staticFiles))

	// Register API handlers
	server.RegisterHandler("/health", c.handleHealth)
	server.RegisterHandler("/history", c.handleTradingHistoryData)
	server.RegisterHandler("/ws", c.wsManager.HandleWebSocket)
	server.RegisterHandler("/", c.handleIndex)
}

// StartSimulation starts candle simulation if configured
func (c *Chart) StartSimulation() {
	if c.simulationInterval <= 0 {
		return
	}

	// Get the first available pair or use a default
	var pair string = c.GetFirstAvailablePair()

	// If no pairs are available, use a default
	if pair == "" {
		pair = "BTC/USDT"
	}

	c.log.Info("Starting candle simulation for pair ", pair, " with interval ", c.simulationInterval)
	c.StartCandleSimulation(pair, c.simulationInterval)
}

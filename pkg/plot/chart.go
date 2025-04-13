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
	"github.com/raykavin/backnrun/pkg/core"
	"github.com/raykavin/backnrun/pkg/exchange"
	"github.com/raykavin/backnrun/pkg/logger"
)

// Static assets embedded in the binary
var (
	//go:embed assets
	staticFiles embed.FS
)

// Chart handles the visualization of trading data
type Chart struct {
	sync.Mutex
	port            int
	debug           bool
	candles         map[string][]Candle
	dataframe       map[string]*core.Dataframe
	ordersIDsByPair map[string]*set.LinkedHashSetINT64
	orderByID       map[int64]core.Order
	indicators      []Indicator
	paperWallet     *exchange.PaperWallet
	scriptContent   string
	indexHTML       *template.Template
	strategy        core.Strategy
	lastUpdate      time.Time
	log             logger.Logger
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

// NewChart creates a new chart instance with the provided options
func NewChart(log logger.Logger, options ...Option) (*Chart, error) {
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
	chart.indexHTML, err = template.ParseFS(staticFiles, "assets/chart.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse chart template: %w", err)
	}

	// Read and transpile chart JavaScript
	chartJS, err := staticFiles.ReadFile("assets/chart.js")
	if err != nil {
		return nil, fmt.Errorf("failed to read chart.js: %w", err)
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

	return chart, nil
}

// Start initializes the HTTP server for the chart
func (c *Chart) Start() error {
	// Set up static file handlers
	http.Handle(
		"/assets/",
		http.FileServer(http.FS(staticFiles)),
	)

	// Set up chart.js handler
	http.HandleFunc("/assets/chart.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, c.scriptContent)
	})

	// Set up API handlers
	http.HandleFunc("/health", c.handleHealth)
	http.HandleFunc("/history", c.handleTradingHistoryData)
	http.HandleFunc("/data", c.handleData)
	http.HandleFunc("/", c.handleIndex)

	// Start the server
	fmt.Printf("Chart available at http://localhost:%d\n", c.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", c.port), nil)
}

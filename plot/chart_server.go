package plot

import (
	"fmt"

	"github.com/raykavin/backnrun/core"
)

// ChartServer is a wrapper that combines a Chart with an HTTP server
type ChartServer struct {
	chart  *Chart
	server HTTPServer
	log    core.Logger
}

// NewChartServer creates a new ChartServer
func NewChartServer(chart *Chart, server HTTPServer, log core.Logger) *ChartServer {
	return &ChartServer{
		chart:  chart,
		server: server,
		log:    log,
	}
}

// Start initializes the HTTP server for the chart
func (cs *ChartServer) Start() error {
	// Register handlers on the server
	cs.chart.RegisterHandlers(cs.server)

	// Start simulation if configured
	cs.chart.StartSimulation()

	// Start the server
	port := cs.chart.GetPort()
	fmt.Printf("Chart available at http://localhost:%d\n", port)
	return cs.server.Start(port)
}

/**
 * TradingChart class - Main chart component with TradingView-style resizable panes
 */

import { getCurrentThemeColors } from '../config/Theme.js';
import { formatCandleData, processOrderMarkers, fetchChartData } from '../services/DataService.js';
import { createMainChart, createSecondaryChart, setupResizeHandler, methodExists, syncChartWithMain } from './ChartCreator.js';
import { addCustomMarkers, testMarkerSupport } from '../utils/Markers.js';
import { createTooltip, setupTooltip } from './Tooltip.js';
import { createEquityChart } from './EquityChart.js';
import { applyChartStyles } from '../styles/ChartStyles.js';
import { createElement } from '../utils/Helpers.js';
import { initWebSocket, closeWebSocket, registerMessageHandler, unregisterMessageHandler, isWebSocketConnected } from '../services/WebsocketService.js';
import { PaneResizer } from '../utils/PaneResizer.js';

/**
 * TradingChart class for chart creation and management
 */
export class TradingChart {
  /**
   * Constructor
   */
  constructor() {
    this.chartPanes = [];  // Array of all chart panes (main + indicators)
    this.mainChart = null;
    this.candleSeries = null;
    this.tooltip = null;
    this.pair = '';
    this.buyMarkers = [];
    this.sellMarkers = [];
    this.currentTheme = document.documentElement.getAttribute('data-theme') || 'light';
    this.colors = getCurrentThemeColors();
    this.wsHandlers = {};
    this.paneResizer = null;
    this.additionalCharts = [];
  }

  /**
   * Initialize the chart
   */
  init() {
    // Get pair from URL
    const params = new URLSearchParams(window.location.search);
    this.pair = params.get("pair") || "";

    // Create tooltip
    this.tooltip = createTooltip();

    // Apply chart styles
    applyChartStyles(this.colors);

    // Check if LightweightCharts is available
    if (typeof LightweightCharts === 'undefined') {
      console.error('LightweightCharts library is not loaded!');
      document.getElementById('graph').innerHTML = `
        <div style="padding: 20px; color: red;">
          Error: Chart library not loaded. Please check your network connection and refresh the page.
        </div>
      `;
      return;
    }

    // Setup WebSocket handlers first
    this.setupWebSocketHandlers();

    // Then fetch data through WebSocket
    this.fetchData();
  }

  /**
   * Setup WebSocket handlers for real-time updates
   */
  setupWebSocketHandlers() {
    // Handler for initial data
    this.wsHandlers.initialData = (payload) => {
      console.log('Received initial data via WebSocket');
      this.renderCharts(payload);
    };

    // Handler for new candle
    this.wsHandlers.newCandle = (payload) => {
      console.log('Received new candle via WebSocket', payload);
      if (payload.pair !== this.pair) return;

      const candle = payload.candle;

      // Format the candle for the chart
      const formattedCandle = {
        time: new Date(candle.time).getTime() / 1000,
        open: candle.open,
        high: candle.high,
        low: candle.low,
        close: candle.close,
        volume: candle.volume
      };

      // Update the candlestick series
      if (this.candleSeries && methodExists(this.candleSeries, 'update')) {
        this.candleSeries.update(formattedCandle);

        // If this is a real-time update, ensure the chart stays scrolled to the right
        if (!candle.complete) {
          if (methodExists(this.mainChart, 'timeScale') &&
            methodExists(this.mainChart.timeScale(), 'scrollToRealTime')) {
            this.mainChart.timeScale().scrollToRealTime();
          } else if (methodExists(this.mainChart, 'timeScale') &&
            methodExists(this.mainChart.timeScale(), 'scrollToPosition')) {
            this.mainChart.timeScale().scrollToPosition(0, false);
          }
        }
      }
    };

    // Handler for new order
    this.wsHandlers.newOrder = (payload) => {
      console.log('Received new order via WebSocket', payload);
      if (payload.pair !== this.pair) return;

      const order = payload.order;

      // Only process filled orders
      if (order.status !== 'FILLED') return;

      // Create a marker for the order
      const marker = {
        time: new Date(order.updatedAt).getTime() / 1000,
        position: order.side === 'BUY' ? 'belowBar' : 'aboveBar',
        color: order.side === 'BUY' ? this.colors.UP : this.colors.DOWN,
        shape: order.side === 'BUY' ? 'arrowUp' : 'arrowDown',
        text: order.side === 'BUY' ? 'B' : 'S',
        size: 2,
        price: order.price,
        id: order.id,
        order: order
      };

      // Add the marker to the appropriate array
      if (order.side === 'BUY') {
        this.buyMarkers.push(marker);
      } else {
        this.sellMarkers.push(marker);
      }

      // Update the markers on the chart
      if (this.candleSeries && methodExists(this.candleSeries, 'setMarkers')) {
        try {
          this.candleSeries.setMarkers([...this.buyMarkers, ...this.sellMarkers]);
        } catch (error) {
          console.error('Error updating markers:', error);
          addCustomMarkers(this.mainChart, this.candleSeries,
            [...this.buyMarkers, ...this.sellMarkers], this.colors);
        }
      }
    };

    // Register the handlers
    registerMessageHandler('initialData', this.wsHandlers.initialData);
    registerMessageHandler('newCandle', this.wsHandlers.newCandle);
    registerMessageHandler('newOrder', this.wsHandlers.newOrder);

    // Initialize WebSocket connection
    if (this.pair) {
      initWebSocket(this.pair).catch(error => {
        console.error('Failed to initialize WebSocket:', error);
      });
    }
  }

  /**
   * Toggles the visibility of grid lines on the chart
   */
  toggleGrid() {
    if (!this.gridState) {
      this.gridState = {
        vertLinesVisible: true,
        horzLinesVisible: true
      };
    }

    this.gridState.vertLinesVisible = !this.gridState.vertLinesVisible;
    this.gridState.horzLinesVisible = !this.gridState.horzLinesVisible;

    if (this.mainChart) {
      this.mainChart.applyOptions({
        grid: {
          vertLines: { visible: this.gridState.vertLinesVisible },
          horzLines: { visible: this.gridState.horzLinesVisible }
        }
      });
    }

    if (this.additionalCharts && this.additionalCharts.length > 0) {
      this.additionalCharts.forEach(chart => {
        if (chart) {
          chart.applyOptions({
            grid: {
              vertLines: { visible: this.gridState.vertLinesVisible },
              horzLines: { visible: this.gridState.horzLinesVisible }
            }
          });
        }
      });
    }

    return this.gridState;
  }

  /**
   * Update chart theme colors
   * @param {string} theme - Theme name ('light' or 'dark')
   */
  updateChartTheme(theme) {
    this.currentTheme = theme;
    this.colors = getCurrentThemeColors();
    applyChartStyles(this.colors);
    closeWebSocket();
    this.setupWebSocketHandlers();
    this.fetchData();
  }

  /**
   * Fetch data from the server via WebSocket
   */
  async fetchData() {
    try {
      // Show loading indicator
      const graphContainer = document.getElementById('graph');
      graphContainer.innerHTML = `
        <div class="loading-container">
          <div class="spinner"></div>
          <div class="loading-text">Connecting to WebSocket and loading data...</div>
        </div>
      `;

      // Initialize WebSocket connection if not already connected
      if (!isWebSocketConnected()) {
        await initWebSocket(this.pair);
      }

      // Fetch data through WebSocket
      const data = await fetchChartData(this.pair);

      // Render charts with the received data
      this.renderCharts(data);
    } catch (error) {
      console.error('Error fetching chart data:', error);
      document.getElementById('graph').innerHTML = `
        <div style="padding: 20px; color: red;">
          Error loading chart data: ${error.message}<br>
          Please check your connection and try again.
        </div>
      `;
    }
  }

  /**
   * Render all charts
   * @param {Object} data - Chart data
   */
  renderCharts(data) {
    try {
      // Clear existing content and arrays
      const graphContainer = document.getElementById('graph');
      graphContainer.innerHTML = '';
      this.chartPanes = [];
      this.additionalCharts = [];

      // Count standalone indicators
      const standaloneIndicators = data.indicators ? data.indicators.filter(ind => !ind.overlay) : [];
      const indicatorCount = standaloneIndicators.length;

      
      // Create main chart container for all panes
      const mainChartContainer = createElement('div', 'chart-container tradingview-style-container', graphContainer);
      mainChartContainer.id = 'main-chart-container';
      mainChartContainer.style.height = '100%';
      mainChartContainer.style.border = `1px solid ${this.colors.BORDER}`;
      mainChartContainer.style.borderRadius = '8px';
      mainChartContainer.style.marginBottom = '15px';
      mainChartContainer.style.position = 'relative';
      mainChartContainer.style.display = 'flex';
      mainChartContainer.style.flexDirection = 'column';

      // Create price chart pane (first pane)
      const priceChartPane = createElement('div', 'chart-pane main-chart-pane', mainChartContainer);
      priceChartPane.style.height = indicatorCount > 0 ? '400px' : '100%';
      priceChartPane.style.position = 'relative';
      priceChartPane.style.width = '100%';

      // Add to panes array
      this.chartPanes.push(priceChartPane);

      // Create legend container
      const legendContainer = createElement('div', 'legend-container', priceChartPane);

      // Initialize main chart in the price chart pane
      this.mainChart = createMainChart(priceChartPane, this.colors, LightweightCharts);
      this.mainChart.container = priceChartPane;
      priceChartPane.chartInstance = this.mainChart;

      // Format candle data
      const candleData = formatCandleData(data.candles);

      // Add candlestick series
      try {
        if (methodExists(this.mainChart, 'addCandlestickSeries')) {
          this.candleSeries = this.mainChart.addCandlestickSeries({
            upColor: this.colors.UP,
            downColor: this.colors.DOWN,
            borderVisible: false,
            wickUpColor: this.colors.UP,
            wickDownColor: this.colors.DOWN,
          });
        }
        else if (methodExists(this.mainChart, 'createPriceSeries')) {
          this.candleSeries = this.mainChart.createPriceSeries({
            type: 'candlestick',
            upColor: this.colors.UP,
            downColor: this.colors.DOWN,
            borderVisible: false,
            wickUpColor: this.colors.UP,
            wickDownColor: this.colors.DOWN,
          });
        }
        else if (methodExists(this.mainChart, 'addBarSeries')) {
          this.candleSeries = this.mainChart.addBarSeries({
            upColor: this.colors.UP,
            downColor: this.colors.DOWN,
            thinBars: false,
          });
        }
        else if (methodExists(this.mainChart, 'addAreaSeries')) {
          // Fallback to area series
          this.candleSeries = this.mainChart.addAreaSeries({
            topColor: this.colors.UP,
            bottomColor: this.colors.UP.replace('1)', '0.1)'),
            lineColor: this.colors.UP,
            lineWidth: 2,
          });

          // Convert candlestick data to line data
          const lineData = candleData.map(candle => ({
            time: candle.time,
            value: candle.close,
            orders: candle.orders
          }));

          this.candleSeries.setData(lineData);
        }
        else {
          throw new Error("No suitable chart series method found");
        }
      } catch (error) {
        console.error("Error creating chart series:", error);
        graphContainer.innerHTML = `
          <div style="padding: 20px; color: red;">
            Error creating chart: ${error.message}<br>
            Please check browser console for details and ensure you're using the latest version of the library.
          </div>
        `;
        return;
      }

      // Set data if not already (in the area series case)
      if (methodExists(this.candleSeries, 'setData') &&
        this.candleSeries.constructor.name !== 'AreaSeries') {
        this.candleSeries.setData(candleData);
      }

      // Process markers
      const { allMarkers, buyMarkers, sellMarkers } = processOrderMarkers(data.candles, this.colors);
      this.buyMarkers = buyMarkers;
      this.sellMarkers = sellMarkers;

      // Add markers
      if (allMarkers.length > 0) {
        const hasMarkerSupport = testMarkerSupport(this.mainChart);
        let nativeMarkersWorked = false;

        if (hasMarkerSupport && methodExists(this.candleSeries, 'setMarkers')) {
          try {
            this.candleSeries.setMarkers(allMarkers);
            nativeMarkersWorked = true;
          } catch (error) {
            console.error("Native setMarkers method failed:", error);
          }
        }

        if (!nativeMarkersWorked) {
          addCustomMarkers(this.mainChart, this.candleSeries, allMarkers, this.colors);
        }
      }

      // Setup tooltip
      setupTooltip(this.mainChart, this.tooltip, this.buyMarkers, this.sellMarkers);

      // Add overlay indicators to main chart
      const overlayIndicators = data.indicators ? data.indicators.filter(ind => ind.overlay) : [];
      if (overlayIndicators.length > 0) {
        this.addOverlayIndicators(overlayIndicators, this.mainChart, priceChartPane);
      }

      // Create indicator panes
      if (standaloneIndicators.length > 0) {
        standaloneIndicators.forEach((indicator, index) => {
          // Create pane
          const indicatorPane = createElement('div', 'chart-pane indicator-pane', mainChartContainer);
          indicatorPane.dataset.name = indicator.name;
          indicatorPane.style.height = '150px';
          indicatorPane.style.position = 'relative';
          indicatorPane.style.width = '100%';

          // Add to panes array
          this.chartPanes.push(indicatorPane);

          // Create header
          const indicatorHeader = createElement('div', 'indicator-header', indicatorPane);
          indicatorHeader.textContent = indicator.name;

          // Create chart
          const showTimeScale = index === standaloneIndicators.length - 1;
          const indicatorChart = createSecondaryChart(indicatorPane, this.colors, LightweightCharts, showTimeScale);
          indicatorChart.container = indicatorPane;
          indicatorPane.chartInstance = indicatorChart;

          // Add to charts array
          this.additionalCharts.push(indicatorChart);

          // Sync with main chart
          syncChartWithMain(this.mainChart, indicatorChart);

          // Add indicator series
          this.addIndicatorSeries(indicator, indicatorChart, indicatorPane);
        });
      }

      // Add equity chart as a separate chart below
      if (data.equity_values && data.equity_values.length > 0) {
        const equityHeight = indicatorCount > 0 ? 120 : 150;
        data.equityHeight = equityHeight;

        const equityChart = createEquityChart(
          data,
          graphContainer,
          this.mainChart,
          this.colors,
          (container, showTimeScale) => createSecondaryChart(container, this.colors, LightweightCharts, showTimeScale)
        );

        if (equityChart) {
          this.additionalCharts.push(equityChart);
        }
      }

      // Initialize resizer
      this.initPaneResizer();

      // Fit content
      this.fitAllCharts();

    } catch (error) {
      console.error("Error rendering charts:", error);
      document.getElementById('graph').innerHTML = `
        <div style="padding: 20px; color: red;">
          Error rendering charts: ${error.message}<br>
          Please check browser console for details.
        </div>
      `;
    }
  }

  /**
   * Add overlay indicators to the main chart
   * @param {Array} indicators - Overlay indicators
   * @param {Object} chart - Main chart instance
   * @param {HTMLElement} container - Container element
   */
  addOverlayIndicators(indicators, chart, container) {
    // Create or get legend
    let legend = container.querySelector('.legend-container');
    if (!legend) {
      legend = createElement('div', 'legend-container', container);
    }

    indicators.forEach(indicator => {
      if (!indicator.metrics || indicator.metrics.length === 0) return;

      indicator.metrics.forEach(metric => {
        if (!metric.time || !metric.value || metric.time.length !== metric.value.length) return;

        // Format data
        const indicatorData = metric.time.map((time, i) => ({
          time: new Date(time).getTime() / 1000,
          value: metric.value[i]
        }));

        // Add series
        try {
          const indicatorSeries = chart.addLineSeries({
            color: metric.color || this.colors.DEFAULT_INDICATOR || '#2962FF',
            lineWidth: 1.5,
            priceLineVisible: false,
            lastValueVisible: true,
            priceFormat: {
              type: 'price',
              precision: 5,
              minMove: 0.00001,
            }
          });
          indicatorSeries.setData(indicatorData);

          // Add to legend
          const name = indicator.name + (metric.name ? ` - ${metric.name}` : '');
          this.addLegendItem(legend, name, metric.color || this.colors.DEFAULT_INDICATOR || '#2962FF');
        } catch (error) {
          console.error(`Failed to add overlay indicator ${indicator.name}:`, error);
        }
      });
    });
  }

/**
   * Add series data for an indicator
   * @param {Object} indicator - Indicator definition
   * @param {Object} chart - Chart instance for the indicator
   * @param {HTMLElement} container - Container element
   */
  addIndicatorSeries(indicator, chart, container) {
    // Create legend container if it doesn't exist
    let legend = container.querySelector('.legend-container');
    if (!legend) {
      legend = createElement('div', 'legend-container', container);
    }

    // Check for metrics data
    if (!indicator.metrics || indicator.metrics.length === 0) {
      console.error(`No metrics found for indicator ${indicator.name}`);
      const header = container.querySelector('.indicator-header');
      if (header) {
        header.innerHTML += ' <span style="color:red">(No data available)</span>';
      }
      return;
    }

    // Add each metric as a series
    indicator.metrics.forEach(metric => {
      if (!metric.time || !metric.value || metric.time.length === 0) {
        console.error(`Invalid metric data for ${indicator.name}:`, metric);
        return;
      }

      try {
        // Format data
        const indicatorData = metric.time.map((time, i) => ({
          time: new Date(time).getTime() / 1000,
          value: metric.value[i]
        }));

        // Determine series type
        let indicatorSeries;

        if (metric.style === 'histogram') {
          indicatorSeries = chart.addHistogramSeries({
            color: metric.color || this.colors.DEFAULT_INDICATOR || '#2962FF',
            priceLineVisible: false,
            lastValueVisible: true,
            priceFormat: {
              type: 'price',
              precision: 5,
              minMove: 0.00001,
            },
            base: 0, // Important for MACD histograms
          });
        } else {
          indicatorSeries = chart.addLineSeries({
            color: metric.color || this.colors.DEFAULT_INDICATOR || '#2962FF',
            lineWidth: 1.5,
            priceLineVisible: false,
            lastValueVisible: true,
            priceFormat: {
              type: 'price',
              precision: 5,
              minMove: 0.00001,
            }
          });
        }

        indicatorSeries.setData(indicatorData);

        // Add to legend
        const name = metric.name || indicator.name;
        this.addLegendItem(legend, name, metric.color || this.colors.DEFAULT_INDICATOR || '#2962FF');

      } catch (error) {
        console.error(`Failed to add series for ${metric.name || 'unnamed'}:`, error);
      }
    });

    // Adjust price scale
    try {
      const rightPriceScale = chart.priceScale('right');
      if (rightPriceScale) {
        rightPriceScale.applyOptions({
          autoScale: true,
          scaleMargins: {
            top: 0.1,
            bottom: 0.1,
          },
        });
      }
    } catch (error) {
      console.error('Failed to adjust price scale:', error);
    }
  }

  /**
   * Add a legend item
   * @param {HTMLElement} legend - Legend container
   * @param {string} name - Item name
   * @param {string} color - Item color
   */
  addLegendItem(legend, name, color) {
    const item = document.createElement('div');
    item.className = 'legend-item';
    
    const marker = document.createElement('div');
    marker.className = 'legend-marker';
    marker.style.backgroundColor = color;
    
    const label = document.createElement('span');
    label.textContent = name;
    
    item.appendChild(marker);
    item.appendChild(label);
    legend.appendChild(item);
  }

  /**
   * Initialize pane resizer
   */
  initPaneResizer() {
    // Destroy existing resizer if any
    if (this.paneResizer) {
      this.paneResizer.destroy();
    }
    
    // Create new resizer
    this.paneResizer = new PaneResizer({
      minPaneHeight: 80,
      handleHeight: 8,
      handleClass: 'pane-resize-handle',
      onResize: () => {
        // When a resize occurs, fit content in all charts
        this.fitAllCharts();
      }
    });
    
    // Setup panes for resizing
    this.paneResizer.setupPanes(this.chartPanes);
  }
  
  /**
   * Fit content in all charts
   */
  fitAllCharts() {
    // Fit main chart
    if (this.mainChart && methodExists(this.mainChart, 'timeScale') &&
      methodExists(this.mainChart.timeScale(), 'fitContent')) {
      this.mainChart.timeScale().fitContent();
    }
    
    // Fit additional charts
    this.additionalCharts.forEach(chart => {
      if (chart && methodExists(chart, 'timeScale') &&
        methodExists(chart.timeScale(), 'fitContent')) {
        chart.timeScale().fitContent();
      }
    });
  }

  /**
   * Clean up resources when the chart is destroyed
   */
  destroy() {
    // Unregister WebSocket handlers
    if (this.wsHandlers) {
      unregisterMessageHandler('initialData', this.wsHandlers.initialData);
      unregisterMessageHandler('newCandle', this.wsHandlers.newCandle);
      unregisterMessageHandler('newOrder', this.wsHandlers.newOrder);
    }

    // Close WebSocket connection
    closeWebSocket();
    
    // Destroy pane resizer
    if (this.paneResizer) {
      this.paneResizer.destroy();
      this.paneResizer = null;
    }

    // Clean up chart resources
    if (this.mainChart) {
      this.mainChart.remove();
      this.mainChart = null;
    }

    this.additionalCharts.forEach(chart => {
      if (chart) {
        chart.remove();
      }
    });
    
    // Clear arrays
    this.additionalCharts = [];
    this.chartPanes = [];
  }
}
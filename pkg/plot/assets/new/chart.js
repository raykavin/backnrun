/**
 * BackNRun - Trading Chart Visualization
 * 
 * This script handles the visualization of trading data including
 * price charts, trade markers, equity values, and technical indicators.
 * Added dark mode support and wider chart display.
 */

// Constants
const ORDER_TYPES = {
  LIMIT: "LIMIT",
  MARKET: "MARKET",
  STOP_LOSS: "STOP_LOSS",
  LIMIT_MAKER: "LIMIT_MAKER"
};

const SIDES = {
  SELL: "SELL",
  BUY: "BUY"
};

const STATUS = {
  FILLED: "FILLED"
};

// Theme-sensitive colors
const THEME_COLORS = {
  light: {
    UP: '#26a69a',
    DOWN: '#ef5350',
    EQUITY: 'rgba(38, 166, 154, 1)',
    ASSET: 'rgba(239, 83, 80, 1)',
    DEFAULT_INDICATOR: '#2196F3',
    GRID: 'rgba(197, 203, 206, 0.5)',
    BORDER: 'rgba(197, 203, 206, 1)',
    BACKGROUND: '#ffffff',
    TEXT: '#333'
  },
  dark: {
    UP: '#4ecca3',
    DOWN: '#ff6b6b',
    EQUITY: 'rgba(78, 204, 163, 1)',
    ASSET: 'rgba(255, 107, 107, 1)',
    DEFAULT_INDICATOR: '#64b5f6',
    GRID: 'rgba(120, 123, 134, 0.3)',
    BORDER: 'rgba(120, 123, 134, 1)',
    BACKGROUND: '#1e1e1e',
    TEXT: '#e0e0e0'
  }
};

// Initial colors based on current theme
let COLORS = getCurrentThemeColors();

// Get current theme colors based on data-theme attribute
function getCurrentThemeColors() {
  const isDarkMode = document.documentElement.getAttribute('data-theme') === 'dark';
  return isDarkMode ? THEME_COLORS.dark : THEME_COLORS.light;
}

/**
 * Helper Functions
 */

// Extract values from arrays
function unpack(rows, key) {
  return rows.map(row => row[key]);
}

// Format dates for display
function formatDate(dateStr) {
  const date = new Date(dateStr);
  return date.toLocaleString();
}

// Create DOM element with optional class and parent
function createElement(tag, className, parent) {
  const element = document.createElement(tag);
  if (className) element.className = className;
  if (parent) parent.appendChild(element);
  return element;
}

// Add an item to the legend
function addLegendItem(container, name, color) {
  const item = createElement('div', 'legend-item', container);

  const marker = createElement('div', 'legend-marker', item);
  marker.style.backgroundColor = color;

  const label = createElement('div', '', item);
  label.textContent = name;

  return item;
}

/**
 * Chart Creation and Management
 */
class TradingChart {
  constructor() {
    this.additionalCharts = [];
    this.mainChart = null;
    this.candleSeries = null;
    this.tooltip = null;
    this.pair = '';
    this.buyMarkers = [];
    this.sellMarkers = [];
    this.currentTheme = document.documentElement.getAttribute('data-theme') || 'light';
    // We'll use the global LightweightCharts variable directly instead of storing it
  }

  // Initialize the chart when DOM is ready
  init() {
    const params = new URLSearchParams(window.location.search);
    this.pair = params.get("pair") || "";

    this.tooltip = createElement('div', 'tooltip', document.body);

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

    // Store reference to chart library
    this.chartLib = LightweightCharts;

    this.fetchData();
  }

  // Update chart theme colors
  updateChartTheme(theme) {
    this.currentTheme = theme;
    COLORS = theme === 'dark' ? THEME_COLORS.dark : THEME_COLORS.light;

    // Re-fetch and re-render with new theme colors
    this.fetchData();
  }

  // Fetch data from the server
  async fetchData() {
    try {
      const response = await fetch("/data?pair=" + this.pair);
      const data = await response.json();
      this.renderCharts(data);
    } catch (error) {
      console.error('Error fetching chart data:', error);
      document.getElementById('graph').innerHTML = `
        <div style="padding: 20px; color: red;">
          Error loading chart data: ${error.message}
        </div>
      `;
    }
  }

  // Create and configure the main chart
  createMainChart(container) {
    if (typeof LightweightCharts === 'undefined' || typeof LightweightCharts.createChart !== 'function') {
      throw new Error('LightweightCharts library not properly initialized');
    }

    // Use the global LightweightCharts variable directly instead of this.chartLib
    return LightweightCharts.createChart(container, {
      width: container.clientWidth,
      height: container.clientHeight,
      layout: {
        backgroundColor: COLORS.BACKGROUND,
        textColor: COLORS.TEXT,
      },
      grid: {
        vertLines: { color: COLORS.GRID },
        horzLines: { color: COLORS.GRID },
      },
      crosshair: {
        mode: LightweightCharts.CrosshairMode.Normal,
      },
      rightPriceScale: {
        borderColor: COLORS.BORDER,
      },
      timeScale: {
        borderColor: COLORS.BORDER,
        timeVisible: true,
      },
    });
  }

  // Create a secondary chart (for indicators, equity, etc.)
  createSecondaryChart(container, showTimeScale = false) {
    if (typeof LightweightCharts === 'undefined' || typeof LightweightCharts.createChart !== 'function') {
      throw new Error('LightweightCharts library not properly initialized');
    }

    // Use the global LightweightCharts variable directly
    const chart = LightweightCharts.createChart(container, {
      width: container.clientWidth,
      height: container.clientHeight,
      layout: {
        backgroundColor: COLORS.BACKGROUND,
        textColor: COLORS.TEXT,
      },
      grid: {
        vertLines: { color: COLORS.GRID },
        horzLines: { color: COLORS.GRID },
      },
      rightPriceScale: {
        borderColor: COLORS.BORDER,
      },
      timeScale: {
        borderColor: COLORS.BORDER,
        visible: showTimeScale,
      },
    });

    this.additionalCharts.push(chart);
    return chart;
  }

  // Set up window resize handler
  setupResizeHandler() {
    window.addEventListener('resize', () => {
      if (this.mainChart) {
        const container = document.getElementById('main-chart');
        this.mainChart.applyOptions({
          width: container.clientWidth,
          height: container.clientHeight,
        });
      }

      this.additionalCharts.forEach(chart => {
        chart.applyOptions({
          width: chart.container.clientWidth,
          height: chart.container.clientHeight,
        });
      });
    });
  }

  // Format candle data for TradingView
  formatCandleData(candles) {
    return candles.map(candle => ({
      time: new Date(candle.time).getTime() / 1000,
      open: candle.open,
      high: candle.high,
      low: candle.low,
      close: candle.close,
      orders: candle.orders || []
    }));
  }

  // Process buy/sell markers
  processOrderMarkers(candles) {
    this.buyMarkers = [];
    this.sellMarkers = [];

    candles.forEach(candle => {
      if (!candle.orders) return;

      candle.orders
        .filter(order => order.status === STATUS.FILLED)
        .forEach(order => {
          const marker = {
            time: new Date(candle.time).getTime() / 1000,
            position: order.side === SIDES.BUY ? 'belowBar' : 'aboveBar',
            color: order.side === SIDES.BUY ? COLORS.UP : COLORS.DOWN,
            shape: order.side === SIDES.BUY ? 'arrowUp' : 'arrowDown',
            text: order.side === SIDES.BUY ? 'B' : 'S',
            size: 2,
            id: order.id,
            order: order
          };

          if (order.side === SIDES.BUY) {
            this.buyMarkers.push(marker);
          } else {
            this.sellMarkers.push(marker);
          }
        });
    });

    return [...this.buyMarkers, ...this.sellMarkers];
  }

  // Setup tooltip functionality
  setupTooltip() {
    this.mainChart.subscribeCrosshairMove(param => {
      if (!param.point || !param.time) {
        this.tooltip.style.display = 'none';
        return;
      }

      const markers = [...this.buyMarkers, ...this.sellMarkers];
      const marker = markers.find(m => m.time === param.time);

      if (marker && marker.order) {
        const order = marker.order;
        this.tooltip.innerHTML = `
          <div><strong>${order.side} Order</strong></div>
          <div>Time: ${formatDate(order.updated_at || order.created_at)}</div>
          <div>ID: ${order.id}</div>
          <div>Price: ${order.price.toLocaleString()}</div>
          <div>Size: ${order.quantity.toPrecision(4).toLocaleString()}</div>
          <div>Type: ${order.type}</div>
          ${order.profit ? `<div>Profit: ${(order.profit * 100).toPrecision(2).toLocaleString()}%</div>` : ''}
        `;

        const x = param.point.x;
        const y = param.point.y;

        this.tooltip.style.display = 'block';
        this.tooltip.style.left = `${x + 15}px`;
        this.tooltip.style.top = `${y + 15}px`;
      } else {
        this.tooltip.style.display = 'none';
      }
    });
  }

  // Create equity chart
  createEquityChart(data, graphContainer) {
    if (!data.equity_values || data.equity_values.length === 0) return;

    // Create container
    const equityContainer = createElement('div', 'chart-container', graphContainer);
    equityContainer.style.height = '150px';

    // Create chart
    const equityChart = this.createSecondaryChart(equityContainer, false);
    equityChart.container = equityContainer;

    // Sync with main chart
    this.syncChartWithMain(equityChart);

    // Create legend
    const equityLegend = createElement('div', 'legend-container', equityContainer);

    // Format equity data
    const equityData = data.equity_values.map(item => ({
      time: new Date(item.time).getTime() / 1000,
      value: item.value
    }));

    // Add equity series
    const equitySeries = equityChart.addAreaSeries({
      topColor: this.currentTheme === 'dark' ? 'rgba(78, 204, 163, 0.56)' : 'rgba(38, 166, 154, 0.56)',
      bottomColor: this.currentTheme === 'dark' ? 'rgba(78, 204, 163, 0.04)' : 'rgba(38, 166, 154, 0.04)',
      lineColor: COLORS.EQUITY,
      lineWidth: 2,
    });
    equitySeries.setData(equityData);

    // Add equity to legend
    addLegendItem(equityLegend, `Equity (${data.quote})`, COLORS.EQUITY);

    // Handle drawdown if available
    this.addDrawdownVisualization(data, equityChart, equitySeries, equityLegend);

    // Add asset values if available
    this.addAssetSeries(data, equityChart, equityLegend);

    return equityChart;
  }

  // Add drawdown visualization
  addDrawdownVisualization(data, chart, series, legend) {
    if (!data.max_drawdown) return;

    // Find time range for drawdown
    const startTime = new Date(data.max_drawdown.start).getTime() / 1000;
    const endTime = new Date(data.max_drawdown.end).getTime() / 1000;

    // Add drawdown marker
    const drawdownMarker = {
      time: (startTime + endTime) / 2,
      position: 'aboveBar',
      color: COLORS.DOWN,
      shape: 'square',
      text: `Drawdown: ${data.max_drawdown.value}%`,
    };

    series.setMarkers([drawdownMarker]);

    // Add vertical lines
    if (chart.addVerticalLine) {
      chart.addVerticalLine({
        time: startTime,
        color: `rgba(${this.currentTheme === 'dark' ? '255, 107, 107' : '239, 83, 80'}, 0.5)`,
        lineWidth: 1,
        lineStyle: LightweightCharts.LineStyle.Dashed,
      });

      chart.addVerticalLine({
        time: endTime,
        color: `rgba(${this.currentTheme === 'dark' ? '255, 107, 107' : '239, 83, 80'}, 0.5)`,
        lineWidth: 1,
        lineStyle: LightweightCharts.LineStyle.Dashed,
      });
    }

    // Add to legend
    addLegendItem(legend, `Max Drawdown: ${data.max_drawdown.value}%`, COLORS.DOWN);
  }

  // Add asset value series
  addAssetSeries(data, chart, legend) {
    if (!data.asset_values || data.asset_values.length === 0) return;

    // Format asset data
    const assetData = data.asset_values.map(item => ({
      time: new Date(item.time).getTime() / 1000,
      value: item.value
    }));

    // Add asset series
    const assetSeries = chart.addLineSeries({
      color: COLORS.ASSET,
      lineWidth: 2,
    });
    assetSeries.setData(assetData);

    // Add to legend
    addLegendItem(legend, `Position (${data.asset}/${data.quote})`, COLORS.ASSET);
  }

  // Sync a chart's time scale with the main chart
  syncChartWithMain(chart) {
    this.mainChart.timeScale().subscribeVisibleTimeRangeChange(timeRange => {
      chart.timeScale().setVisibleRange(timeRange);
    });

    if (chart.setCrosshairPosition) {
      this.mainChart.timeScale().subscribeCrosshairMove(param => {
        chart.setCrosshairPosition(param.time, param.point.x);
      });
    }
  }

  // Add indicators to charts
  addIndicators(data, graphContainer, mainLegend) {
    if (!data.indicators || data.indicators.length === 0) return;

    // Group indicators
    const overlayIndicators = data.indicators.filter(ind => ind.overlay);
    const standaloneIndicators = data.indicators.filter(ind => !ind.overlay);

    // Add overlay indicators to main chart
    this.addOverlayIndicators(overlayIndicators, mainLegend);

    // Add standalone indicators as separate charts
    this.addStandaloneIndicators(standaloneIndicators, graphContainer);
  }

  // Add overlay indicators to the main chart
  addOverlayIndicators(indicators, legend) {
    indicators.forEach(indicator => {
      indicator.metrics.forEach(metric => {
        // Format indicator data
        const indicatorData = metric.time.map((time, i) => ({
          time: new Date(time).getTime() / 1000,
          value: metric.value[i]
        }));

        // Add indicator series
        try {
          const indicatorSeries = this.mainChart.addLineSeries({
            color: metric.color || COLORS.DEFAULT_INDICATOR,
            lineWidth: 1,
            priceLineVisible: false,
          });
          indicatorSeries.setData(indicatorData);

          // Add to legend
          const name = indicator.name + (metric.name ? ` - ${metric.name}` : '');
          addLegendItem(legend, name, metric.color || COLORS.DEFAULT_INDICATOR);
        } catch (error) {
          console.error(`Failed to add overlay indicator ${indicator.name}:`, error);
        }
      });
    });
  }

  // Add standalone indicators as separate charts
  addStandaloneIndicators(indicators, graphContainer) {
    indicators.forEach((indicator, index) => {
      try {
        // Create container
        const indicatorContainer = createElement('div', 'chart-container', graphContainer);
        indicatorContainer.style.height = '150px';

        // Create chart
        const showTimeScale = index === indicators.length - 1; // Only show time on last indicator
        const indicatorChart = this.createSecondaryChart(indicatorContainer, showTimeScale);
        indicatorChart.container = indicatorContainer;

        // Sync with main chart
        this.syncChartWithMain(indicatorChart);

        // Create legend
        const indicatorLegend = createElement('div', 'legend-container', indicatorContainer);

        // Add title to legend
        const titleElement = createElement('div', '', indicatorLegend);
        titleElement.innerHTML = `<strong>${indicator.name}</strong>`;

        // Add each metric as a series
        indicator.metrics.forEach(metric => {
          // Format data
          const indicatorData = metric.time.map((time, i) => ({
            time: new Date(time).getTime() / 1000,
            value: metric.value[i]
          }));

          // Determine series type
          let indicatorSeries;
          if (metric.style === 'histogram') {
            indicatorSeries = indicatorChart.addHistogramSeries({
              color: metric.color || COLORS.DEFAULT_INDICATOR,
              priceLineVisible: false,
            });
          } else {
            indicatorSeries = indicatorChart.addLineSeries({
              color: metric.color || COLORS.DEFAULT_INDICATOR,
              lineWidth: 1,
              priceLineVisible: false,
            });
          }

          indicatorSeries.setData(indicatorData);

          // Add to legend
          const name = metric.name || indicator.name;
          addLegendItem(indicatorLegend, name, metric.color || COLORS.DEFAULT_INDICATOR);
        });
      } catch (error) {
        console.error(`Failed to add standalone indicator ${indicator.name}:`, error);
      }
    });
  }

  // Safely check if a method exists on an object
  methodExists(obj, methodName) {
    return obj && typeof obj[methodName] === 'function';
  }

  // Render all charts
  renderCharts(data) {
    try {
      // Clear existing content
      const graphContainer = document.getElementById('graph');
      graphContainer.innerHTML = '';

      // Create main chart container
      const mainChartContainer = createElement('div', 'chart-container', graphContainer);
      mainChartContainer.id = 'main-chart';

      // Create legend container
      const legendContainer = createElement('div', 'legend-container', mainChartContainer);

      // Initialize main chart
      this.mainChart = this.createMainChart(mainChartContainer);

      // Store container reference
      this.mainChart.container = mainChartContainer;

      // Setup resize handler
      this.setupResizeHandler();

      // Format candle data
      const candleData = this.formatCandleData(data.candles);

      // Log chart object structure to debug
      console.log("Chart object:", this.mainChart);
      console.log("Chart methods:", Object.getOwnPropertyNames(this.mainChart).filter(
        prop => typeof this.mainChart[prop] === 'function'
      ));

      // Add candlestick series with proper error handling
      try {
        // Check for method existence using the proper approach
        if (this.methodExists(this.mainChart, 'addCandlestickSeries')) {
          this.candleSeries = this.mainChart.addCandlestickSeries({
            upColor: COLORS.UP,
            downColor: COLORS.DOWN,
            borderVisible: false,
            wickUpColor: COLORS.UP,
            wickDownColor: COLORS.DOWN,
          });
          console.log("Using addCandlestickSeries method");
        }
        else if (this.methodExists(this.mainChart, 'createPriceSeries')) {
          this.candleSeries = this.mainChart.createPriceSeries({
            type: 'candlestick',
            upColor: COLORS.UP,
            downColor: COLORS.DOWN,
            borderVisible: false,
            wickUpColor: COLORS.UP,
            wickDownColor: COLORS.DOWN,
          });
          console.log("Using createPriceSeries method");
        }
        else if (this.methodExists(this.mainChart, 'addBarSeries')) {
          this.candleSeries = this.mainChart.addBarSeries({
            upColor: COLORS.UP,
            downColor: COLORS.DOWN,
            thinBars: false,
          });
          console.log("Using addBarSeries method as fallback");
        }
        else if (this.methodExists(this.mainChart, 'addAreaSeries')) {
          // Last resort fallback to area series
          this.candleSeries = this.mainChart.addAreaSeries({
            topColor: COLORS.UP,
            bottomColor: this.currentTheme === 'dark' ? 'rgba(78, 204, 163, 0.1)' : 'rgba(38, 166, 154, 0.1)',
            lineColor: COLORS.UP,
            lineWidth: 2,
          });

          // For area series, we need to convert candlestick data to line data
          const lineData = candleData.map(candle => ({
            time: candle.time,
            value: candle.close,
            orders: candle.orders
          }));

          this.candleSeries.setData(lineData);
          console.log("Using addAreaSeries method as last resort");
        }
        else {
          throw new Error("No suitable chart series method found");
        }
      } catch (error) {
        console.error("Error creating chart series:", error);

        // Show error message to user
        graphContainer.innerHTML = `
          <div style="padding: 20px; color: red;">
            Error creating chart: ${error.message}<br>
            Please check browser console for details and ensure you're using the latest version of the library.
          </div>
        `;
        return;
      }

      // Set data if we haven't already (in the area series fallback case)
      if (this.methodExists(this.candleSeries, 'setData') &&
        this.candleSeries.constructor.name !== 'AreaSeries') {
        this.candleSeries.setData(candleData);
      }

      // Add legend item for candlestick
      addLegendItem(legendContainer, 'Candles', COLORS.UP);

      // Process markers
      const markers = this.processOrderMarkers(data.candles);

      // Add markers if method exists
      if (this.methodExists(this.candleSeries, 'setMarkers')) {
        this.candleSeries.setMarkers(markers);

        // Add legend items for markers
        if (this.buyMarkers.length > 0) {
          addLegendItem(legendContainer, 'Buy Points', COLORS.UP);
        }
        if (this.sellMarkers.length > 0) {
          addLegendItem(legendContainer, 'Sell Points', COLORS.DOWN);
        }
      }

      // Setup tooltip
      this.setupTooltip();

      // Create equity chart
      this.createEquityChart(data, graphContainer);

      // Add indicators
      this.addIndicators(data, graphContainer, legendContainer);

      // Fit content
      if (this.methodExists(this.mainChart.timeScale(), 'fitContent')) {
        this.mainChart.timeScale().fitContent();
      }
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
}

// Initialize when DOM is ready and ensure the library is loaded
document.addEventListener("DOMContentLoaded", function () {
  // Check if the library is properly loaded
  if (typeof LightweightCharts === 'undefined') {
    console.error('LightweightCharts library not found! Attempting to load it...');

    // Try to load the library dynamically
    const script = document.createElement('script');
    script.src = 'https://unpkg.com/lightweight-charts/dist/lightweight-charts.standalone.production.js';
    script.onload = function () {
      console.log('LightweightCharts library loaded successfully!');
      initializeChart();
    };
    script.onerror = function () {
      console.error('Failed to load LightweightCharts library!');
      document.getElementById('graph').innerHTML = `
        <div style="padding: 20px; color: red;">
          Error: Failed to load chart library. Please check your network connection and refresh the page.
        </div>
      `;
    };
    document.head.appendChild(script);
  } else {
    // Library already loaded, initialize the chart
    initializeChart();
  }

  function initializeChart() {
    try {
      // Debug the library
      console.log("LightweightCharts library:", LightweightCharts);
      console.log("LightweightCharts.createChart:", typeof LightweightCharts.createChart);

      // Check if createChart exists
      if (typeof LightweightCharts.createChart !== 'function') {
        throw new Error('LightweightCharts.createChart is not a function. Library may be corrupted or incompatible.');
      }

      const chart = new TradingChart();
      chart.init();

      // Make chart accessible globally (for debugging and theme toggling)
      window.tradingChart = chart;
    } catch (error) {
      console.error("Error initializing chart:", error);
      document.getElementById('graph').innerHTML = `
        <div style="padding: 20px; color: red;">
          Error initializing chart: ${error.message}<br>
          Please check browser console for details.
        </div>
      `;
    }
  }
});
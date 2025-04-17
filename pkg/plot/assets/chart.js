/**
 * BackNRun - Trading Chart Visualization
 * 
 * This script handles the visualization of trading data including
 * price charts, trade markers, equity values, and technical indicators.
 * Added dark mode support and wider chart display.
 * Fixed standalone indicator display issues.
 */

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

// Constants
const ORDER_TYPES = {
  LIMIT: "LIMIT",
  MARKET: "MARKET",
  STOP_LOSS: "STOP_LOSS",
  LIMIT_MAKER: "LIMIT_MAKER"
}

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
 * Function to check if marker support is available
 */
function testMarkerSupport(chart) {
  try {
    // Create a test series
    const testSeries = chart.addCandlestickSeries();
    // Check if setMarkers exists
    const hasSetMarkers = typeof testSeries.setMarkers === 'function';
    // Remove the test series if possible
    if (typeof testSeries.remove === 'function') {
      testSeries.remove();
    }
    return hasSetMarkers;
  } catch (e) {
    console.error("Error testing marker support:", e);
    return false;
  }
}

/**
 * Add custom markers as separate series
 */
function addCustomMarkers(chart, candleSeries, markers) {
  // Create separate series for buy and sell markers
  const buyMarkers = markers.filter(m => m.position === 'belowBar');
  const sellMarkers = markers.filter(m => m.position === 'aboveBar');

  console.log(`Adding ${buyMarkers.length} buy markers and ${sellMarkers.length} sell markers as separate series`);

  if (buyMarkers.length > 0) {
    // Format buy markers data for scatter series
    const buyData = buyMarkers.map(marker => ({
      time: marker.time,
      value: marker.price || marker.order.price
    }));

    try {
      // Add buy markers as a scatter series
      const buyMarkerSeries = chart.addLineSeries({
        color: COLORS.UP,
        lineWidth: 0,
        pointsVisible: true,
        pointSize: 8,
        pointShape: 'circle',
        pointFill: COLORS.UP,
        pointBorderColor: COLORS.UP,
        pointBorderWidth: 0,
        priceLineVisible: false,
        lastValueVisible: false,
      });

      buyMarkerSeries.setData(buyData);
      console.log('Buy marker series created successfully');
    } catch (error) {
      console.error('Failed to create buy marker series:', error);

      // Fallback to simpler series if the advanced options fail
      try {
        const simpleBuyMarkerSeries = chart.addLineSeries({
          color: COLORS.UP,
          lineWidth: 0,
          priceLineVisible: false,
          lastValueVisible: false,
        });

        simpleBuyMarkerSeries.setData(buyData);
        console.log('Simple buy marker series created as fallback');
      } catch (fallbackError) {
        console.error('Fallback buy marker series also failed:', fallbackError);
      }
    }
  }

  if (sellMarkers.length > 0) {
    // Format sell markers data for scatter series
    const sellData = sellMarkers.map(marker => ({
      time: marker.time,
      value: marker.price || marker.order.price
    }));

    try {
      // Add sell markers as a scatter series
      const sellMarkerSeries = chart.addLineSeries({
        color: COLORS.DOWN,
        lineWidth: 0,
        pointsVisible: true,
        pointSize: 8,
        pointShape: 'circle',
        pointFill: COLORS.DOWN,
        pointBorderColor: COLORS.DOWN,
        pointBorderWidth: 0,
        priceLineVisible: false,
        lastValueVisible: false,
      });

      sellMarkerSeries.setData(sellData);
      console.log('Sell marker series created successfully');
    } catch (error) {
      console.error('Failed to create sell marker series:', error);

      // Fallback to simpler series if the advanced options fail
      try {
        const simpleSellMarkerSeries = chart.addLineSeries({
          color: COLORS.DOWN,
          lineWidth: 0,
          priceLineVisible: false,
          lastValueVisible: false,
        });

        simpleSellMarkerSeries.setData(sellData);
        console.log('Simple sell marker series created as fallback');
      } catch (fallbackError) {
        console.error('Fallback sell marker series also failed:', fallbackError);
      }
    }
  }
}

/**
 * Create HTML text markers at buy/sell points
 */
function createTextMarkers(chart, markers) {
  // Check if the createElement method exists (needed for HTML markers)
  if (typeof document.createElement !== 'function') {
    console.error('Document createElement not available, cannot create text markers');
    return;
  }

  // Create container for markers if it doesn't exist
  let markerContainer = document.getElementById('custom-markers-container');
  if (!markerContainer) {
    markerContainer = document.createElement('div');
    markerContainer.id = 'custom-markers-container';
    markerContainer.style.position = 'absolute';
    markerContainer.style.top = '0';
    markerContainer.style.left = '0';
    markerContainer.style.width = '100%';
    markerContainer.style.height = '100%';
    markerContainer.style.pointerEvents = 'none'; // Pass through mouse events

    // Find the chart container and append
    const chartContainer = document.getElementById('graph');
    if (chartContainer) {
      chartContainer.style.position = 'relative'; // Ensure relative positioning
      chartContainer.appendChild(markerContainer);
    } else {
      console.error('Chart container not found, cannot add marker container');
      return;
    }
  } else {
    // Clear existing markers
    markerContainer.innerHTML = '';
  }

  // Create a marker element for each buy/sell point
  markers.forEach(marker => {
    try {
      // Get pixel coordinates from chart
      const time = marker.time;
      const price = marker.price || marker.order.price;

      // Skip if we can't get proper coordinates
      if (!chart.timeScale || !chart.priceScale) {
        console.warn('Chart scales not available, skipping text marker');
        return;
      }

      // Convert time to pixel coordinates
      const timeScale = chart.timeScale();
      const priceScale = chart.priceScale('right');

      if (!timeScale || !priceScale) {
        console.warn('Time or price scale not available');
        return;
      }

      const timeCoordinate = timeScale.timeToCoordinate(time);
      const priceCoordinate = priceScale.priceToCoordinate(price);

      if (timeCoordinate === null || priceCoordinate === null) {
        console.warn('Could not convert time or price to coordinates');
        return;
      }

      // Create marker element
      const markerEl = document.createElement('div');
      markerEl.className = 'trade-marker';
      markerEl.style.position = 'absolute';
      markerEl.style.left = `${timeCoordinate}px`;
      markerEl.style.top = `${priceCoordinate}px`;
      markerEl.style.transform = 'translate(-50%, -50%)';
      markerEl.style.color = marker.color;
      markerEl.style.fontSize = '12px';
      markerEl.style.fontWeight = 'bold';
      markerEl.style.zIndex = '1000';
      markerEl.textContent = marker.text;

      markerContainer.appendChild(markerEl);
    } catch (error) {
      console.error('Error creating text marker:', error);
    }
  });
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

    // Add CSS for custom markers
    const style = document.createElement('style');
    style.textContent = `
      .trade-marker {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 20px;
        height: 20px;
        border-radius: 50%;
        background-color: white;
        border: 2px solid;
      }
      
      .chart-container {
        position: relative;
        width: 100%;
        margin-bottom: 10px;
        border-radius: 8px;
        overflow: hidden;
      }
      
      .legend-container {
        position: absolute;
        top: 10px;
        right: 10px;
        background-color: ${COLORS.BACKGROUND}E6;
        padding: 5px;
        border-radius: 4px;
        font-size: 11px;
        z-index: 5;
        max-height: 200px;
        overflow-y: auto;
        border: 1px solid ${COLORS.BORDER};
      }
      
      .indicator-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 5px;
        background-color: ${COLORS.BACKGROUND};
        border-bottom: 1px solid ${COLORS.BORDER};
        font-weight: bold;
      }
      
      .legend-item {
        display: flex;
        align-items: center;
        margin-bottom: 5px;
      }
      
      .legend-marker {
        width: 10px;
        height: 10px;
        margin-right: 5px;
        border-radius: 50%;
      }
    `;
    document.head.appendChild(style);
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

      // Debug the data to see its structure
      console.log('Fetched data:', data);

      // Check if data contains indicators and order information
      if (data.indicators) {
        console.log('Indicators found:', data.indicators.length);
      }

      // Check if candles have orders
      if (data.candles && data.candles.length > 0) {
        const ordersCount = data.candles.reduce((count, candle) =>
          count + (candle.orders ? candle.orders.length : 0), 0);
        console.log('Total orders found:', ordersCount);
      }

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
        fontSize: 11, // Smaller font for indicators
      },
      grid: {
        vertLines: { color: COLORS.GRID },
        horzLines: { color: COLORS.GRID },
      },
      rightPriceScale: {
        borderColor: COLORS.BORDER,
        visible: true,
        scaleMargins: {
          top: 0.1,
          bottom: 0.1,
        },
      },
      timeScale: {
        borderColor: COLORS.BORDER,
        visible: showTimeScale,
        timeVisible: true,
      },
    });

    this.additionalCharts.push(chart);
    return chart;
  }

  // Set up window resize handler
  setupResizeHandler() {
    window.addEventListener('resize', () => {
      // First resize the main chart
      if (this.mainChart) {
        const container = this.mainChart.container || document.getElementById('main-chart');
        if (container) {
          const width = container.clientWidth;
          const height = container.clientHeight;
          if (width > 0 && height > 0) {
            this.mainChart.applyOptions({
              width: width,
              height: height,
            });
          }
        }
      }

      // Then resize all additional charts
      this.additionalCharts.forEach(chart => {
        if (chart.container) {
          const width = chart.container.clientWidth;
          const height = chart.container.clientHeight;
          if (width > 0 && height > 0) {
            chart.applyOptions({
              width: width,
              height: height,
            });
          }
        }
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
          // Create marker with explicit positioning
          const marker = {
            time: new Date(candle.time).getTime() / 1000,
            position: order.side === SIDES.BUY ? 'belowBar' : 'aboveBar',
            color: order.side === SIDES.BUY ? COLORS.UP : COLORS.DOWN,
            shape: order.side === SIDES.BUY ? 'arrowUp' : 'arrowDown',
            text: order.side === SIDES.BUY ? 'B' : 'S',
            size: 2, // Slightly larger for visibility
            price: order.price, // Store price for reference
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

    // Debug markers
    console.log('Buy markers:', this.buyMarkers.length);
    console.log('Sell markers:', this.sellMarkers.length);

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

    // Get height (smaller if we have many indicators)
    const equityHeight = data.equityHeight || 150;

    // Create container
    const equityContainer = createElement('div', 'chart-container', graphContainer);
    equityContainer.style.height = `${equityHeight}px`;
    equityContainer.style.marginBottom = '10px';
    equityContainer.style.border = `1px solid ${COLORS.BORDER}`;

    // Add title
    const equityHeader = createElement('div', 'indicator-header', equityContainer);
    equityHeader.textContent = 'Equity Performance';

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
      priceFormat: {
        type: 'price',
        precision: 2,
        minMove: 0.01,
      },
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

    // Add vertical lines - check for the correct method first
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
      priceFormat: {
        type: 'price',
        precision: 4,
        minMove: 0.0001,
      },
    });
    assetSeries.setData(assetData);

    // Add to legend
    addLegendItem(legend, `Position (${data.asset}/${data.quote})`, COLORS.ASSET);
  }

  // Sync a chart's time scale with the main chart
  syncChartWithMain(chart) {
    try {
      // Make sure main chart exists and has a timeScale method
      if (!this.mainChart || !this.methodExists(this.mainChart, 'timeScale')) {
        console.error('Cannot sync chart: main chart does not exist or has no timeScale method');
        return;
      }

      const mainTimeScale = this.mainChart.timeScale();
      const chartTimeScale = chart.timeScale();

      // First, set the visible range to match main chart's current range
      if (this.methodExists(mainTimeScale, 'getVisibleRange') &&
        this.methodExists(chartTimeScale, 'setVisibleRange')) {
        const visibleRange = mainTimeScale.getVisibleRange();
        if (visibleRange) {
          chartTimeScale.setVisibleRange(visibleRange);
        }
      }

      // Then subscribe to changes
      if (this.methodExists(mainTimeScale, 'subscribeVisibleTimeRangeChange')) {
        mainTimeScale.subscribeVisibleTimeRangeChange(timeRange => {
          if (timeRange && this.methodExists(chartTimeScale, 'setVisibleRange')) {
            chartTimeScale.setVisibleRange(timeRange);
          }
        });
      }

      // Sync crosshair movement
      if (this.mainChart.subscribeCrosshairMove && chart.setCrosshairPosition) {
        this.mainChart.subscribeCrosshairMove(param => {
          if (param && param.time && param.point) {
            chart.setCrosshairPosition(param.time, param.point.x);
          }
        });
      }
    } catch (error) {
      console.error('Error synchronizing charts:', error);
    }
  }

  // Add indicators to charts
  addIndicators(data, graphContainer, mainLegend) {
    if (!data.indicators || data.indicators.length === 0) {
      console.log('No indicators found in data');
      return;
    }

    console.log(`Processing ${data.indicators.length} indicators`);

    // Group indicators
    const overlayIndicators = data.indicators.filter(ind => ind.overlay);
    const standaloneIndicators = data.indicators.filter(ind => !ind.overlay);

    console.log(`Found ${overlayIndicators.length} overlay and ${standaloneIndicators.length} standalone indicators`);

    // Add overlay indicators to main chart
    this.addOverlayIndicators(overlayIndicators, mainLegend);

    // Add standalone indicators as separate charts
    this.addStandaloneIndicators(standaloneIndicators, graphContainer);
  }

  // Add overlay indicators to the main chart
  addOverlayIndicators(indicators, legend) {
    indicators.forEach(indicator => {
      console.log(`Adding overlay indicator: ${indicator.name}`);

      if (!indicator.metrics || indicator.metrics.length === 0) {
        console.log(`No metrics found for indicator ${indicator.name}`);
        return;
      }

      indicator.metrics.forEach(metric => {
        // Make sure metric.time and metric.value exist and have the same length
        if (!metric.time || !metric.value || metric.time.length !== metric.value.length) {
          console.error(`Invalid metric data for ${indicator.name}:`, metric);
          return;
        }

        // Format indicator data
        const indicatorData = metric.time.map((time, i) => ({
          time: new Date(time).getTime() / 1000,
          value: metric.value[i]
        }));

        // Add indicator series
        try {
          const indicatorSeries = this.mainChart.addLineSeries({
            color: metric.color || COLORS.DEFAULT_INDICATOR,
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
          addLegendItem(legend, name, metric.color || COLORS.DEFAULT_INDICATOR);
        } catch (error) {
          console.error(`Failed to add overlay indicator ${indicator.name}:`, error);
        }
      });
    });
  }

  // Add standalone indicators as separate charts
  addStandaloneIndicators(indicators, graphContainer) {
    console.log(`Setting up ${indicators.length} standalone indicators`);

    indicators.forEach((indicator, index) => {
      try {
        console.log(`Creating container for indicator: ${indicator.name}`);

        // Create container with specific height to ensure visibility
        const indicatorContainer = createElement('div', 'chart-container', graphContainer);
        indicatorContainer.style.height = '150px';
        indicatorContainer.style.minHeight = '150px'; // Force minimum height
        indicatorContainer.style.margin = '10px 0'; // Add margin for visual separation
        indicatorContainer.style.position = 'relative'; // Ensure positioning context for legend

        // Add a visible border for better separation
        indicatorContainer.style.border = `1px solid ${COLORS.BORDER}`;
        indicatorContainer.style.borderRadius = '8px';
        indicatorContainer.style.overflow = 'visible'; // Ensure content isn't clipped

        // Add title header
        const indicatorHeader = createElement('div', 'indicator-header', indicatorContainer);
        indicatorHeader.textContent = indicator.name;

        // Create chart
        const showTimeScale = index === indicators.length - 1; // Only show time on last indicator
        const indicatorChart = this.createSecondaryChart(indicatorContainer, showTimeScale);
        indicatorChart.container = indicatorContainer;

        // Force chart dimensions to match container (account for header)
        const chartHeight = indicatorContainer.clientHeight - 30; // Subtract header height
        indicatorChart.applyOptions({
          width: indicatorContainer.clientWidth,
          height: chartHeight > 0 ? chartHeight : 120
        });

        // Create legend
        const indicatorLegend = createElement('div', 'legend-container', indicatorContainer);

        // Check if metric data exists
        if (!indicator.metrics || indicator.metrics.length === 0) {
          console.error(`No metrics found for indicator ${indicator.name}`);
          indicatorHeader.innerHTML += ' <span style="color:red">(No data available)</span>';
          return;
        }

        // Sync with main chart - do this BEFORE adding series
        this.syncChartWithMain(indicatorChart);

        // Add each metric as a series
        indicator.metrics.forEach(metric => {
          if (!metric.time || !metric.value || metric.time.length === 0) {
            console.error(`Invalid metric data for ${indicator.name}:`, metric);
            return;
          }

          console.log(`Adding ${metric.name || 'unnamed'} metric to ${indicator.name} (${metric.time.length} data points)`);

          // Debug some data points
          const samplePoints = Math.min(5, metric.time.length);
          console.log(`Sample data points for ${indicator.name} - ${metric.name || 'unnamed'}:`);
          for (let i = 0; i < samplePoints; i++) {
            console.log(`  Time: ${metric.time[i]}, Value: ${metric.value[i]}`);
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
              indicatorSeries = indicatorChart.addHistogramSeries({
                color: metric.color || COLORS.DEFAULT_INDICATOR,
                priceLineVisible: false,
                lastValueVisible: true, // Show current value
                priceFormat: {
                  type: 'price',
                  precision: 5, // More precision for indicator values
                  minMove: 0.00001,
                },
                base: 0, // This is important for MACD histograms
              });
            } else {
              indicatorSeries = indicatorChart.addLineSeries({
                color: metric.color || COLORS.DEFAULT_INDICATOR,
                lineWidth: 1.5, // Slightly thicker for visibility
                priceLineVisible: false,
                lastValueVisible: true, // Show current value
                priceFormat: {
                  type: 'price',
                  precision: 5, // More precision for indicator values
                  minMove: 0.00001,
                }
              });
            }

            indicatorSeries.setData(indicatorData);

            // Add to legend
            const name = metric.name || indicator.name;
            addLegendItem(indicatorLegend, name, metric.color || COLORS.DEFAULT_INDICATOR);

            console.log(`Added series for ${name} with ${indicatorData.length} points`);
          } catch (seriesError) {
            console.error(`Failed to add series for ${metric.name || 'unnamed'}:`, seriesError);
          }
        });

        // Fit content to the chart
        if (this.methodExists(indicatorChart.timeScale(), 'fitContent')) {
          indicatorChart.timeScale().fitContent();
        }

        // Make sure price scale is adjusted to show all data
        try {
          const rightPriceScale = indicatorChart.priceScale('right');
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

      } catch (error) {
        console.error(`Failed to add standalone indicator ${indicator.name}:`, error);
      }
    });
  }

  // Render all charts
  renderCharts(data) {
    try {
      // Clear existing content and charts array
      const graphContainer = document.getElementById('graph');
      graphContainer.innerHTML = '';
      this.additionalCharts = [];

      // Count standalone indicators to adjust layout
      const standaloneIndicators = data.indicators ? data.indicators.filter(ind => !ind.overlay) : [];
      const indicatorCount = standaloneIndicators.length;

      // Adjust main chart height based on number of indicators
      // If we have indicators, make the main chart smaller to give room
      let mainChartHeight = 400;
      if (indicatorCount > 0) {
        // Allocate space based on indicator count
        const totalHeight = 600; // Total available height
        mainChartHeight = Math.floor(totalHeight * 0.6); // 60% for main chart
        console.log(`Adjusting chart layout for ${indicatorCount} indicators. Main chart height: ${mainChartHeight}px`);
      }

      // Create main chart container
      const mainChartContainer = createElement('div', 'chart-container', graphContainer);
      mainChartContainer.id = 'main-chart';
      mainChartContainer.style.height = `${mainChartHeight}px`;
      mainChartContainer.style.border = `1px solid ${COLORS.BORDER}`;
      mainChartContainer.style.borderRadius = '8px';
      mainChartContainer.style.marginBottom = '15px';

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

      // Store a reference to candle data for alternative marker implementations
      this.candleData = candleData;

      // Check if native marker support is available
      const hasMarkerSupport = testMarkerSupport(this.mainChart);
      console.log("Native marker support available:", hasMarkerSupport);

      // Add markers using the appropriate method
      if (markers.length > 0) {
        console.log(`Adding ${markers.length} markers to chart`);

        let nativeMarkersWorked = false;
        if (hasMarkerSupport && this.methodExists(this.candleSeries, 'setMarkers')) {
          try {
            this.candleSeries.setMarkers(markers);
            console.log("Using native setMarkers method");
            nativeMarkersWorked = true;
          } catch (error) {
            console.error("Native setMarkers method failed:", error);
          }
        }

        // If native markers didn't work, use our custom approach
        if (!nativeMarkersWorked) {
          console.log("Falling back to custom marker implementation");
          addCustomMarkers(this.mainChart, this.candleSeries, markers);
        }

        // Add legend items for markers
        if (this.buyMarkers.length > 0) {
          addLegendItem(legendContainer, 'Buy Points', COLORS.UP);
        }
        if (this.sellMarkers.length > 0) {
          addLegendItem(legendContainer, 'Sell Points', COLORS.DOWN);
        }
      } else {
        console.log('No markers to add');
      }

      // Setup tooltip
      this.setupTooltip();

      // Create equity chart - use smaller height if we have indicators
      if (indicatorCount > 0) {
        data.equityHeight = 120; // Smaller equity chart when we have indicators
      }

      // Create equity chart
      this.createEquityChart(data, graphContainer);

      // Add indicators - call this AFTER creating the main chart and equity chart
      this.addIndicators(data, graphContainer, legendContainer);

      // Fit content
      if (this.methodExists(this.mainChart.timeScale(), 'fitContent')) {
        this.mainChart.timeScale().fitContent();
      }

      // Make sure all additional charts are correctly sized and fitted
      this.additionalCharts.forEach(chart => {
        if (chart.container && chart.container.clientWidth && chart.container.clientHeight) {
          chart.applyOptions({
            width: chart.container.clientWidth,
            height: chart.container.clientHeight
          });

          // Force fit content for each chart
          if (this.methodExists(chart.timeScale(), 'fitContent')) {
            chart.timeScale().fitContent();
          }

          // Make sure price scale shows all data
          try {
            const rightPriceScale = chart.priceScale('right');
            if (rightPriceScale) {
              rightPriceScale.applyOptions({
                autoScale: true,
                scaleMargins: {
                  top: 0.1,
                  bottom: 0.1,
                }
              });
            }
          } catch (error) {
            console.error('Failed to adjust price scale:', error);
          }
        }
      });

      console.log(`Chart rendering complete with ${this.additionalCharts.length} additional charts`);

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

  // Safely check if a method exists on an object
  methodExists(obj, methodName) {
    return obj && typeof obj[methodName] === 'function';
  }
}
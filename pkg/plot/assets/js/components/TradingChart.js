/**
 * TradingChart class - Main chart component
 */

import { getCurrentThemeColors } from '../config/theme.js';
import { formatCandleData, processOrderMarkers, fetchChartData } from '../services/dataService.js';
import { createMainChart, createSecondaryChart, setupResizeHandler, methodExists } from './chartCreator.js';
import { addCustomMarkers, testMarkerSupport } from '../utils/markers.js';
import { createTooltip, setupTooltip } from './tooltip.js';
import { createEquityChart } from './equityChart.js';
import { addIndicators } from './indicators.js';
import { applyChartStyles } from '../styles/chartStyles.js';
import { createElement } from '../utils/helpers.js';
import { initWebSocket, closeWebSocket, registerMessageHandler, unregisterMessageHandler, isWebSocketConnected } from '../services/websocketService.js';

/**
 * TradingChart class for chart creation and management
 */
export class TradingChart {
  /**
   * Constructor
   */
  constructor() {
    this.additionalCharts = [];
    this.mainChart = null;
    this.candleSeries = null;
    this.tooltip = null;
    this.pair = '';
    this.buyMarkers = [];
    this.sellMarkers = [];
    this.currentTheme = document.documentElement.getAttribute('data-theme') || 'light';
    this.colors = getCurrentThemeColors();
    this.wsHandlers = {};
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
        
        // If this is a real-time update (not a complete candle), ensure the chart stays scrolled to the right
        if (!candle.complete) {
          // Ensure the chart stays scrolled to the right to show the latest data
          if (methodExists(this.mainChart, 'timeScale') && 
              methodExists(this.mainChart.timeScale(), 'scrollToRealTime')) {
            this.mainChart.timeScale().scrollToRealTime();
          } else if (methodExists(this.mainChart, 'timeScale') && 
                    methodExists(this.mainChart.timeScale(), 'scrollToPosition')) {
            // Alternative approach if scrollToRealTime is not available
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
          
          // Fallback to custom markers
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
   * Update chart theme colors
   * @param {string} theme - Theme name ('light' or 'dark')
   */
  updateChartTheme(theme) {
    this.currentTheme = theme;
    this.colors = getCurrentThemeColors();
    
    // Apply updated styles
    applyChartStyles(this.colors);

    // Close existing WebSocket connection
    closeWebSocket();
    
    // Setup WebSocket handlers again with new theme
    this.setupWebSocketHandlers();
    
    // Re-fetch data through WebSocket
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
      mainChartContainer.style.border = `1px solid ${this.colors.BORDER}`;
      mainChartContainer.style.borderRadius = '8px';
      mainChartContainer.style.marginBottom = '15px';

      // Create legend container
      const legendContainer = createElement('div', 'legend-container', mainChartContainer);

      // Initialize main chart
      this.mainChart = createMainChart(mainChartContainer, this.colors, LightweightCharts);

      // Store container reference
      this.mainChart.container = mainChartContainer;

      // Setup resize handler
      setupResizeHandler(this.mainChart, this.additionalCharts);

      // Format candle data
      const candleData = formatCandleData(data.candles);

      // Add candlestick series with proper error handling
      try {
        // Check for method existence
        if (methodExists(this.mainChart, 'addCandlestickSeries')) {
          this.candleSeries = this.mainChart.addCandlestickSeries({
            upColor: this.colors.UP,
            downColor: this.colors.DOWN,
            borderVisible: false,
            wickUpColor: this.colors.UP,
            wickDownColor: this.colors.DOWN,
          });
          console.log("Using addCandlestickSeries method");
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
          console.log("Using createPriceSeries method");
        }
        else if (methodExists(this.mainChart, 'addBarSeries')) {
          this.candleSeries = this.mainChart.addBarSeries({
            upColor: this.colors.UP,
            downColor: this.colors.DOWN,
            thinBars: false,
          });
          console.log("Using addBarSeries method as fallback");
        }
        else if (methodExists(this.mainChart, 'addAreaSeries')) {
          // Last resort fallback to area series
          this.candleSeries = this.mainChart.addAreaSeries({
            topColor: this.colors.UP,
            bottomColor: this.colors.UP.replace('1)', '0.1)'),
            lineColor: this.colors.UP,
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
      if (methodExists(this.candleSeries, 'setData') &&
        this.candleSeries.constructor.name !== 'AreaSeries') {
        this.candleSeries.setData(candleData);
      }

      // Process markers
      const { allMarkers, buyMarkers, sellMarkers } = processOrderMarkers(data.candles, this.colors);
      this.buyMarkers = buyMarkers;
      this.sellMarkers = sellMarkers;

      // Check if native marker support is available
      const hasMarkerSupport = testMarkerSupport(this.mainChart);
      console.log("Native marker support available:", hasMarkerSupport);

      // Add markers using the appropriate method
      if (allMarkers.length > 0) {
        console.log(`Adding ${allMarkers.length} markers to chart`);

        let nativeMarkersWorked = false;
        if (hasMarkerSupport && methodExists(this.candleSeries, 'setMarkers')) {
          try {
            this.candleSeries.setMarkers(allMarkers);
            console.log("Using native setMarkers method");
            nativeMarkersWorked = true;
          } catch (error) {
            console.error("Native setMarkers method failed:", error);
          }
        }

        // If native markers didn't work, use our custom approach
        if (!nativeMarkersWorked) {
          console.log("Falling back to custom marker implementation");
          addCustomMarkers(this.mainChart, this.candleSeries, allMarkers, this.colors);
        }
      } else {
        console.log('No markers to add');
      }

      // Setup tooltip
      setupTooltip(this.mainChart, this.tooltip, this.buyMarkers, this.sellMarkers);

      // Create equity chart - use smaller height if we have indicators
      if (indicatorCount > 0) {
        data.equityHeight = 120; // Smaller equity chart when we have indicators
      }

      // Create equity chart
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

      // Add indicators
      const indicatorCharts = addIndicators(
        data, 
        graphContainer, 
        this.mainChart, 
        this.colors, 
        (container, showTimeScale) => createSecondaryChart(container, this.colors, LightweightCharts, showTimeScale)
      );
      
      this.additionalCharts.push(...indicatorCharts);

      // Fit content
      if (methodExists(this.mainChart, 'timeScale') && 
          methodExists(this.mainChart.timeScale(), 'fitContent')) {
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
          if (methodExists(chart, 'timeScale') && 
              methodExists(chart.timeScale(), 'fitContent')) {
            chart.timeScale().fitContent();
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
    this.additionalCharts = [];
  }
}

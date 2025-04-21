/**
 * Chart creation and configuration utilities
 */

/**
 * Create and configure the main chart
 * @param {HTMLElement} container - Container element
 * @param {Object} colors - Theme colors
 * @param {Object} LightweightCharts - Chart library
 * @returns {Object} - Chart instance
 */
export function createMainChart(container, colors, LightweightCharts, showHorLine = true, showVertLine = true) {
  if (typeof LightweightCharts === 'undefined' || typeof LightweightCharts.createChart !== 'function') {
    throw new Error('LightweightCharts library not properly initialized');
  }

  return LightweightCharts.createChart(container, {
    width: container.clientWidth,
    height: container.clientHeight,
    layout: {
      backgroundColor: colors.BACKGROUND,
      textColor: colors.TEXT,
    },
    grid: {
      vertLines: { color: colors.GRID, visible: showHorLine },
      horzLines: { color: colors.GRID, visible: showVertLine },
    },
    crosshair: {
      mode: LightweightCharts.CrosshairMode.Normal,
    },
    rightPriceScale: {
      borderColor: colors.BORDER,
    },
    timeScale: {
      borderColor: colors.BORDER,
      timeVisible: true,
    },
  });
}

/**
 * Create a secondary chart (for indicators, equity, etc.)
 * @param {HTMLElement} container - Container element
 * @param {Object} colors - Theme colors
 * @param {Object} LightweightCharts - Chart library
 * @param {boolean} showTimeScale - Whether to show time scale
 * @returns {Object} - Chart instance
 */
export function createSecondaryChart(container, colors, LightweightCharts, showTimeScale = false) {
  if (typeof LightweightCharts === 'undefined' || typeof LightweightCharts.createChart !== 'function') {
    throw new Error('LightweightCharts library not properly initialized');
  }

  return LightweightCharts.createChart(container, {
    width: container.clientWidth,
    height: container.clientHeight,
    layout: {
      backgroundColor: colors.BACKGROUND,
      textColor: colors.TEXT,
      fontSize: 11, // Smaller font for indicators
    },
    grid: {
      vertLines: { color: colors.GRID },
      horzLines: { color: colors.GRID },
    },
    rightPriceScale: {
      borderColor: colors.BORDER,
      visible: true,
      scaleMargins: {
        top: 0.1,
        bottom: 0.1,
      },
    },
    timeScale: {
      borderColor: colors.BORDER,
      visible: showTimeScale,
      timeVisible: true,
    },
  });
}

/**
 * Sync a chart's time scale with the main chart
 * @param {Object} mainChart - Main chart instance
 * @param {Object} chart - Chart to sync
 */
export function syncChartWithMain(mainChart, chart) {
  try {
    // Make sure main chart exists and has a timeScale method
    if (!mainChart || typeof mainChart.timeScale !== 'function') {
      console.error('Cannot sync chart: main chart does not exist or has no timeScale method');
      return;
    }

    const mainTimeScale = mainChart.timeScale();
    const chartTimeScale = chart.timeScale();

    // First, set the visible range to match main chart's current range
    if (typeof mainTimeScale.getVisibleRange === 'function' &&
      typeof chartTimeScale.setVisibleRange === 'function') {
      const visibleRange = mainTimeScale.getVisibleRange();
      if (visibleRange) {
        chartTimeScale.setVisibleRange(visibleRange);
      }
    }

    // Then subscribe to changes
    if (typeof mainTimeScale.subscribeVisibleTimeRangeChange === 'function') {
      mainTimeScale.subscribeVisibleTimeRangeChange(timeRange => {
        if (timeRange && typeof chartTimeScale.setVisibleRange === 'function') {
          chartTimeScale.setVisibleRange(timeRange);
        }
      });
    }

    // Sync crosshair movement
    if (mainChart.subscribeCrosshairMove && chart.setCrosshairPosition) {
      mainChart.subscribeCrosshairMove(param => {
        if (param && param.time && param.point) {
          chart.setCrosshairPosition(param.time, param.point.x);
        }
      });
    }
  } catch (error) {
    console.error('Error synchronizing charts:', error);
  }
}

/**
 * Set up window resize handler for charts
 * @param {Object} mainChart - Main chart instance
 * @param {Array} additionalCharts - Additional chart instances
 */
export function setupResizeHandler(mainChart, additionalCharts) {
  window.addEventListener('resize', () => {
    // First resize the main chart
    if (mainChart) {
      const container = mainChart.container || document.getElementById('main-chart');
      if (container) {
        const width = container.clientWidth;
        const height = container.clientHeight;
        if (width > 0 && height > 0) {
          mainChart.applyOptions({
            width: width,
            height: height,
          });
        }
      }
    }

    // Then resize all additional charts
    additionalCharts.forEach(chart => {
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

/**
 * Safely check if a method exists on an object
 * @param {Object} obj - Object to check
 * @param {string} methodName - Method name to check
 * @returns {boolean} - Whether method exists
 */
export function methodExists(obj, methodName) {
  return obj && typeof obj[methodName] === 'function';
}

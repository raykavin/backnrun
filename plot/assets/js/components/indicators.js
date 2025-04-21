/**
 * Indicators component for handling technical indicators
 */

import { addLegendItem, createElement } from '../utils/helpers.js';
import { syncChartWithMain, methodExists } from './chartCreator.js';

/**
 * Add indicators to charts
 * @param {Object} data - Chart data
 * @param {HTMLElement} graphContainer - Graph container
 * @param {Object} mainChart - Main chart instance
 * @param {Object} colors - Theme colors
 * @param {Function} createSecondaryChart - Function to create secondary chart
 * @returns {Array} - Array of created indicator charts
 */
export function addIndicators(data, graphContainer, mainChart, colors, createSecondaryChart) {
  if (!data.indicators || data.indicators.length === 0) {
    console.log('No indicators found in data');
    return [];
  }

  console.log(`Processing ${data.indicators.length} indicators`);

  // Group indicators
  const overlayIndicators = data.indicators.filter(ind => ind.overlay);
  const standaloneIndicators = data.indicators.filter(ind => !ind.overlay);

  console.log(`Found ${overlayIndicators.length} overlay and ${standaloneIndicators.length} standalone indicators`);

  // Add overlay indicators to main chart
  addOverlayIndicators(overlayIndicators, mainChart, colors);

  // Add standalone indicators as separate charts
  const indicatorCharts = addStandaloneIndicators(standaloneIndicators, graphContainer, mainChart, colors, createSecondaryChart);

  return indicatorCharts;
}

/**
 * Add overlay indicators to the main chart
 * @param {Array} indicators - Overlay indicators
 * @param {Object} mainChart - Main chart instance
 * @param {Object} colors - Theme colors
 */
function addOverlayIndicators(indicators, mainChart, colors) {
  // Get or create legend
  let legend = document.querySelector('#main-chart .legend-container');
  if (!legend) {
    const mainChartContainer = document.getElementById('main-chart');
    if (mainChartContainer) {
      legend = createElement('div', 'legend-container', mainChartContainer);
    } else {
      console.error('Main chart container not found, cannot add legend');
      return;
    }
  }

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
        const indicatorSeries = mainChart.addLineSeries({
          color: metric.color || colors.DEFAULT_INDICATOR,
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
        addLegendItem(legend, name, metric.color || colors.DEFAULT_INDICATOR);
      } catch (error) {
        console.error(`Failed to add overlay indicator ${indicator.name}:`, error);
      }
    });
  });
}

/**
 * Add standalone indicators as separate charts
 * @param {Array} indicators - Standalone indicators
 * @param {HTMLElement} graphContainer - Graph container
 * @param {Object} mainChart - Main chart instance
 * @param {Object} colors - Theme colors
 * @param {Function} createSecondaryChart - Function to create secondary chart
 * @returns {Array} - Array of created indicator charts
 */
function addStandaloneIndicators(indicators, graphContainer, mainChart, colors, createSecondaryChart) {
  console.log(`Setting up ${indicators.length} standalone indicators`);
  const indicatorCharts = [];

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
      indicatorContainer.style.border = `1px solid ${colors.BORDER}`;
      indicatorContainer.style.borderRadius = '8px';
      indicatorContainer.style.overflow = 'visible'; // Ensure content isn't clipped

      // Add title header
      const indicatorHeader = createElement('div', 'indicator-header', indicatorContainer);
      indicatorHeader.textContent = indicator.name;

      // Create chart
      const showTimeScale = index === indicators.length - 1; // Only show time on last indicator
      const indicatorChart = createSecondaryChart(indicatorContainer, showTimeScale);
      indicatorChart.container = indicatorContainer;
      indicatorCharts.push(indicatorChart);

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
      syncChartWithMain(mainChart, indicatorChart);

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
              color: metric.color || colors.DEFAULT_INDICATOR,
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
              color: metric.color || colors.DEFAULT_INDICATOR,
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
          addLegendItem(indicatorLegend, name, metric.color || colors.DEFAULT_INDICATOR);

          console.log(`Added series for ${name} with ${indicatorData.length} points`);
        } catch (seriesError) {
          console.error(`Failed to add series for ${metric.name || 'unnamed'}:`, seriesError);
        }
      });

      // Fit content to the chart
      if (methodExists(indicatorChart, 'timeScale') && 
          methodExists(indicatorChart.timeScale(), 'fitContent')) {
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

  return indicatorCharts;
}

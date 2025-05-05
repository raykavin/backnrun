/**
 * Equity chart component
 */

import { addLegendItem, createElement } from '../utils/Helpers.js';
import { syncChartWithMain } from './ChartCreator.js';
// import { methodExists } from './chartCreator.js';

/**
 * Create equity chart
 * @param {Object} data - Chart data
 * @param {HTMLElement} graphContainer - Graph container
 * @param {Object} mainChart - Main chart instance
 * @param {Object} colors - Theme colors
 * @param {Function} createSecondaryChart - Function to create secondary chart
 * @returns {Object} - Equity chart instance
 */
export function createEquityChart(data, graphContainer, mainChart, colors, createSecondaryChart) {
  if (!data.equity_values || data.equity_values.length === 0) return null;

  // Get height (smaller if we have many indicators)
  const equityHeight = data.equityHeight || 150;

  // Create container
  const equityContainer = createElement('div', 'chart-container', graphContainer);
  equityContainer.style.height = `${equityHeight}px`;
  equityContainer.style.marginBottom = '10px';
  equityContainer.style.border = `1px solid ${colors.BORDER}`;

  // Add title
  const equityHeader = createElement('div', 'indicator-header', equityContainer);
  equityHeader.textContent = 'Equity Performance';

  // Create chart
  const equityChart = createSecondaryChart(equityContainer, false);
  equityChart.container = equityContainer;

  // Sync with main chart
  syncChartWithMain(mainChart, equityChart);

  // Create legend
  const equityLegend = createElement('div', 'legend-container', equityContainer);

  // Format equity data
  const equityData = data.equity_values.map(item => ({
    time: new Date(item.time).getTime() / 1000,
    value: item.value
  }));

  // Add equity series
  const equitySeries = equityChart.addAreaSeries({
    topColor: colors.EQUITY.replace('1)', '0.56)'),
    bottomColor: colors.EQUITY.replace('1)', '0.04)'),
    lineColor: colors.EQUITY,
    lineWidth: 2,
    priceFormat: {
      type: 'price',
      precision: 2,
      minMove: 0.01,
    },
  });
  equitySeries.setData(equityData);

  // Add equity to legend
  addLegendItem(equityLegend, `Equity (${data.quote})`, colors.EQUITY);

  // Handle drawdown if available
  addDrawdownVisualization(data, equityChart, equitySeries, equityLegend, colors);

  // Add asset values if available
  addAssetSeries(data, equityChart, equityLegend, colors);

  return equityChart;
}

/**
 * Add drawdown visualization
 * @param {Object} data - Chart data
 * @param {Object} chart - Chart instance
 * @param {Object} series - Series instance
 * @param {HTMLElement} legend - Legend element
 * @param {Object} colors - Theme colors
 */
function addDrawdownVisualization(data, chart, series, legend, colors) {
  if (!data.max_drawdown) return;

  // Find time range for drawdown
  const startTime = new Date(data.max_drawdown.start).getTime() / 1000;
  const endTime = new Date(data.max_drawdown.end).getTime() / 1000;

  // Add drawdown marker
  const drawdownMarker = {
    time: (startTime + endTime) / 2,
    position: 'aboveBar',
    color: colors.DOWN,
    shape: 'square',
    text: `Drawdown: ${data.max_drawdown.value}%`,
  };

  series.setMarkers([drawdownMarker]);

  // Add vertical lines - check for the correct method first
  if (chart.addVerticalLine) {
    chart.addVerticalLine({
      time: startTime,
      color: colors.DOWN.replace('1)', '0.5)'),
      lineWidth: 1,
      lineStyle: 1, // Dashed
    });

    chart.addVerticalLine({
      time: endTime,
      color: colors.DOWN.replace('1)', '0.5)'),
      lineWidth: 1,
      lineStyle: 1, // Dashed
    });
  }

  // Add to legend
  addLegendItem(legend, `Max Drawdown: ${data.max_drawdown.value}%`, colors.DOWN);
}

/**
 * Add asset value series
 * @param {Object} data - Chart data
 * @param {Object} chart - Chart instance
 * @param {HTMLElement} legend - Legend element
 * @param {Object} colors - Theme colors
 */
function addAssetSeries(data, chart, legend, colors) {
  if (!data.asset_values || data.asset_values.length === 0) return;

  // Format asset data
  const assetData = data.asset_values.map(item => ({
    time: new Date(item.time).getTime() / 1000,
    value: item.value
  }));

  // Add asset series
  const assetSeries = chart.addLineSeries({
    color: colors.ASSET,
    lineWidth: 2,
    priceFormat: {
      type: 'price',
      precision: 4,
      minMove: 0.0001,
    },
  });
  assetSeries.setData(assetData);

  // Add to legend
  addLegendItem(legend, `Position (${data.asset}/${data.quote})`, colors.ASSET);
}

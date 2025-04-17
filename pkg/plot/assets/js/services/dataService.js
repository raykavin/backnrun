/**
 * Service for fetching and processing chart data
 */

import { SIDES, STATUS } from '../config/constants.js';

/**
 * Format candle data for TradingView
 * @param {Array} candles - Raw candle data
 * @returns {Array} - Formatted candle data
 */
export function formatCandleData(candles) {
  return candles.map(candle => ({
    time: new Date(candle.time).getTime() / 1000,
    open: candle.open,
    high: candle.high,
    low: candle.low,
    close: candle.close,
    orders: candle.orders || []
  }));
}

/**
 * Process buy/sell markers from candle data
 * @param {Array} candles - Candle data
 * @param {Object} colors - Theme colors
 * @returns {Object} - Object containing markers and arrays for buy/sell markers
 */
export function processOrderMarkers(candles, colors) {
  const buyMarkers = [];
  const sellMarkers = [];

  candles.forEach(candle => {
    if (!candle.orders) return;

    candle.orders
      .filter(order => order.status === STATUS.FILLED)
      .forEach(order => {
        // Create marker with explicit positioning
        const marker = {
          time: new Date(candle.time).getTime() / 1000,
          position: order.side === SIDES.BUY ? 'belowBar' : 'aboveBar',
          color: order.side === SIDES.BUY ? colors.UP : colors.DOWN,
          shape: order.side === SIDES.BUY ? 'arrowUp' : 'arrowDown',
          text: order.side === SIDES.BUY ? 'B' : 'S',
          size: 2, // Slightly larger for visibility
          price: order.price, // Store price for reference
          id: order.id,
          order: order
        };

        if (order.side === SIDES.BUY) {
          buyMarkers.push(marker);
        } else {
          sellMarkers.push(marker);
        }
      });
  });

  // Debug markers
  console.log('Buy markers:', buyMarkers.length);
  console.log('Sell markers:', sellMarkers.length);

  return {
    allMarkers: [...buyMarkers, ...sellMarkers],
    buyMarkers,
    sellMarkers
  };
}

/**
 * Fetch data from the server
 * @param {string} pair - Trading pair
 * @returns {Promise} - Promise resolving to chart data
 */
export async function fetchChartData(pair) {
  try {
    const response = await fetch("/data?pair=" + pair);
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

    return data;
  } catch (error) {
    console.error('Error fetching chart data:', error);
    throw error;
  }
}

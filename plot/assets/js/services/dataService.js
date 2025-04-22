/**
 * Service for fetching and processing chart data
 */

import { SIDES, STATUS } from '../config/Constants.js';

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
 * Fetch data from the server via WebSocket
 * @param {string} pair - Trading pair
 * @returns {Promise} - Promise resolving to chart data
 */
export async function fetchChartData(pair) {
  return new Promise((resolve, reject) => {
    // This function is now a placeholder since we'll get data via WebSocket
    // The actual data will be received through the WebSocket 'initialData' message
    console.log('Waiting for data via WebSocket for pair:', pair);
    
    // Set a timeout to reject the promise if no data is received
    const timeout = setTimeout(() => {
      reject(new Error('Timeout waiting for WebSocket data'));
    }, 10000); // 10 seconds timeout
    
    // Create a one-time handler for the initialData message
    const initialDataHandler = (data) => {
      clearTimeout(timeout);
      console.log('Received initial data via WebSocket in fetchChartData');
      resolve(data);
      
      // This handler will be removed by the caller after use
    };
    
    // Store the handler in a global variable so it can be accessed by the WebSocket service
    window.pendingInitialDataHandler = initialDataHandler;
  });
}

/**
 * Utilities for handling chart markers
 */

import { createElement } from './helpers.js';

/**
 * Function to check if marker support is available in the chart library
 * @param {Object} chart - Chart instance
 * @returns {boolean} - Whether marker support is available
 */
export function testMarkerSupport(chart) {
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
 * Add custom markers as separate series when native marker support is unavailable
 * @param {Object} chart - Chart instance
 * @param {Object} candleSeries - Candlestick series
 * @param {Array} markers - Markers to add
 */
export function addCustomMarkers(chart, candleSeries, markers, colors) {
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
        color: colors.UP,
        lineWidth: 0,
        pointsVisible: true,
        pointSize: 8,
        pointShape: 'circle',
        pointFill: colors.UP,
        pointBorderColor: colors.UP,
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
          color: colors.UP,
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
        color: colors.DOWN,
        lineWidth: 0,
        pointsVisible: true,
        pointSize: 8,
        pointShape: 'circle',
        pointFill: colors.DOWN,
        pointBorderColor: colors.DOWN,
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
          color: colors.DOWN,
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
 * @param {Object} chart - Chart instance
 * @param {Array} markers - Markers to add
 */
export function createTextMarkers(chart, markers) {
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

/**
 * Marker utilities for chart
 */

/**
 * Test if the chart supports native markers
 * @param {Object} chart - Chart instance
 * @returns {boolean} - True if native marker support is available
 */
export function testMarkerSupport(chart) {
  try {
    // Check if the chart has a method to add markers
    const hasAddMarkerMethod = typeof chart.addMarker === 'function';
    
    // Check if the candlestick series has a method to set markers
    const series = chart.series && chart.series[0];
    const hasSetMarkersMethod = series && typeof series.setMarkers === 'function';
    
    return hasAddMarkerMethod || hasSetMarkersMethod;
  } catch (error) {
    console.error('Error testing marker support:', error);
    return false;
  }
}

/**
 * Add custom markers to the chart
 * @param {Object} chart - Chart instance
 * @param {Object} series - Series instance
 * @param {Array} markers - Array of marker objects
 * @param {Object} colors - Theme colors
 */
export function addCustomMarkers(chart, series, markers, colors) {
  if (!chart || !series || !markers || !markers.length) return;
  
  try {
    // Create a custom marker layer if it doesn't exist
    let markerLayer = document.getElementById('custom-marker-layer');
    if (!markerLayer) {
      const chartContainer = chart.container;
      if (!chartContainer) return;
      
      markerLayer = document.createElement('div');
      markerLayer.id = 'custom-marker-layer';
      markerLayer.style.position = 'absolute';
      markerLayer.style.top = '0';
      markerLayer.style.left = '0';
      markerLayer.style.width = '100%';
      markerLayer.style.height = '100%';
      markerLayer.style.pointerEvents = 'none';
      markerLayer.style.zIndex = '2';
      
      chartContainer.appendChild(markerLayer);
    }
    
    // Clear existing markers
    markerLayer.innerHTML = '';
    
    // Get chart dimensions
    const chartWidth = chart.container.clientWidth;
    const chartHeight = chart.container.clientHeight;
    
    // Add markers
    markers.forEach(marker => {
      try {
        // Get coordinates for the marker
        const x = getXCoordinate(chart, marker.time);
        const y = getYCoordinate(series, marker.price);
        
        if (x !== null && y !== null) {
          // Create marker element
          const markerElement = document.createElement('div');
          markerElement.className = 'custom-marker';
          markerElement.style.position = 'absolute';
          markerElement.style.left = `${x}px`;
          markerElement.style.top = `${y}px`;
          markerElement.style.transform = 'translate(-50%, -50%)';
          markerElement.style.width = '20px';
          markerElement.style.height = '20px';
          markerElement.style.borderRadius = '50%';
          markerElement.style.display = 'flex';
          markerElement.style.alignItems = 'center';
          markerElement.style.justifyContent = 'center';
          markerElement.style.fontSize = '12px';
          markerElement.style.fontWeight = 'bold';
          markerElement.style.color = 'white';
          markerElement.style.zIndex = '3';
          
          // Set marker style based on type
          if (marker.shape === 'arrowUp') {
            markerElement.style.backgroundColor = colors.UP;
            markerElement.innerHTML = '▲';
          } else if (marker.shape === 'arrowDown') {
            markerElement.style.backgroundColor = colors.DOWN;
            markerElement.innerHTML = '▼';
          } else {
            markerElement.style.backgroundColor = marker.color || colors.DEFAULT_INDICATOR;
            markerElement.innerHTML = marker.text || '';
          }
          
          // Add tooltip if marker has text
          if (marker.tooltip) {
            markerElement.title = marker.tooltip;
          }
          
          // Append marker to layer
          markerLayer.appendChild(markerElement);
        }
      } catch (error) {
        console.error('Error adding marker:', error);
      }
    });
    
    // Update markers on chart resize
    const updateMarkers = () => {
      // If chart dimensions have changed, reposition markers
      if (chart.container.clientWidth !== chartWidth || chart.container.clientHeight !== chartHeight) {
        addCustomMarkers(chart, series, markers, colors);
      }
    };
    
    // Add resize event listener
    window.addEventListener('resize', updateMarkers);
    
    // Store the event listener on the chart for cleanup
    chart._markerResizeListener = updateMarkers;
    
  } catch (error) {
    console.error('Error adding custom markers:', error);
  }
}

/**
 * Get X coordinate for a time value
 * @param {Object} chart - Chart instance
 * @param {number} time - Time value
 * @returns {number|null} - X coordinate or null if not found
 */
function getXCoordinate(chart, time) {
  try {
    if (!chart || !chart.timeScale) return null;
    
    const timeScale = chart.timeScale();
    if (!timeScale || typeof timeScale.timeToCoordinate !== 'function') return null;
    
    const coordinate = timeScale.timeToCoordinate(time);
    return coordinate;
  } catch (error) {
    console.error('Error getting X coordinate:', error);
    return null;
  }
}

/**
 * Get Y coordinate for a price value
 * @param {Object} series - Series instance
 * @param {number} price - Price value
 * @returns {number|null} - Y coordinate or null if not found
 */
function getYCoordinate(series, price) {
  try {
    if (!series || !series.priceToCoordinate) return null;
    
    const coordinate = series.priceToCoordinate(price);
    return coordinate;
  } catch (error) {
    console.error('Error getting Y coordinate:', error);
    return null;
  }
}

/**
 * Remove custom markers from the chart
 * @param {Object} chart - Chart instance
 */
export function removeCustomMarkers(chart) {
  if (!chart) return;
  
  try {
    // Remove marker layer
    const markerLayer = document.getElementById('custom-marker-layer');
    if (markerLayer) {
      markerLayer.remove();
    }
    
    // Remove resize event listener
    if (chart._markerResizeListener) {
      window.removeEventListener('resize', chart._markerResizeListener);
      delete chart._markerResizeListener;
    }
  } catch (error) {
    console.error('Error removing custom markers:', error);
  }
}

/**
 * Create a marker object
 * @param {number} time - Time value
 * @param {number} price - Price value
 * @param {string} shape - Marker shape ('arrowUp', 'arrowDown', 'circle', 'square', etc.)
 * @param {string} color - Marker color
 * @param {string} text - Marker text
 * @param {string} tooltip - Marker tooltip
 * @returns {Object} - Marker object
 */
export function createMarker(time, price, shape, color, text, tooltip) {
  return {
    time,
    price,
    shape,
    color,
    text,
    tooltip
  };
}

/**
 * Create a buy marker
 * @param {number} time - Time value
 * @param {number} price - Price value
 * @param {string} tooltip - Marker tooltip
 * @returns {Object} - Buy marker object
 */
export function createBuyMarker(time, price, tooltip) {
  return createMarker(time, price, 'arrowUp', '#4CAF50', '▲', tooltip);
}

/**
 * Create a sell marker
 * @param {number} time - Time value
 * @param {number} price - Price value
 * @param {string} tooltip - Marker tooltip
 * @returns {Object} - Sell marker object
 */
export function createSellMarker(time, price, tooltip) {
  return createMarker(time, price, 'arrowDown', '#F44336', '▼', tooltip);
}

/**
 * Create a custom marker
 * @param {number} time - Time value
 * @param {number} price - Price value
 * @param {string} text - Marker text
 * @param {string} color - Marker color
 * @param {string} tooltip - Marker tooltip
 * @returns {Object} - Custom marker object
 */
export function createCustomMarker(time, price, text, color, tooltip) {
  return createMarker(time, price, 'circle', color, text, tooltip);
}

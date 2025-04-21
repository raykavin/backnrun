/**
 * Chart styles for the trading dashboard
 */

/**
 * Apply chart styles to the document
 * @param {Object} colors - Theme colors
 */
export function applyChartStyles(colors) {
  // Check if styles already exist
  const existingStyles = document.getElementById('chart-styles');
  if (existingStyles) {
    existingStyles.remove();
  }

  // Create style element
  const style = document.createElement('style');
  style.id = 'chart-styles';
  
  // Define CSS
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
      background-color: ${colors.BACKGROUND}E6;
      padding: 5px;
      border-radius: 4px;
      font-size: 11px;
      z-index: 5;
      max-height: 200px;
      overflow-y: auto;
      border: 1px solid ${colors.BORDER};
    }
    
    .indicator-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 5px;
      background-color: ${colors.BACKGROUND};
      border-bottom: 1px solid ${colors.BORDER};
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

    .tooltip {
      position: absolute;
      display: none;
      padding: 10px;
      background-color: ${colors.BACKGROUND}F5;
      border: none;
      border-radius: 8px;
      font-size: 12px;
      z-index: 100;
      pointer-events: none;
      box-shadow: 0 4px 15px rgba(0, 0, 0, 0.15);
      color: ${colors.TEXT};
      transition: background-color 0.3s ease, color 0.3s ease, box-shadow 0.3s ease;
      max-width: 250px;
    }
  `;
  
  // Add to document
  document.head.appendChild(style);
}

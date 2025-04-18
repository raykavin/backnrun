/**
 * Main entry point for the trading dashboard
 */

import { TradingChart } from './components/TradingChart.js';
import { ManualOrderForm } from './components/ManualOrderForm.js';
import { ChartDrawingTools } from './components/ChartDrawingTools.js';
import { getCurrentThemeColors } from './config/theme.js';
import { closeWebSocket } from './services/websocketService.js';

// Initialize when DOM is ready
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
      
      // Clean up previous chart instance when navigating away
      window.addEventListener('beforeunload', () => {
        if (window.tradingChart) {
          window.tradingChart.destroy();
        }
      });

      // Initialize manual order form
      initializeOrderForm();

      // Initialize drawing tools
      initializeDrawingTools(chart.mainChart);

      // Set up theme toggle handler
      setupThemeToggle();
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

  /**
   * Initialize the manual order form
   */
  function initializeOrderForm() {
    const orderFormContainer = document.getElementById('manual-order-container');
    if (!orderFormContainer) return;

    // Get current pair from URL or use default
    const params = new URLSearchParams(window.location.search);
    const currentPair = params.get("pair") || "BTC/USDT";

    // Create order form
    const orderForm = new ManualOrderForm('manual-order-container', {
      defaultPair: currentPair,
      onSubmit: handleOrderSubmit
    });

    // Make order form accessible globally
    window.manualOrderForm = orderForm;

      // Add event listener to update order form when pair changes
      const pairButtons = document.querySelectorAll('.pair-btn');
      pairButtons.forEach(btn => {
        btn.addEventListener('click', function(e) {
          const pair = this.textContent.trim();
          if (window.manualOrderForm) {
            window.manualOrderForm.updatePair(pair);
          }
          
          // Clean up previous WebSocket connection
          if (window.tradingChart) {
            window.tradingChart.destroy();
          }
        });
      });
  }

  /**
   * Handle order submission
   * @param {Object} orderData - Order data
   */
  function handleOrderSubmit(orderData) {
    console.log('Order submitted:', orderData);
    
    // In a real app, you would send this to your backend
    // For now, we'll just add it to the recent trades list
    addOrderToRecentTrades(orderData);
  }

  /**
   * Add an order to the recent trades list
   * @param {Object} orderData - Order data
   */
  function addOrderToRecentTrades(orderData) {
    const tradeList = document.querySelector('.trade-list');
    if (!tradeList) return;

    // Create trade item
    const tradeItem = document.createElement('div');
    tradeItem.className = 'trade-item';

    // Format current time
    const now = new Date();
    const timeStr = `Today, ${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}`;

    // Create trade item content
    tradeItem.innerHTML = `
      <div>
        <div class="trade-pair">${orderData.pair}</div>
        <div class="trade-time">${timeStr}</div>
      </div>
      <div class="trade-side ${orderData.side}">${orderData.side.toUpperCase()}</div>
      <div>
        <div class="trade-price">$${orderData.price ? orderData.price.toLocaleString() : 'Market'}</div>
        <div class="trade-profit">Pending</div>
      </div>
    `;

    // Add to the beginning of the list
    tradeList.insertBefore(tradeItem, tradeList.firstChild);

    // Remove the last item if there are more than 5
    if (tradeList.children.length > 5) {
      tradeList.removeChild(tradeList.lastChild);
    }
  }

  /**
   * Initialize drawing tools
   * @param {Object} chart - Chart instance
   */
  function initializeDrawingTools(chart) {
    if (!chart) return;

    // Create drawing tools container
    const drawingToolsContainer = document.getElementById('drawing-tools-container');
    if (!drawingToolsContainer) return;

    // Create drawing tools UI
    createDrawingToolsUI(drawingToolsContainer);

    // Initialize drawing tools
    const drawingTools = new ChartDrawingTools(chart, {
      container: document.getElementById('main-chart'),
      onDrawingComplete: (drawing) => {
        console.log('Drawing completed:', drawing);
      },
      onDrawingRemoved: (drawing) => {
        console.log('Drawing removed:', drawing);
      },
      onToolChange: (tool) => {
        updateActiveToolButton(tool);
      }
    });

    // Make drawing tools accessible globally
    window.drawingTools = drawingTools;

    // Set up toggle drawing button
    const toggleDrawingBtn = document.getElementById('toggle-drawing');
    if (toggleDrawingBtn) {
      toggleDrawingBtn.addEventListener('click', function() {
        const isVisible = drawingToolsContainer.style.display === 'block';
        drawingToolsContainer.style.display = isVisible ? 'none' : 'block';
        
        // Deactivate drawing tool when hiding the container
        if (isVisible && window.drawingTools) {
          window.drawingTools.setTool(null);
        }
      });
    }
  }

  /**
   * Create drawing tools UI
   * @param {HTMLElement} container - Container element
   */
  function createDrawingToolsUI(container) {
    // Clear container
    container.innerHTML = '';

    // Create header
    const header = document.createElement('div');
    header.className = 'drawing-tools-header';
    header.innerHTML = `
      <div class="drawing-tools-title">Drawing Tools</div>
      <div class="drawing-tools-close"><i class="fas fa-times"></i></div>
    `;
    container.appendChild(header);

    // Add close button event listener
    const closeBtn = header.querySelector('.drawing-tools-close');
    closeBtn.addEventListener('click', function() {
      container.style.display = 'none';
      
      // Deactivate drawing tool
      if (window.drawingTools) {
        window.drawingTools.setTool(null);
      }
    });

    // Create tools grid
    const toolsGrid = document.createElement('div');
    toolsGrid.className = 'drawing-tools-grid';
    container.appendChild(toolsGrid);

    // Add drawing tools
    const tools = [
      { id: 'line', icon: 'fa-slash', tooltip: 'Line' },
      { id: 'horizontalLine', icon: 'fa-minus', tooltip: 'Horizontal Line' },
      { id: 'verticalLine', icon: 'fa-grip-lines-vertical', tooltip: 'Vertical Line' },
      { id: 'rectangle', icon: 'fa-square', tooltip: 'Rectangle' },
      { id: 'circle', icon: 'fa-circle', tooltip: 'Circle' },
      { id: 'triangle', icon: 'fa-play', tooltip: 'Triangle' },
      { id: 'text', icon: 'fa-font', tooltip: 'Text' },
      { id: 'fibonacciRetracement', icon: 'fa-wave-square', tooltip: 'Fibonacci' }
    ];

    tools.forEach(tool => {
      const toolBtn = document.createElement('div');
      toolBtn.className = 'drawing-tool-btn';
      toolBtn.dataset.tool = tool.id;
      toolBtn.innerHTML = `<i class="fas ${tool.icon}"></i>`;
      toolBtn.title = tool.tooltip;
      
      toolBtn.addEventListener('click', function() {
        if (window.drawingTools) {
          const isActive = this.classList.contains('active');
          window.drawingTools.setTool(isActive ? null : tool.id);
        }
      });
      
      toolsGrid.appendChild(toolBtn);
    });

    // Create drawing properties section
    const propertiesSection = document.createElement('div');
    propertiesSection.className = 'drawing-properties';
    container.appendChild(propertiesSection);

    // Add color picker
    const colorProperty = document.createElement('div');
    colorProperty.className = 'drawing-property';
    colorProperty.innerHTML = `
      <div class="drawing-property-label">Color</div>
      <div class="color-picker-container">
        <div class="color-option active" style="background-color: #2962FF;" data-color="#2962FF"></div>
        <div class="color-option" style="background-color: #FF6B6B;" data-color="#FF6B6B"></div>
        <div class="color-option" style="background-color: #4ECCA3;" data-color="#4ECCA3"></div>
        <div class="color-option" style="background-color: #FFD166;" data-color="#FFD166"></div>
        <div class="color-option" style="background-color: #A78BFA;" data-color="#A78BFA"></div>
        <div class="color-option" style="background-color: #F472B6;" data-color="#F472B6"></div>
      </div>
    `;
    propertiesSection.appendChild(colorProperty);

    // Add line width slider
    const lineWidthProperty = document.createElement('div');
    lineWidthProperty.className = 'drawing-property';
    lineWidthProperty.innerHTML = `
      <div class="drawing-property-label">Line Width</div>
      <input type="range" class="line-width-slider" min="1" max="5" value="2">
    `;
    propertiesSection.appendChild(lineWidthProperty);

    // Add actions section
    const actionsSection = document.createElement('div');
    actionsSection.className = 'drawing-actions';
    actionsSection.innerHTML = `
      <div class="drawing-action-btn clear">Clear All</div>
      <div class="drawing-action-btn save">Save</div>
    `;
    container.appendChild(actionsSection);

    // Add event listeners for color picker
    const colorOptions = colorProperty.querySelectorAll('.color-option');
    colorOptions.forEach(option => {
      option.addEventListener('click', function() {
        // Remove active class from all options
        colorOptions.forEach(opt => opt.classList.remove('active'));
        
        // Add active class to clicked option
        this.classList.add('active');
        
        // Set drawing color
        if (window.drawingTools) {
          window.drawingTools.setDrawingProps({
            color: this.dataset.color
          });
        }
      });
    });

    // Add event listener for line width slider
    const lineWidthSlider = lineWidthProperty.querySelector('.line-width-slider');
    lineWidthSlider.addEventListener('input', function() {
      if (window.drawingTools) {
        window.drawingTools.setDrawingProps({
          lineWidth: parseInt(this.value)
        });
      }
    });

    // Add event listeners for action buttons
    const clearBtn = actionsSection.querySelector('.clear');
    clearBtn.addEventListener('click', function() {
      if (window.drawingTools) {
        window.drawingTools.clearAllDrawings();
      }
    });

    const saveBtn = actionsSection.querySelector('.save');
    saveBtn.addEventListener('click', function() {
      if (window.drawingTools) {
        const drawings = window.drawingTools.exportDrawings();
        localStorage.setItem('savedDrawings', drawings);
        alert('Drawings saved successfully!');
      }
    });

    // Load saved drawings if any
    const savedDrawings = localStorage.getItem('savedDrawings');
    if (savedDrawings && window.drawingTools) {
      try {
        window.drawingTools.importDrawings(savedDrawings);
      } catch (error) {
        console.error('Error loading saved drawings:', error);
      }
    }
  }

  /**
   * Update active tool button
   * @param {string} tool - Tool ID
   */
  function updateActiveToolButton(tool) {
    const toolButtons = document.querySelectorAll('.drawing-tool-btn');
    toolButtons.forEach(btn => {
      btn.classList.toggle('active', btn.dataset.tool === tool);
    });
  }

  /**
   * Set up theme toggle handler
   */
  function setupThemeToggle() {
    const themeToggle = document.getElementById('theme-toggle');
    if (!themeToggle) return;

    themeToggle.addEventListener('click', function () {
      const currentTheme = document.documentElement.getAttribute('data-theme');
      const newTheme = currentTheme === 'dark' ? 'light' : 'dark';

      document.documentElement.setAttribute('data-theme', newTheme);
      localStorage.setItem('theme', newTheme);
      updateThemeIcons(newTheme);

      // If chart exists, update it with the new theme
      if (window.tradingChart) {
        window.tradingChart.updateChartTheme(newTheme);
      }
      
      // If drawing tools exist, update them with the new theme
      if (window.drawingTools) {
        window.drawingTools.updateTheme(newTheme);
      }
    });
  }

  // Update theme icons
  function updateThemeIcons(theme) {
    const darkIcon = document.getElementById('theme-icon-dark');
    const lightIcon = document.getElementById('theme-icon-light');

    if (!darkIcon || !lightIcon) return;

    if (theme === 'dark') {
      darkIcon.style.display = 'none';
      lightIcon.style.display = 'block';
    } else {
      darkIcon.style.display = 'block';
      lightIcon.style.display = 'none';
    }
  }
});

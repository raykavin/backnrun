/**
 * ChartDrawingTools component
 * Provides drawing tools functionality for the trading chart
 */

import { getCurrentThemeColors } from '../config/theme.js';

export class ChartDrawingTools {
  /**
   * Constructor
   * @param {Object} chart - The main chart instance
   * @param {Object} options - Configuration options
   */
  constructor(chart, options = {}) {
    this.chart = chart;
    this.options = Object.assign({
      container: document.getElementById('main-chart'),
      onDrawingComplete: null,
      onDrawingRemoved: null,
      onToolChange: null
    }, options);
    
    this.container = this.options.container;
    this.colors = getCurrentThemeColors();
    this.currentTool = null;
    this.isDrawing = false;
    this.currentDrawing = null;
    this.drawings = [];
    this.drawingId = 0;
    
    // Drawing properties
    this.drawingProps = {
      color: '#2962FF',
      lineWidth: 2,
      fontSize: 14,
      fontFamily: 'Arial, sans-serif'
    };
    
    // Mouse positions
    this.startPos = { x: 0, y: 0, time: null, price: null };
    this.endPos = { x: 0, y: 0, time: null, price: null };
    
    // Drawing layer
    this.canvas = null;
    this.ctx = null;
    
    // Text input for text tool
    this.textInput = null;
    
    // Initialize
    this.init();
  }
  
  /**
   * Initialize the drawing tools
   */
  init() {
    if (!this.container || !this.chart) {
      console.error('Drawing tools container or chart not found');
      return;
    }
    
    this.createDrawingLayer();
    this.attachEventListeners();
  }
  
  /**
   * Create the drawing layer (canvas)
   */
  createDrawingLayer() {
    // Remove existing canvas if any
    const existingCanvas = this.container.querySelector('.drawing-layer');
    if (existingCanvas) {
      existingCanvas.remove();
    }
    
    // Create canvas element
    this.canvas = document.createElement('canvas');
    this.canvas.className = 'drawing-layer';
    this.canvas.style.position = 'absolute';
    this.canvas.style.top = '0';
    this.canvas.style.left = '0';
    this.canvas.style.pointerEvents = 'none'; // Allow clicks to pass through to chart
    this.canvas.width = this.container.clientWidth;
    this.canvas.height = this.container.clientHeight;
    
    // Append canvas to container
    this.container.appendChild(this.canvas);
    
    // Get context
    this.ctx = this.canvas.getContext('2d');
    
    // Redraw all existing drawings
    this.redrawAllDrawings();
  }
  
  /**
   * Attach event listeners
   */
  attachEventListeners() {
    // Mouse events for drawing
    this.container.addEventListener('mousedown', this.handleMouseDown.bind(this));
    this.container.addEventListener('mousemove', this.handleMouseMove.bind(this));
    this.container.addEventListener('mouseup', this.handleMouseUp.bind(this));
    
    // Window resize event
    window.addEventListener('resize', this.handleResize.bind(this));
  }
  
  /**
   * Handle mouse down event
   * @param {MouseEvent} e - Mouse event
   */
  handleMouseDown(e) {
    if (!this.currentTool || this.isDrawing) return;
    
    // Start drawing
    this.isDrawing = true;
    
    // Get mouse position
    const rect = this.canvas.getBoundingClientRect();
    this.startPos.x = e.clientX - rect.left;
    this.startPos.y = e.clientY - rect.top;
    
    // Get time and price at mouse position
    const priceScale = this.chart.priceScale('right');
    const timeScale = this.chart.timeScale();
    
    if (priceScale && timeScale) {
      const coordinate = {
        x: this.startPos.x,
        y: this.startPos.y
      };
      
      // Convert coordinate to price and time
      const price = priceScale.coordinateToPrice(coordinate.y);
      const time = timeScale.coordinateToTime(coordinate.x);
      
      this.startPos.price = price;
      this.startPos.time = time;
    }
    
    // Create new drawing
    this.currentDrawing = {
      id: ++this.drawingId,
      tool: this.currentTool,
      startPos: { ...this.startPos },
      endPos: { ...this.startPos }, // Initially same as start
      props: { ...this.drawingProps },
      data: {} // Additional data specific to the tool
    };
    
    // Special handling for text tool
    if (this.currentTool === 'text') {
      this.createTextInput(this.startPos.x, this.startPos.y);
    }
  }
  
  /**
   * Handle mouse move event
   * @param {MouseEvent} e - Mouse event
   */
  handleMouseMove(e) {
    if (!this.isDrawing || !this.currentDrawing) return;
    
    // Get mouse position
    const rect = this.canvas.getBoundingClientRect();
    this.endPos.x = e.clientX - rect.left;
    this.endPos.y = e.clientY - rect.top;
    
    // Get time and price at mouse position
    const priceScale = this.chart.priceScale('right');
    const timeScale = this.chart.timeScale();
    
    if (priceScale && timeScale) {
      const coordinate = {
        x: this.endPos.x,
        y: this.endPos.y
      };
      
      // Convert coordinate to price and time
      const price = priceScale.coordinateToPrice(coordinate.y);
      const time = timeScale.coordinateToTime(coordinate.x);
      
      this.endPos.price = price;
      this.endPos.time = time;
    }
    
    // Update current drawing
    this.currentDrawing.endPos = { ...this.endPos };
    
    // Redraw
    this.redrawAllDrawings();
  }
  
  /**
   * Handle mouse up event
   * @param {MouseEvent} e - Mouse event
   */
  handleMouseUp(e) {
    if (!this.isDrawing || !this.currentDrawing) return;
    
    // Special handling for text tool
    if (this.currentTool === 'text') {
      // Text tool is handled separately with text input
      // Don't finalize drawing yet
      return;
    }
    
    // Finalize drawing
    this.finalizeDrawing();
  }
  
  /**
   * Handle window resize event
   */
  handleResize() {
    // Resize canvas
    this.canvas.width = this.container.clientWidth;
    this.canvas.height = this.container.clientHeight;
    
    // Redraw all drawings
    this.redrawAllDrawings();
  }
  
  /**
   * Set the current drawing tool
   * @param {string} tool - Tool name
   */
  setTool(tool) {
    this.currentTool = tool;
    
    // Enable/disable pointer events on canvas based on tool
    if (tool) {
      this.canvas.style.pointerEvents = 'auto';
      this.container.style.cursor = this.getToolCursor(tool);
    } else {
      this.canvas.style.pointerEvents = 'none';
      this.container.style.cursor = 'default';
    }
    
    // Call onToolChange callback if provided
    if (typeof this.options.onToolChange === 'function') {
      this.options.onToolChange(tool);
    }
  }
  
  /**
   * Get cursor style for the current tool
   * @param {string} tool - Tool name
   * @returns {string} - Cursor style
   */
  getToolCursor(tool) {
    switch (tool) {
      case 'line':
      case 'horizontalLine':
      case 'verticalLine':
      case 'fibonacciRetracement':
        return 'crosshair';
      case 'rectangle':
      case 'circle':
      case 'triangle':
        return 'crosshair';
      case 'text':
        return 'text';
      default:
        return 'default';
    }
  }
  
  /**
   * Set drawing properties
   * @param {Object} props - Drawing properties
   */
  setDrawingProps(props) {
    this.drawingProps = { ...this.drawingProps, ...props };
  }
  
  /**
   * Create text input for text tool
   * @param {number} x - X coordinate
   * @param {number} y - Y coordinate
   */
  createTextInput(x, y) {
    // Remove existing text input if any
    this.removeTextInput();
    
    // Create text input
    this.textInput = document.createElement('textarea');
    this.textInput.className = 'drawing-text-input';
    this.textInput.style.position = 'absolute';
    this.textInput.style.left = `${x}px`;
    this.textInput.style.top = `${y}px`;
    this.textInput.style.background = 'transparent';
    this.textInput.style.border = '1px dashed ' + this.drawingProps.color;
    this.textInput.style.color = this.drawingProps.color;
    this.textInput.style.fontFamily = this.drawingProps.fontFamily;
    this.textInput.style.fontSize = `${this.drawingProps.fontSize}px`;
    this.textInput.style.padding = '4px';
    this.textInput.style.minWidth = '100px';
    this.textInput.style.minHeight = '24px';
    this.textInput.style.resize = 'both';
    this.textInput.style.overflow = 'hidden';
    this.textInput.style.zIndex = '1000';
    
    // Add event listeners
    this.textInput.addEventListener('blur', this.finalizeTextDrawing.bind(this));
    this.textInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' && e.shiftKey === false) {
        e.preventDefault();
        this.finalizeTextDrawing();
      }
    });
    
    // Append to container
    this.container.appendChild(this.textInput);
    
    // Focus
    this.textInput.focus();
  }
  
  /**
   * Remove text input
   */
  removeTextInput() {
    if (this.textInput) {
      this.textInput.remove();
      this.textInput = null;
    }
  }
  
  /**
   * Finalize text drawing
   */
  finalizeTextDrawing() {
    if (!this.textInput || !this.currentDrawing) return;
    
    // Get text
    const text = this.textInput.value.trim();
    
    // If text is empty, cancel drawing
    if (!text) {
      this.cancelDrawing();
      return;
    }
    
    // Add text to drawing data
    this.currentDrawing.data.text = text;
    
    // Remove text input
    this.removeTextInput();
    
    // Finalize drawing
    this.finalizeDrawing();
  }
  
  /**
   * Finalize the current drawing
   */
  finalizeDrawing() {
    if (!this.currentDrawing) return;
    
    // Add drawing to list
    this.drawings.push(this.currentDrawing);
    
    // Call onDrawingComplete callback if provided
    if (typeof this.options.onDrawingComplete === 'function') {
      this.options.onDrawingComplete(this.currentDrawing);
    }
    
    // Reset current drawing
    this.currentDrawing = null;
    this.isDrawing = false;
    
    // Redraw
    this.redrawAllDrawings();
  }
  
  /**
   * Cancel the current drawing
   */
  cancelDrawing() {
    // Remove text input if any
    this.removeTextInput();
    
    // Reset current drawing
    this.currentDrawing = null;
    this.isDrawing = false;
    
    // Redraw
    this.redrawAllDrawings();
  }
  
  /**
   * Remove a drawing by ID
   * @param {number} id - Drawing ID
   */
  removeDrawing(id) {
    const index = this.drawings.findIndex(d => d.id === id);
    if (index !== -1) {
      const drawing = this.drawings[index];
      this.drawings.splice(index, 1);
      
      // Call onDrawingRemoved callback if provided
      if (typeof this.options.onDrawingRemoved === 'function') {
        this.options.onDrawingRemoved(drawing);
      }
      
      // Redraw
      this.redrawAllDrawings();
    }
  }
  
  /**
   * Clear all drawings
   */
  clearAllDrawings() {
    this.drawings = [];
    this.redrawAllDrawings();
  }
  
  /**
   * Redraw all drawings
   */
  redrawAllDrawings() {
    // Clear canvas
    this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
    
    // Draw all drawings
    this.drawings.forEach(drawing => {
      this.drawDrawing(drawing);
    });
    
    // Draw current drawing if any
    if (this.currentDrawing) {
      this.drawDrawing(this.currentDrawing);
    }
  }
  
  /**
   * Draw a single drawing
   * @param {Object} drawing - Drawing object
   */
  drawDrawing(drawing) {
    const { tool, startPos, endPos, props } = drawing;
    
    // Set drawing styles
    this.ctx.strokeStyle = props.color;
    this.ctx.fillStyle = props.color;
    this.ctx.lineWidth = props.lineWidth;
    this.ctx.font = `${props.fontSize}px ${props.fontFamily}`;
    
    // Draw based on tool type
    switch (tool) {
      case 'line':
        this.drawLine(startPos, endPos);
        break;
      case 'horizontalLine':
        this.drawHorizontalLine(startPos, endPos);
        break;
      case 'verticalLine':
        this.drawVerticalLine(startPos, endPos);
        break;
      case 'rectangle':
        this.drawRectangle(startPos, endPos);
        break;
      case 'circle':
        this.drawCircle(startPos, endPos);
        break;
      case 'triangle':
        this.drawTriangle(startPos, endPos);
        break;
      case 'text':
        if (drawing.data.text) {
          this.drawText(startPos, drawing.data.text);
        }
        break;
      case 'fibonacciRetracement':
        this.drawFibonacciRetracement(startPos, endPos);
        break;
    }
  }
  
  /**
   * Draw a line
   * @param {Object} start - Start position
   * @param {Object} end - End position
   */
  drawLine(start, end) {
    this.ctx.beginPath();
    this.ctx.moveTo(start.x, start.y);
    this.ctx.lineTo(end.x, end.y);
    this.ctx.stroke();
    
    // Draw small circles at start and end points
    this.drawPoint(start.x, start.y);
    this.drawPoint(end.x, end.y);
  }
  
  /**
   * Draw a horizontal line
   * @param {Object} start - Start position
   * @param {Object} end - End position
   */
  drawHorizontalLine(start, end) {
    this.ctx.beginPath();
    this.ctx.moveTo(0, start.y);
    this.ctx.lineTo(this.canvas.width, start.y);
    this.ctx.stroke();
    
    // Draw price label
    if (start.price !== null) {
      this.ctx.fillText(start.price.toFixed(2), 5, start.y - 5);
    }
  }
  
  /**
   * Draw a vertical line
   * @param {Object} start - Start position
   * @param {Object} end - End position
   */
  drawVerticalLine(start, end) {
    this.ctx.beginPath();
    this.ctx.moveTo(start.x, 0);
    this.ctx.lineTo(start.x, this.canvas.height);
    this.ctx.stroke();
    
    // Draw time label if available
    if (start.time !== null) {
      const timeStr = new Date(start.time * 1000).toLocaleTimeString();
      this.ctx.fillText(timeStr, start.x + 5, this.canvas.height - 5);
    }
  }
  
  /**
   * Draw a rectangle
   * @param {Object} start - Start position
   * @param {Object} end - End position
   */
  drawRectangle(start, end) {
    const width = end.x - start.x;
    const height = end.y - start.y;
    
    this.ctx.beginPath();
    this.ctx.rect(start.x, start.y, width, height);
    this.ctx.stroke();
    
    // Draw semi-transparent fill
    this.ctx.globalAlpha = 0.1;
    this.ctx.fill();
    this.ctx.globalAlpha = 1.0;
  }
  
  /**
   * Draw a circle
   * @param {Object} start - Start position (center)
   * @param {Object} end - End position
   */
  drawCircle(start, end) {
    const radius = Math.sqrt(
      Math.pow(end.x - start.x, 2) + Math.pow(end.y - start.y, 2)
    );
    
    this.ctx.beginPath();
    this.ctx.arc(start.x, start.y, radius, 0, 2 * Math.PI);
    this.ctx.stroke();
    
    // Draw semi-transparent fill
    this.ctx.globalAlpha = 0.1;
    this.ctx.fill();
    this.ctx.globalAlpha = 1.0;
  }
  
  /**
   * Draw a triangle
   * @param {Object} start - Start position
   * @param {Object} end - End position
   */
  drawTriangle(start, end) {
    const width = end.x - start.x;
    const height = end.y - start.y;
    
    this.ctx.beginPath();
    this.ctx.moveTo(start.x, start.y);
    this.ctx.lineTo(start.x + width, start.y);
    this.ctx.lineTo(start.x + width / 2, start.y + height);
    this.ctx.closePath();
    this.ctx.stroke();
    
    // Draw semi-transparent fill
    this.ctx.globalAlpha = 0.1;
    this.ctx.fill();
    this.ctx.globalAlpha = 1.0;
  }
  
  /**
   * Draw text
   * @param {Object} pos - Position
   * @param {string} text - Text to draw
   */
  drawText(pos, text) {
    // Draw text
    this.ctx.fillText(text, pos.x, pos.y);
  }
  
  /**
   * Draw Fibonacci retracement
   * @param {Object} start - Start position
   * @param {Object} end - End position
   */
  drawFibonacciRetracement(start, end) {
    // Fibonacci levels
    const levels = [0, 0.236, 0.382, 0.5, 0.618, 0.786, 1];
    
    // Calculate price range
    const priceRange = start.price - end.price;
    const startY = start.y;
    const endY = end.y;
    const height = endY - startY;
    
    // Draw main trend line
    this.ctx.beginPath();
    this.ctx.moveTo(start.x, start.y);
    this.ctx.lineTo(end.x, end.y);
    this.ctx.stroke();
    
    // Draw levels
    levels.forEach(level => {
      const y = startY + height * level;
      const price = start.price - priceRange * level;
      
      // Draw horizontal line
      this.ctx.beginPath();
      this.ctx.moveTo(0, y);
      this.ctx.lineTo(this.canvas.width, y);
      this.ctx.stroke();
      
      // Draw level label
      this.ctx.fillText(`${(level * 100).toFixed(1)}% - ${price.toFixed(2)}`, 5, y - 5);
    });
  }
  
  /**
   * Draw a point (small circle)
   * @param {number} x - X coordinate
   * @param {number} y - Y coordinate
   */
  drawPoint(x, y) {
    this.ctx.beginPath();
    this.ctx.arc(x, y, 3, 0, 2 * Math.PI);
    this.ctx.fill();
  }
  
  /**
   * Update chart theme
   * @param {string} theme - Theme name ('light' or 'dark')
   */
  updateTheme(theme) {
    this.colors = getCurrentThemeColors();
    this.redrawAllDrawings();
  }
  
  /**
   * Get all drawings
   * @returns {Array} - Array of drawing objects
   */
  getAllDrawings() {
    return [...this.drawings];
  }
  
  /**
   * Load drawings from saved data
   * @param {Array} drawings - Array of drawing objects
   */
  loadDrawings(drawings) {
    this.drawings = drawings.map(drawing => ({
      ...drawing,
      id: ++this.drawingId
    }));
    
    this.redrawAllDrawings();
  }
  
  /**
   * Export drawings as JSON
   * @returns {string} - JSON string
   */
  exportDrawings() {
    return JSON.stringify(this.drawings);
  }
  
  /**
   * Import drawings from JSON
   * @param {string} json - JSON string
   */
  importDrawings(json) {
    try {
      const drawings = JSON.parse(json);
      this.loadDrawings(drawings);
      return true;
    } catch (error) {
      console.error('Error importing drawings:', error);
      return false;
    }
  }
  
  /**
   * Destroy the drawing tools
   */
  destroy() {
    // Remove event listeners
    this.container.removeEventListener('mousedown', this.handleMouseDown);
    this.container.removeEventListener('mousemove', this.handleMouseMove);
    this.container.removeEventListener('mouseup', this.handleMouseUp);
    window.removeEventListener('resize', this.handleResize);
    
    // Remove canvas
    if (this.canvas) {
      this.canvas.remove();
    }
    
    // Remove text input
    this.removeTextInput();
  }
}

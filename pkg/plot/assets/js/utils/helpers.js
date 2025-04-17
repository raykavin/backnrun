/**
 * Helper functions for the trading dashboard
 */

/**
 * Extract values from arrays
 * @param {Array} rows - Array of objects
 * @param {string} key - Key to extract from each object
 * @returns {Array} - Array of values
 */
export function unpack(rows, key) {
  return rows.map(row => row[key]);
}

/**
 * Format dates for display
 * @param {string} dateStr - Date string
 * @returns {string} - Formatted date string
 */
export function formatDate(dateStr) {
  const date = new Date(dateStr);
  return date.toLocaleString();
}

/**
 * Create DOM element with optional class and parent
 * @param {string} tag - HTML tag name
 * @param {string} className - CSS class name
 * @param {HTMLElement} parent - Parent element to append to
 * @returns {HTMLElement} - Created element
 */
export function createElement(tag, className, parent) {
  const element = document.createElement(tag);
  if (className) element.className = className;
  if (parent) parent.appendChild(element);
  return element;
}

/**
 * Add an item to the legend
 * @param {HTMLElement} container - Container element
 * @param {string} name - Legend item name
 * @param {string} color - Legend item color
 * @returns {HTMLElement} - Created legend item
 */
export function addLegendItem(container, name, color) {
  const item = createElement('div', 'legend-item', container);

  const marker = createElement('div', 'legend-marker', item);
  marker.style.backgroundColor = color;

  const label = createElement('div', '', item);
  label.textContent = name;

  return item;
}

/**
 * Helper utility functions
 */

/**
 * Add a legend item to a legend container
 * @param {HTMLElement} container - Legend container
 * @param {string} text - Legend text
 * @param {string} color - Legend color
 * @returns {HTMLElement} - The created legend item
 */
export function addLegendItem(container, text, color) {
  if (!container) return null;
  
  const item = document.createElement('div');
  item.className = 'legend-item';
  
  const marker = document.createElement('div');
  marker.className = 'legend-marker';
  marker.style.backgroundColor = color;
  
  const label = document.createElement('div');
  label.className = 'legend-label';
  label.textContent = text;
  
  item.appendChild(marker);
  item.appendChild(label);
  container.appendChild(item);
  
  return item;
}

/**
 * Create an HTML element with the specified class and append it to a parent
 * @param {string} tag - HTML tag name
 * @param {string} className - CSS class name
 * @param {HTMLElement} parent - Parent element to append to
 * @returns {HTMLElement} - The created element
 */
export function createElement(tag, className, parent) {
  const element = document.createElement(tag);
  if (className) {
    element.className = className;
  }
  if (parent) {
    parent.appendChild(element);
  }
  return element;
}

/**
 * Format a number as currency
 * @param {number} value - Number to format
 * @param {string} currency - Currency code (e.g., 'USD')
 * @param {string} locale - Locale (e.g., 'en-US')
 * @returns {string} - Formatted currency string
 */
export function formatCurrency(value, currency = 'USD', locale = 'en-US') {
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency: currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(value);
}

/**
 * Format a number with specified decimal places
 * @param {number} value - Number to format
 * @param {number} decimals - Number of decimal places
 * @param {string} locale - Locale (e.g., 'en-US')
 * @returns {string} - Formatted number string
 */
export function formatNumber(value, decimals = 2, locale = 'en-US') {
  return new Intl.NumberFormat(locale, {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals
  }).format(value);
}

/**
 * Format a date
 * @param {Date|number|string} date - Date to format
 * @param {string} format - Format type ('date', 'time', 'datetime')
 * @param {string} locale - Locale (e.g., 'en-US')
 * @returns {string} - Formatted date string
 */
export function formatDate(date, format = 'datetime', locale = 'en-US') {
  const dateObj = date instanceof Date ? date : new Date(date);
  
  switch (format) {
    case 'date':
      return dateObj.toLocaleDateString(locale);
    case 'time':
      return dateObj.toLocaleTimeString(locale);
    case 'datetime':
    default:
      return dateObj.toLocaleString(locale);
  }
}

/**
 * Debounce a function
 * @param {Function} func - Function to debounce
 * @param {number} wait - Wait time in milliseconds
 * @returns {Function} - Debounced function
 */
export function debounce(func, wait = 300) {
  let timeout;
  return function(...args) {
    const context = this;
    clearTimeout(timeout);
    timeout = setTimeout(() => func.apply(context, args), wait);
  };
}

/**
 * Throttle a function
 * @param {Function} func - Function to throttle
 * @param {number} limit - Limit time in milliseconds
 * @returns {Function} - Throttled function
 */
export function throttle(func, limit = 300) {
  let inThrottle;
  return function(...args) {
    const context = this;
    if (!inThrottle) {
      func.apply(context, args);
      inThrottle = true;
      setTimeout(() => inThrottle = false, limit);
    }
  };
}

/**
 * Generate a unique ID
 * @returns {string} - Unique ID
 */
export function generateId() {
  return '_' + Math.random().toString(36).substr(2, 9);
}

/**
 * Check if a value is empty (null, undefined, empty string, empty array, empty object)
 * @param {*} value - Value to check
 * @returns {boolean} - True if empty, false otherwise
 */
export function isEmpty(value) {
  if (value === null || value === undefined) {
    return true;
  }
  
  if (typeof value === 'string' || Array.isArray(value)) {
    return value.length === 0;
  }
  
  if (typeof value === 'object') {
    return Object.keys(value).length === 0;
  }
  
  return false;
}

/**
 * Get a value from an object by path
 * @param {Object} obj - Object to get value from
 * @param {string} path - Path to value (e.g., 'user.profile.name')
 * @param {*} defaultValue - Default value if path doesn't exist
 * @returns {*} - Value at path or default value
 */
export function getValueByPath(obj, path, defaultValue = undefined) {
  const keys = path.split('.');
  let result = obj;
  
  for (const key of keys) {
    if (result === undefined || result === null) {
      return defaultValue;
    }
    result = result[key];
  }
  
  return result === undefined ? defaultValue : result;
}

/**
 * Set a value in an object by path
 * @param {Object} obj - Object to set value in
 * @param {string} path - Path to value (e.g., 'user.profile.name')
 * @param {*} value - Value to set
 * @returns {Object} - Updated object
 */
export function setValueByPath(obj, path, value) {
  const keys = path.split('.');
  const lastKey = keys.pop();
  let current = obj;
  
  for (const key of keys) {
    if (current[key] === undefined || current[key] === null) {
      current[key] = {};
    }
    current = current[key];
  }
  
  current[lastKey] = value;
  return obj;
}

/**
 * Deep clone an object
 * @param {*} obj - Object to clone
 * @returns {*} - Cloned object
 */
export function deepClone(obj) {
  if (obj === null || typeof obj !== 'object') {
    return obj;
  }
  
  if (Array.isArray(obj)) {
    return obj.map(item => deepClone(item));
  }
  
  const cloned = {};
  for (const key in obj) {
    if (Object.prototype.hasOwnProperty.call(obj, key)) {
      cloned[key] = deepClone(obj[key]);
    }
  }
  
  return cloned;
}

/**
 * Deep merge two objects
 * @param {Object} target - Target object
 * @param {Object} source - Source object
 * @returns {Object} - Merged object
 */
export function deepMerge(target, source) {
  const output = { ...target };
  
  if (isObject(target) && isObject(source)) {
    Object.keys(source).forEach(key => {
      if (isObject(source[key])) {
        if (!(key in target)) {
          output[key] = source[key];
        } else {
          output[key] = deepMerge(target[key], source[key]);
        }
      } else {
        output[key] = source[key];
      }
    });
  }
  
  return output;
}

/**
 * Check if a value is an object
 * @param {*} item - Value to check
 * @returns {boolean} - True if object, false otherwise
 */
function isObject(item) {
  return (item && typeof item === 'object' && !Array.isArray(item));
}

/**
 * Convert a string to camelCase
 * @param {string} str - String to convert
 * @returns {string} - camelCase string
 */
export function toCamelCase(str) {
  return str
    .replace(/(?:^\w|[A-Z]|\b\w)/g, (word, index) => {
      return index === 0 ? word.toLowerCase() : word.toUpperCase();
    })
    .replace(/\s+/g, '');
}

/**
 * Convert a string to kebab-case
 * @param {string} str - String to convert
 * @returns {string} - kebab-case string
 */
export function toKebabCase(str) {
  return str
    .replace(/([a-z])([A-Z])/g, '$1-$2')
    .replace(/\s+/g, '-')
    .toLowerCase();
}

/**
 * Convert a string to snake_case
 * @param {string} str - String to convert
 * @returns {string} - snake_case string
 */
export function toSnakeCase(str) {
  return str
    .replace(/([a-z])([A-Z])/g, '$1_$2')
    .replace(/\s+/g, '_')
    .toLowerCase();
}

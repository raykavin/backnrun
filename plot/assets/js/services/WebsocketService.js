/**
 * WebSocket service for real-time data updates
 */

let socket = null;
let reconnectAttempts = 0;
let maxReconnectAttempts = 5;
let reconnectTimeout = null;
let messageHandlers = {};
let isConnecting = false;

// DOM elements for connection status
let statusIndicator = null;
let statusText = null;

/**
 * Initialize WebSocket connection
 * @param {string} pair - Trading pair
 * @returns {Promise} - Promise that resolves when connection is established
 */
export function initWebSocket(pair) {
  // Get status indicator elements
  statusIndicator = document.getElementById('ws-status-indicator');
  statusText = document.getElementById('ws-status-text');
  
  // Update status to connecting
  updateConnectionStatus('connecting');
  return new Promise((resolve, reject) => {
    if (socket && socket.readyState === WebSocket.OPEN) {
      console.log('WebSocket already connected');
      resolve(socket);
      return;
    }

    if (isConnecting) {
      console.log('WebSocket connection already in progress');
      // Wait for the connection to be established
      const checkInterval = setInterval(() => {
        if (socket && socket.readyState === WebSocket.OPEN) {
          clearInterval(checkInterval);
          resolve(socket);
        }
      }, 100);
      return;
    }

    isConnecting = true;
    
    // Close existing socket if it exists
    if (socket) {
      socket.close();
    }

    // Determine WebSocket URL
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws?pair=${pair}`;
    
    console.log(`Connecting to WebSocket at ${wsUrl}`);
    
    try {
      socket = new WebSocket(wsUrl);
      
      socket.onopen = () => {
        console.log('WebSocket connection established');
        reconnectAttempts = 0;
        isConnecting = false;
        updateConnectionStatus('connected');
        resolve(socket);
      };
      
      socket.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          handleMessage(message);
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      };
      
      socket.onerror = (error) => {
        console.error('WebSocket error:', error);
        isConnecting = false;
        updateConnectionStatus('disconnected');
        reject(error);
      };
      
      socket.onclose = (event) => {
        console.log(`WebSocket connection closed: ${event.code} ${event.reason}`);
        isConnecting = false;
        updateConnectionStatus('disconnected');
        
        // Attempt to reconnect if not a normal closure
        if (event.code !== 1000 && event.code !== 1001) {
          attemptReconnect(pair);
        }
      };
    } catch (error) {
      console.error('Error creating WebSocket:', error);
      isConnecting = false;
      reject(error);
    }
  });
}

/**
 * Attempt to reconnect to WebSocket
 * @param {string} pair - Trading pair
 */
function attemptReconnect(pair) {
  if (reconnectAttempts >= maxReconnectAttempts) {
    console.error('Maximum reconnect attempts reached');
    return;
  }
  
  reconnectAttempts++;
  
  const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
  console.log(`Attempting to reconnect in ${delay}ms (attempt ${reconnectAttempts}/${maxReconnectAttempts})`);
  
  clearTimeout(reconnectTimeout);
  reconnectTimeout = setTimeout(() => {
    console.log('Reconnecting...');
    initWebSocket(pair).catch(error => {
      console.error('Reconnect failed:', error);
    });
  }, delay);
}

/**
 * Close WebSocket connection
 */
export function closeWebSocket() {
  if (socket) {
    socket.close();
    socket = null;
  }
  
  clearTimeout(reconnectTimeout);
  reconnectAttempts = 0;
  isConnecting = false;
  updateConnectionStatus('disconnected');
}

/**
 * Update connection status UI
 * @param {string} status - Connection status ('connected', 'connecting', 'disconnected')
 */
function updateConnectionStatus(status) {
  if (!statusIndicator || !statusText) {
    statusIndicator = document.getElementById('ws-status-indicator');
    statusText = document.getElementById('ws-status-text');
    
    if (!statusIndicator || !statusText) {
      console.warn('Connection status elements not found in DOM');
      return;
    }
  }
  
  // Remove all status classes
  statusIndicator.classList.remove('connected', 'connecting', 'disconnected');
  
  // Add appropriate class and update text
  statusIndicator.classList.add(status);
  
  switch (status) {
    case 'connected':
      statusText.textContent = 'Conectado';
      break;
    case 'connecting':
      statusText.textContent = 'Conectando...';
      break;
    case 'disconnected':
      statusText.textContent = 'Desconectado';
      break;
    default:
      statusText.textContent = 'Desconhecido';
  }
}

/**
 * Register a message handler
 * @param {string} messageType - Message type to handle
 * @param {Function} handler - Handler function
 */
export function registerMessageHandler(messageType, handler) {
  if (!messageHandlers[messageType]) {
    messageHandlers[messageType] = [];
  }
  
  messageHandlers[messageType].push(handler);
}

/**
 * Unregister a message handler
 * @param {string} messageType - Message type
 * @param {Function} handler - Handler function to remove
 */
export function unregisterMessageHandler(messageType, handler) {
  if (!messageHandlers[messageType]) {
    return;
  }
  
  messageHandlers[messageType] = messageHandlers[messageType].filter(h => h !== handler);
}

/**
 * Handle incoming WebSocket message
 * @param {Object} message - Message object
 */
function handleMessage(message) {
  const { type, payload } = message;
  
  console.log(`Received WebSocket message of type: ${type}`);
  
  // Special handling for initialData if we have a pending handler
  if (type === 'initialData' && window.pendingInitialDataHandler) {
    try {
      window.pendingInitialDataHandler(payload);
      window.pendingInitialDataHandler = null;
      console.log('Handled initial data with pending handler');
    } catch (error) {
      console.error('Error in pending initial data handler:', error);
    }
  }
  
  // Regular handler processing
  if (!messageHandlers[type]) {
    console.warn(`No handlers registered for message type: ${type}`);
    return;
  }
  
  // Call all registered handlers for this message type
  messageHandlers[type].forEach(handler => {
    try {
      handler(payload);
    } catch (error) {
      console.error(`Error in handler for message type ${type}:`, error);
    }
  });
}

/**
 * Check if WebSocket is connected
 * @returns {boolean} - True if connected
 */
export function isWebSocketConnected() {
  return socket && socket.readyState === WebSocket.OPEN;
}

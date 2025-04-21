/**
 * Tooltip component for displaying information on hover
 */

import { formatDate } from '../utils/helpers.js';
import { createElement } from '../utils/helpers.js';

/**
 * Create tooltip element
 * @returns {HTMLElement} - Tooltip element
 */
export function createTooltip() {
  const tooltip = createElement('div', 'tooltip', document.body);
  tooltip.style.display = 'none';
  return tooltip;
}

/**
 * Setup tooltip functionality
 * @param {Object} mainChart - Main chart instance
 * @param {HTMLElement} tooltip - Tooltip element
 * @param {Array} buyMarkers - Buy markers
 * @param {Array} sellMarkers - Sell markers
 */
export function setupTooltip(mainChart, tooltip, buyMarkers, sellMarkers) {
  mainChart.subscribeCrosshairMove(param => {
    if (!param.point || !param.time) {
      tooltip.style.display = 'none';
      return;
    }

    const markers = [...buyMarkers, ...sellMarkers];
    const marker = markers.find(m => m.time === param.time);

    if (marker && marker.order) {
      const order = marker.order;
      tooltip.innerHTML = `
        <div><strong>${order.side} Order</strong></div>
        <div>Time: ${formatDate(order.updated_at || order.created_at)}</div>
        <div>ID: ${order.id}</div>
        <div>Price: ${order.price.toLocaleString()}</div>
        <div>Size: ${order.quantity.toPrecision(4).toLocaleString()}</div>
        <div>Type: ${order.type}</div>
        ${order.profit ? `<div>Profit: ${(order.profit * 100).toPrecision(2).toLocaleString()}%</div>` : ''}
      `;

      const x = param.point.x;
      const y = param.point.y;

      tooltip.style.display = 'block';
      tooltip.style.left = `${x + 15}px`;
      tooltip.style.top = `${y + 15}px`;
    } else {
      tooltip.style.display = 'none';
    }
  });
}

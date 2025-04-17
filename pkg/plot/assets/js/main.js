/**
 * Main entry point for the trading dashboard
 */

import { TradingChart } from './components/TradingChart.js';

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

  // Set up theme toggle handler
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

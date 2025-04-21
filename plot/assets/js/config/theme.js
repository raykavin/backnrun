/**
 * Theme configuration for the trading dashboard
 * Contains color definitions for both light and dark themes
 */

// Theme-sensitive colors
export const THEME_COLORS = {
  light: {
    UP: '#26a69a',
    DOWN: '#ef5350',
    EQUITY: 'rgba(38, 166, 154, 1)',
    ASSET: 'rgba(239, 83, 80, 1)',
    DEFAULT_INDICATOR: '#2196F3',
    GRID: 'rgba(197, 203, 206, 0.5)',
    BORDER: 'rgba(197, 203, 206, 1)',
    BACKGROUND: '#ffffff',
    TEXT: '#333'
  },
  dark: {
    UP: '#4ecca3',
    DOWN: '#ff6b6b',
    EQUITY: 'rgba(78, 204, 163, 1)',
    ASSET: 'rgba(255, 107, 107, 1)',
    DEFAULT_INDICATOR: '#64b5f6',
    GRID: 'rgba(120, 123, 134, 0.3)',
    BORDER: 'rgba(120, 123, 134, 1)',
    BACKGROUND: '#1e1e1e',
    TEXT: '#e0e0e0'
  }
};

// Get current theme colors based on data-theme attribute
export function getCurrentThemeColors() {
  const isDarkMode = document.documentElement.getAttribute('data-theme') === 'dark';
  return isDarkMode ? THEME_COLORS.dark : THEME_COLORS.light;
}

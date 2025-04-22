/**
 * PaneResizer class
 * Implements TradingView-style pane resizing
 */
export class PaneResizer {
    /**
     * Constructor
     * @param {Object} options - Configuration options
     */
    constructor(options = {}) {
        this.options = Object.assign({
            minPaneHeight: 50,     // Minimum pane height in pixels
            handleHeight: 8,       // Height of resize handle in pixels
            onResize: null,        // Callback when resize occurs
            handleClass: 'pane-resize-handle'
        }, options);

        this.chartPanes = [];
        this.activeHandle = null;
        this.activeIndex = -1;
        this.startY = 0;
        this.startHeights = [];
        this.totalHeight = 0;
        this.isResizing = false;

        // Bind event handlers
        this.handleMouseDown = this.handleMouseDown.bind(this);
        this.handleMouseMove = this.handleMouseMove.bind(this);
        this.handleMouseUp = this.handleMouseUp.bind(this);

        // Add global event listeners
        document.addEventListener('mousemove', this.handleMouseMove);
        document.addEventListener('mouseup', this.handleMouseUp);
    }

    /**
     * Set up panes for resizing
     * @param {Array} panes - Array of pane container elements
     */
    setupPanes(panes) {
        this.chartPanes = panes;

        // Remove existing handles
        this.removeAllHandles();

        // No handles needed if we have 0 or 1 pane
        if (!panes || panes.length <= 1) return;

        // Create resize handles between panes
        panes.forEach((pane, index) => {
            // Don't add handle after the last pane
            if (index === panes.length - 1) return;

            this.addResizeHandle(pane, index);
        });

        // Store initial heights
        this.updateInitialHeights();
    }

    /**
     * Add a resize handle to a pane
     * @param {HTMLElement} pane - Pane container element
     * @param {number} index - Index of the pane
     */
    addResizeHandle(pane, index) {
        // Create handle element
        const handle = document.createElement('div');
        handle.className = this.options.handleClass;
        handle.dataset.index = index;

        // Position at the bottom of the pane
        handle.style.position = 'absolute';
        handle.style.left = '0';
        handle.style.right = '0';
        handle.style.bottom = '0';
        handle.style.height = `${this.options.handleHeight}px`;
        handle.style.zIndex = '100';
        handle.style.cursor = 'ns-resize';
        handle.style.backgroundColor = 'transparent';

        // Add hover line to indicate it's a handle
        const line = document.createElement('div');
        line.className = `${this.options.handleClass}-line`;
        line.style.position = 'absolute';
        line.style.left = '0';
        line.style.right = '0';
        line.style.top = '50%';
        line.style.height = '1px';
        line.style.backgroundColor = 'rgba(100, 100, 100, 0.2)';
        line.style.transform = 'translateY(-50%)';
        line.style.transition = 'background-color 0.2s ease, height 0.2s ease';
        handle.appendChild(line);

        // Add hover and active states
        handle.addEventListener('mouseenter', () => {
            line.style.backgroundColor = 'rgba(54, 116, 217, 0.5)';
            line.style.height = '3px';
        });

        handle.addEventListener('mouseleave', () => {
            if (this.activeHandle !== handle) {
                line.style.backgroundColor = 'rgba(100, 100, 100, 0.2)';
                line.style.height = '1px';
            }
        });

        // Add mousedown event listener
        handle.addEventListener('mousedown', (e) => this.handleMouseDown(e, handle, index));

        // Make sure pane has position relative to position handle correctly
        if (getComputedStyle(pane).position === 'static') {
            pane.style.position = 'relative';
        }

        // Add to pane
        pane.appendChild(handle);
    }

    /**
     * Update initial heights of all panes
     */
    updateInitialHeights() {
        this.startHeights = this.chartPanes.map(pane => pane.clientHeight);
        this.totalHeight = this.startHeights.reduce((sum, height) => sum + height, 0);
    }

    /**
     * Handle mouse down event on a resize handle
     * @param {MouseEvent} e - Mouse event
     * @param {HTMLElement} handle - Handle element
     * @param {number} index - Index of the pane above the handle
     */
    handleMouseDown(e, handle, index) {
        e.preventDefault();
        e.stopPropagation();

        this.isResizing = true;
        this.activeHandle = handle;
        this.activeIndex = index;
        this.startY = e.clientY;

        // Update heights in case something changed
        this.updateInitialHeights();

        // Add active class to handle
        const line = handle.querySelector(`.${this.options.handleClass}-line`);
        if (line) {
            line.style.backgroundColor = 'rgba(54, 116, 217, 0.8)';
            line.style.height = '3px';
        }

        // Add resizing class to body
        document.body.classList.add('resizing');
    }

    /**
     * Handle mouse move event
     * @param {MouseEvent} e - Mouse event
     */
    handleMouseMove(e) {
        if (!this.isResizing || this.activeIndex < 0) return;

        const deltaY = e.clientY - this.startY;

        // Get the two panes affected by this handle
        const topPane = this.chartPanes[this.activeIndex];
        const bottomPane = this.chartPanes[this.activeIndex + 1];

        if (!topPane || !bottomPane) return;

        // Calculate new heights
        let newTopHeight = this.startHeights[this.activeIndex] + deltaY;
        let newBottomHeight = this.startHeights[this.activeIndex + 1] - deltaY;

        // Enforce minimum heights
        if (newTopHeight < this.options.minPaneHeight) {
            const deficit = this.options.minPaneHeight - newTopHeight;
            newTopHeight = this.options.minPaneHeight;
            newBottomHeight -= deficit;
        }

        if (newBottomHeight < this.options.minPaneHeight) {
            const deficit = this.options.minPaneHeight - newBottomHeight;
            newBottomHeight = this.options.minPaneHeight;
            newTopHeight -= deficit;
        }

        // Apply new heights
        topPane.style.height = `${newTopHeight}px`;
        bottomPane.style.height = `${newBottomHeight}px`;

        // Resize charts in affected panes
        this.resizeChartInPane(topPane);
        this.resizeChartInPane(bottomPane);

        // Call onResize callback if provided
        if (typeof this.options.onResize === 'function') {
            this.options.onResize(this.activeIndex, newTopHeight, newBottomHeight);
        }
    }

    /**
     * Resize chart in a pane
     * @param {HTMLElement} pane - Pane element
     */
    resizeChartInPane(pane) {
        // Find chart instance on the pane element
        const chart = pane.chartInstance;

        // Resize chart if it exists
        if (chart && typeof chart.applyOptions === 'function') {
            chart.applyOptions({
                width: pane.clientWidth,
                height: pane.clientHeight - this.options.handleHeight
            });
        }
    }

    /**
     * Handle mouse up event
     */
    handleMouseUp() {
        if (!this.isResizing) return;

        this.isResizing = false;

        // Reset active handle styling
        if (this.activeHandle) {
            const line = this.activeHandle.querySelector(`.${this.options.handleClass}-line`);
            if (line) {
                line.style.backgroundColor = 'rgba(100, 100, 100, 0.2)';
                line.style.height = '1px';
            }
        }

        // Remove resizing class from body
        document.body.classList.remove('resizing');

        // Reset variables
        this.activeHandle = null;
        this.activeIndex = -1;

        // Update heights
        this.updateInitialHeights();

        // Resize all charts one final time
        this.resizeAllCharts();
    }

    /**
     * Resize all charts
     */
    resizeAllCharts() {
        this.chartPanes.forEach(pane => {
            this.resizeChartInPane(pane);
        });
    }

    /**
     * Remove all resize handles
     */
    removeAllHandles() {
        // Remove existing handles
        const handles = document.querySelectorAll(`.${this.options.handleClass}`);
        handles.forEach(handle => {
            const parent = handle.parentNode;
            if (parent) {
                parent.removeChild(handle);
            }
        });
    }

    /**
     * Destroy resizer and clean up
     */
    destroy() {
        // Remove event listeners
        document.removeEventListener('mousemove', this.handleMouseMove);
        document.removeEventListener('mouseup', this.handleMouseUp);

        // Remove all handles
        this.removeAllHandles();

        // Clear panes array
        this.chartPanes = [];
    }
}
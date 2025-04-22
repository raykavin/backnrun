/**
 * ManualOrderForm component
 * Handles the creation and submission of manual orders
 */

import { getCurrentThemeColors } from '../config/Theme.js';
import { createElement } from '../utils/Helpers.js';

export class ManualOrderForm {
  /**
   * Constructor
   * @param {string} containerId - ID of the container element
   * @param {Object} options - Configuration options
   */
  constructor(containerId, options = {}) {
    this.container = document.getElementById(containerId);
    this.options = Object.assign({
      onSubmit: null,
      defaultPair: 'BTC/USDT',
      availableBalance: {
        USDT: 25432.18,
        BTC: 0.85,
        ETH: 12.5,
        SOL: 125,
      }
    }, options);
    
    this.currentPair = this.options.defaultPair;
    this.currentSide = 'buy';
    this.currentType = 'market';
    this.colors = getCurrentThemeColors();
    
    // Elements references
    this.elements = {
      form: null,
      typeSelect: null,
      sideRadios: null,
      amountInput: null,
      priceInput: null,
      totalDisplay: null,
      availableDisplay: null,
      submitButton: null,
      sliderContainer: null,
      percentageSlider: null,
      leverageContainer: null,
      leverageSlider: null,
      leverageValue: null,
      advancedToggle: null,
      advancedOptions: null,
      stopLossInput: null,
      takeProfitInput: null,
      errorMessage: null
    };
    
    // Initialize
    this.init();
  }
  
  /**
   * Initialize the form
   */
  init() {
    if (!this.container) {
      console.error('Order form container not found');
      return;
    }
    
    this.render();
    this.attachEventListeners();
    this.updateAvailableBalance();
    this.calculateTotal();
  }
  
  /**
   * Render the form
   */
  render() {
    // Clear container
    this.container.innerHTML = '';
    
    // Create form
    const form = createElement('form', 'order-form', this.container);
    form.id = 'manual-order-form';
    this.elements.form = form;
    
    // Create header with tabs for order types
    const header = createElement('div', 'order-form-header', form);
    
    const tabsContainer = createElement('div', 'order-type-tabs', header);
    
    ['Market', 'Limit', 'Stop'].forEach(type => {
      const typeId = type.toLowerCase();
      const tab = createElement('div', `order-type-tab ${typeId === this.currentType ? 'active' : ''}`, tabsContainer);
      tab.dataset.type = typeId;
      tab.textContent = type;
      
      tab.addEventListener('click', () => {
        this.setOrderType(typeId);
      });
    });
    
    // Create side selection (Buy/Sell)
    const sideContainer = createElement('div', 'order-side-container', form);
    
    const buyBtn = createElement('button', `order-side-btn buy ${this.currentSide === 'buy' ? 'active' : ''}`, sideContainer);
    buyBtn.type = 'button';
    buyBtn.dataset.side = 'buy';
    buyBtn.innerHTML = '<i class="fas fa-arrow-up me-1"></i>Buy';
    
    const sellBtn = createElement('button', `order-side-btn sell ${this.currentSide === 'sell' ? 'active' : ''}`, sideContainer);
    sellBtn.type = 'button';
    sellBtn.dataset.side = 'sell';
    sellBtn.innerHTML = '<i class="fas fa-arrow-down me-1"></i>Sell';
    
    this.elements.sideRadios = { buyBtn, sellBtn };
    
    // Create amount input
    const amountGroup = createElement('div', 'form-group', form);
    const amountLabel = createElement('label', 'form-label', amountGroup);
    amountLabel.textContent = 'Amount';
    
    const amountInputGroup = createElement('div', 'input-group', amountGroup);
    
    const amountInput = createElement('input', 'form-control', amountInputGroup);
    amountInput.type = 'number';
    amountInput.min = '0';
    amountInput.step = '0.001';
    amountInput.placeholder = '0.00';
    this.elements.amountInput = amountInput;
    
    const amountCurrency = createElement('span', 'input-group-text', amountInputGroup);
    amountCurrency.textContent = this.currentPair.split('/')[0]; // First part of the pair (e.g., BTC)
    
    // Create percentage slider
    const sliderContainer = createElement('div', 'percentage-slider-container', amountGroup);
    this.elements.sliderContainer = sliderContainer;
    
    const percentageSlider = createElement('input', 'percentage-slider', sliderContainer);
    percentageSlider.type = 'range';
    percentageSlider.min = '0';
    percentageSlider.max = '100';
    percentageSlider.step = '1';
    percentageSlider.value = '0';
    this.elements.percentageSlider = percentageSlider;
    
    const percentageLabels = createElement('div', 'percentage-labels', sliderContainer);
    
    [0, 25, 50, 75, 100].forEach(percent => {
      const label = createElement('span', 'percentage-label', percentageLabels);
      label.textContent = `${percent}%`;
      label.style.left = `${percent}%`;
      
      // Add click handler to set slider to this percentage
      label.addEventListener('click', () => {
        percentageSlider.value = percent;
        this.updateAmountFromPercentage(percent);
      });
    });
    
    // Create price input (hidden for market orders)
    const priceGroup = createElement('div', 'form-group', form);
    priceGroup.id = 'price-input-group';
    if (this.currentType === 'market') {
      priceGroup.style.display = 'none';
    }
    
    const priceLabel = createElement('label', 'form-label', priceGroup);
    priceLabel.textContent = 'Price';
    
    const priceInputGroup = createElement('div', 'input-group', priceGroup);
    
    const priceInput = createElement('input', 'form-control', priceInputGroup);
    priceInput.type = 'number';
    priceInput.min = '0';
    priceInput.step = '0.01';
    priceInput.placeholder = '0.00';
    this.elements.priceInput = priceInput;
    
    const priceCurrency = createElement('span', 'input-group-text', priceInputGroup);
    priceCurrency.textContent = this.currentPair.split('/')[1]; // Second part of the pair (e.g., USDT)
    
    // Create leverage slider (initially hidden, shown in advanced options)
    const leverageContainer = createElement('div', 'leverage-container', form);
    leverageContainer.style.display = 'none';
    this.elements.leverageContainer = leverageContainer;
    
    const leverageLabel = createElement('label', 'form-label', leverageContainer);
    leverageLabel.textContent = 'Leverage';
    
    const leverageSliderContainer = createElement('div', 'leverage-slider-container', leverageContainer);
    
    const leverageSlider = createElement('input', 'leverage-slider', leverageSliderContainer);
    leverageSlider.type = 'range';
    leverageSlider.min = '1';
    leverageSlider.max = '100';
    leverageSlider.step = '1';
    leverageSlider.value = '1';
    this.elements.leverageSlider = leverageSlider;
    
    const leverageValue = createElement('span', 'leverage-value', leverageSliderContainer);
    leverageValue.textContent = '1x';
    this.elements.leverageValue = leverageValue;
    
    // Create advanced options toggle
    const advancedToggle = createElement('button', 'advanced-toggle', form);
    advancedToggle.type = 'button';
    advancedToggle.innerHTML = 'Advanced Options <i class="fas fa-chevron-down"></i>';
    this.elements.advancedToggle = advancedToggle;
    
    // Create advanced options container (initially hidden)
    const advancedOptions = createElement('div', 'advanced-options', form);
    advancedOptions.style.display = 'none';
    this.elements.advancedOptions = advancedOptions;
    
    // Stop Loss
    const stopLossGroup = createElement('div', 'form-group', advancedOptions);
    const stopLossLabel = createElement('label', 'form-label', stopLossGroup);
    stopLossLabel.textContent = 'Stop Loss';
    
    const stopLossInputGroup = createElement('div', 'input-group', stopLossGroup);
    
    const stopLossInput = createElement('input', 'form-control', stopLossInputGroup);
    stopLossInput.type = 'number';
    stopLossInput.min = '0';
    stopLossInput.step = '0.01';
    stopLossInput.placeholder = 'Optional';
    this.elements.stopLossInput = stopLossInput;
    
    const stopLossCurrency = createElement('span', 'input-group-text', stopLossInputGroup);
    stopLossCurrency.textContent = this.currentPair.split('/')[1];
    
    // Take Profit
    const takeProfitGroup = createElement('div', 'form-group', advancedOptions);
    const takeProfitLabel = createElement('label', 'form-label', takeProfitGroup);
    takeProfitLabel.textContent = 'Take Profit';
    
    const takeProfitInputGroup = createElement('div', 'input-group', takeProfitGroup);
    
    const takeProfitInput = createElement('input', 'form-control', takeProfitInputGroup);
    takeProfitInput.type = 'number';
    takeProfitInput.min = '0';
    takeProfitInput.step = '0.01';
    takeProfitInput.placeholder = 'Optional';
    this.elements.takeProfitInput = takeProfitInput;
    
    const takeProfitCurrency = createElement('span', 'input-group-text', takeProfitInputGroup);
    takeProfitCurrency.textContent = this.currentPair.split('/')[1];
    
    // Leverage option (checkbox to show/hide leverage slider)
    const leverageCheckGroup = createElement('div', 'form-check', advancedOptions);
    
    const leverageCheck = createElement('input', 'form-check-input', leverageCheckGroup);
    leverageCheck.type = 'checkbox';
    leverageCheck.id = 'leverage-check';
    
    const leverageCheckLabel = createElement('label', 'form-check-label', leverageCheckGroup);
    leverageCheckLabel.htmlFor = 'leverage-check';
    leverageCheckLabel.textContent = 'Use Leverage';
    
    // Add event listener to toggle leverage container
    leverageCheck.addEventListener('change', () => {
      this.elements.leverageContainer.style.display = leverageCheck.checked ? 'block' : 'none';
      this.calculateTotal();
    });
    
    // Create order summary
    const summaryContainer = createElement('div', 'order-summary', form);
    
    const totalContainer = createElement('div', 'summary-item', summaryContainer);
    const totalLabel = createElement('div', 'summary-label', totalContainer);
    totalLabel.textContent = 'Total';
    
    const totalDisplay = createElement('div', 'summary-value', totalContainer);
    totalDisplay.textContent = '0.00 USDT';
    this.elements.totalDisplay = totalDisplay;
    
    const availableContainer = createElement('div', 'summary-item', summaryContainer);
    const availableLabel = createElement('div', 'summary-label', availableContainer);
    availableLabel.textContent = 'Available';
    
    const availableDisplay = createElement('div', 'summary-value', availableContainer);
    availableDisplay.textContent = `${this.options.availableBalance.USDT.toLocaleString()} USDT`;
    this.elements.availableDisplay = availableDisplay;
    
    // Error message container
    const errorMessage = createElement('div', 'error-message', form);
    errorMessage.style.display = 'none';
    this.elements.errorMessage = errorMessage;
    
    // Create submit button
    const submitButton = createElement('button', 'btn btn-primary w-100', form);
    submitButton.type = 'submit';
    submitButton.innerHTML = '<i class="fas fa-check-circle me-1"></i>Place Order';
    this.elements.submitButton = submitButton;
  }
  
  /**
   * Attach event listeners
   */
  attachEventListeners() {
    // Form submission
    this.elements.form.addEventListener('submit', (e) => {
      e.preventDefault();
      this.submitOrder();
    });
    
    // Side selection
    this.elements.sideRadios.buyBtn.addEventListener('click', () => this.setOrderSide('buy'));
    this.elements.sideRadios.sellBtn.addEventListener('click', () => this.setOrderSide('sell'));
    
    // Amount and price inputs
    this.elements.amountInput.addEventListener('input', () => {
      this.calculateTotal();
      this.updatePercentageFromAmount();
    });
    
    this.elements.priceInput.addEventListener('input', () => {
      this.calculateTotal();
    });
    
    // Percentage slider
    this.elements.percentageSlider.addEventListener('input', () => {
      this.updateAmountFromPercentage(this.elements.percentageSlider.value);
    });
    
    // Leverage slider
    this.elements.leverageSlider.addEventListener('input', () => {
      const value = this.elements.leverageSlider.value;
      this.elements.leverageValue.textContent = `${value}x`;
      this.calculateTotal();
    });
    
    // Advanced options toggle
    this.elements.advancedToggle.addEventListener('click', () => {
      const isVisible = this.elements.advancedOptions.style.display === 'block';
      this.elements.advancedOptions.style.display = isVisible ? 'none' : 'block';
      this.elements.advancedToggle.innerHTML = `Advanced Options <i class="fas fa-chevron-${isVisible ? 'down' : 'up'}"></i>`;
    });
    
    // Stop loss and take profit inputs
    this.elements.stopLossInput.addEventListener('input', () => this.validateStopLossAndTakeProfit());
    this.elements.takeProfitInput.addEventListener('input', () => this.validateStopLossAndTakeProfit());
  }
  
  /**
   * Set the order type
   * @param {string} type - Order type (market, limit, stop)
   */
  setOrderType(type) {
    this.currentType = type;
    
    // Update UI
    const tabs = this.container.querySelectorAll('.order-type-tab');
    tabs.forEach(tab => {
      tab.classList.toggle('active', tab.dataset.type === type);
    });
    
    // Show/hide price input based on order type
    const priceGroup = document.getElementById('price-input-group');
    if (priceGroup) {
      priceGroup.style.display = type === 'market' ? 'none' : 'block';
    }
    
    // Clear price input for market orders
    if (type === 'market') {
      this.elements.priceInput.value = '';
    }
    
    this.calculateTotal();
  }
  
  /**
   * Set the order side
   * @param {string} side - Order side (buy, sell)
   */
  setOrderSide(side) {
    this.currentSide = side;
    
    // Update UI
    this.elements.sideRadios.buyBtn.classList.toggle('active', side === 'buy');
    this.elements.sideRadios.sellBtn.classList.toggle('active', side === 'sell');
    
    this.updateAvailableBalance();
    this.calculateTotal();
  }
  
  /**
   * Update the available balance display
   */
  updateAvailableBalance() {
    const pair = this.currentPair.split('/');
    const baseCurrency = pair[0]; // e.g., BTC
    const quoteCurrency = pair[1]; // e.g., USDT
    
    let availableCurrency, availableAmount;
    
    if (this.currentSide === 'buy') {
      // When buying, we spend quote currency (e.g., USDT)
      availableCurrency = quoteCurrency;
      availableAmount = this.options.availableBalance[quoteCurrency] || 0;
    } else {
      // When selling, we spend base currency (e.g., BTC)
      availableCurrency = baseCurrency;
      availableAmount = this.options.availableBalance[baseCurrency] || 0;
    }
    
    this.elements.availableDisplay.textContent = `${availableAmount.toLocaleString()} ${availableCurrency}`;
  }
  
  /**
   * Calculate the total order value
   */
  calculateTotal() {
    const amount = parseFloat(this.elements.amountInput.value) || 0;
    let price = 0;
    
    if (this.currentType === 'market') {
      // For market orders, use the current market price (this would be fetched from an API in a real app)
      // For now, we'll use a placeholder value
      price = 60000; // Placeholder BTC price
    } else {
      price = parseFloat(this.elements.priceInput.value) || 0;
    }
    
    // Calculate total
    let total = amount * price;
    
    // Apply leverage if enabled
    const leverageContainer = this.elements.leverageContainer;
    if (leverageContainer && leverageContainer.style.display === 'block') {
      const leverage = parseInt(this.elements.leverageSlider.value) || 1;
      // In a real app, you'd calculate this differently based on your margin requirements
      // This is a simplified example
      total = total / leverage;
    }
    
    // Update total display
    const quoteCurrency = this.currentPair.split('/')[1]; // e.g., USDT
    this.elements.totalDisplay.textContent = `${total.toLocaleString(undefined, {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    })} ${quoteCurrency}`;
    
    return total;
  }
  
  /**
   * Update the amount input based on the percentage slider
   * @param {number} percentage - Percentage value (0-100)
   */
  updateAmountFromPercentage(percentage) {
    const pair = this.currentPair.split('/');
    const baseCurrency = pair[0]; // e.g., BTC
    const quoteCurrency = pair[1]; // e.g., USDT
    
    let availableAmount;
    
    if (this.currentSide === 'buy') {
      // When buying, we use quote currency (e.g., USDT)
      availableAmount = this.options.availableBalance[quoteCurrency] || 0;
      
      // For buying, we need to convert the quote currency to base currency
      // using the current price
      let price;
      if (this.currentType === 'market') {
        // For market orders, use the current market price
        price = 60000; // Placeholder BTC price
      } else {
        price = parseFloat(this.elements.priceInput.value) || 0;
      }
      
      if (price > 0) {
        // Calculate how much base currency we can buy with the percentage of available quote currency
        const quoteAmount = availableAmount * (percentage / 100);
        const baseAmount = quoteAmount / price;
        
        this.elements.amountInput.value = baseAmount.toFixed(8);
      }
    } else {
      // When selling, we use base currency (e.g., BTC)
      availableAmount = this.options.availableBalance[baseCurrency] || 0;
      
      // Calculate the amount based on the percentage
      const amount = availableAmount * (percentage / 100);
      this.elements.amountInput.value = amount.toFixed(8);
    }
    
    this.calculateTotal();
  }
  
  /**
   * Update the percentage slider based on the amount input
   */
  updatePercentageFromAmount() {
    const amount = parseFloat(this.elements.amountInput.value) || 0;
    const pair = this.currentPair.split('/');
    const baseCurrency = pair[0]; // e.g., BTC
    const quoteCurrency = pair[1]; // e.g., USDT
    
    let availableAmount, percentage;
    
    if (this.currentSide === 'buy') {
      // When buying, we need to convert the base amount to quote amount
      let price;
      if (this.currentType === 'market') {
        price = 60000; // Placeholder BTC price
      } else {
        price = parseFloat(this.elements.priceInput.value) || 0;
      }
      
      if (price > 0) {
        const quoteAmount = amount * price;
        availableAmount = this.options.availableBalance[quoteCurrency] || 0;
        percentage = (quoteAmount / availableAmount) * 100;
      } else {
        percentage = 0;
      }
    } else {
      // When selling, we directly use the base amount
      availableAmount = this.options.availableBalance[baseCurrency] || 0;
      percentage = (amount / availableAmount) * 100;
    }
    
    // Update slider (cap at 100%)
    percentage = Math.min(percentage, 100);
    this.elements.percentageSlider.value = percentage;
  }
  
  /**
   * Validate stop loss and take profit inputs
   */
  validateStopLossAndTakeProfit() {
    const stopLoss = parseFloat(this.elements.stopLossInput.value) || 0;
    const takeProfit = parseFloat(this.elements.takeProfitInput.value) || 0;
    let price;
    
    if (this.currentType === 'market') {
      price = 60000; // Placeholder market price
    } else {
      price = parseFloat(this.elements.priceInput.value) || 0;
    }
    
    let errorMessage = '';
    
    if (this.currentSide === 'buy') {
      // For buy orders, stop loss should be below price and take profit above
      if (stopLoss > 0 && stopLoss >= price) {
        errorMessage = 'Stop loss must be below the entry price for buy orders';
      } else if (takeProfit > 0 && takeProfit <= price) {
        errorMessage = 'Take profit must be above the entry price for buy orders';
      }
    } else {
      // For sell orders, stop loss should be above price and take profit below
      if (stopLoss > 0 && stopLoss <= price) {
        errorMessage = 'Stop loss must be above the entry price for sell orders';
      } else if (takeProfit > 0 && takeProfit >= price) {
        errorMessage = 'Take profit must be below the entry price for sell orders';
      }
    }
    
    // Display error message if any
    if (errorMessage) {
      this.elements.errorMessage.textContent = errorMessage;
      this.elements.errorMessage.style.display = 'block';
    } else {
      this.elements.errorMessage.style.display = 'none';
    }
    
    return !errorMessage;
  }
  
  /**
   * Submit the order
   */
  submitOrder() {
    // Validate inputs
    const amount = parseFloat(this.elements.amountInput.value) || 0;
    if (amount <= 0) {
      this.showError('Please enter a valid amount');
      return;
    }
    
    if (this.currentType !== 'market') {
      const price = parseFloat(this.elements.priceInput.value) || 0;
      if (price <= 0) {
        this.showError('Please enter a valid price');
        return;
      }
    }
    
    // Validate stop loss and take profit
    if (!this.validateStopLossAndTakeProfit()) {
      return;
    }
    
    // Calculate total
    const total = this.calculateTotal();
    
    // Check if user has enough balance
    const pair = this.currentPair.split('/');
    const baseCurrency = pair[0]; // e.g., BTC
    const quoteCurrency = pair[1]; // e.g., USDT
    
    if (this.currentSide === 'buy') {
      const availableQuote = this.options.availableBalance[quoteCurrency] || 0;
      if (total > availableQuote) {
        this.showError(`Insufficient ${quoteCurrency} balance`);
        return;
      }
    } else {
      const availableBase = this.options.availableBalance[baseCurrency] || 0;
      if (amount > availableBase) {
        this.showError(`Insufficient ${baseCurrency} balance`);
        return;
      }
    }
    
    // Prepare order data
    const orderData = {
      pair: this.currentPair,
      side: this.currentSide,
      type: this.currentType,
      amount: amount,
      price: this.currentType === 'market' ? null : parseFloat(this.elements.priceInput.value) || 0,
      stopLoss: parseFloat(this.elements.stopLossInput.value) || null,
      takeProfit: parseFloat(this.elements.takeProfitInput.value) || null,
      leverage: this.elements.leverageContainer.style.display === 'block' ? 
        parseInt(this.elements.leverageSlider.value) || 1 : 1,
      total: total
    };
    
    // Call the onSubmit callback if provided
    if (typeof this.options.onSubmit === 'function') {
      this.options.onSubmit(orderData);
    }
    
    // Show success message
    this.showSuccess(`${this.currentSide.toUpperCase()} order placed successfully`);
    
    // Reset form
    this.resetForm();
  }
  
  /**
   * Show error message
   * @param {string} message - Error message
   */
  showError(message) {
    this.elements.errorMessage.textContent = message;
    this.elements.errorMessage.style.display = 'block';
    this.elements.errorMessage.classList.add('error');
    this.elements.errorMessage.classList.remove('success');
    
    // Hide after 5 seconds
    setTimeout(() => {
      this.elements.errorMessage.style.display = 'none';
    }, 5000);
  }
  
  /**
   * Show success message
   * @param {string} message - Success message
   */
  showSuccess(message) {
    this.elements.errorMessage.textContent = message;
    this.elements.errorMessage.style.display = 'block';
    this.elements.errorMessage.classList.add('success');
    this.elements.errorMessage.classList.remove('error');
    
    // Hide after 5 seconds
    setTimeout(() => {
      this.elements.errorMessage.style.display = 'none';
    }, 5000);
  }
  
  /**
   * Reset the form
   */
  resetForm() {
    this.elements.amountInput.value = '';
    this.elements.priceInput.value = '';
    this.elements.stopLossInput.value = '';
    this.elements.takeProfitInput.value = '';
    this.elements.percentageSlider.value = 0;
    this.elements.leverageSlider.value = 1;
    this.elements.leverageValue.textContent = '1x';
    
    this.calculateTotal();
  }
  
  /**
   * Update the current trading pair
   * @param {string} pair - Trading pair (e.g., BTC/USDT)
   */
  updatePair(pair) {
    this.currentPair = pair;
    
    // Update currency labels
    const baseCurrency = pair.split('/')[0];
    const quoteCurrency = pair.split('/')[1];
    
    const amountCurrency = this.container.querySelector('.input-group-text');
    if (amountCurrency) {
      amountCurrency.textContent = baseCurrency;
    }
    
    const priceCurrency = this.container.querySelectorAll('.input-group-text')[1];
    if (priceCurrency) {
      priceCurrency.textContent = quoteCurrency;
    }
    
    // Update available balance
    this.updateAvailableBalance();
    
    // Reset form
    this.resetForm();
  }
}

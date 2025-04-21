// Package notification provides implementations for various notification services
package notification

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"slices"

	"github.com/raykavin/backnrun/core"
	"github.com/raykavin/backnrun/exchange"
	"github.com/raykavin/backnrun/order"
	tb "gopkg.in/tucnak/telebot.v2"
)

// Constants and regex patterns
const (
	pollingTimeout = 10 * time.Second
)

var (
	buyRegexp  = regexp.MustCompile(`/buy\s+(?P<pair>\w+)\s+(?P<amount>\d+(?:\.\d+)?)(?P<percent>%)?`)
	sellRegexp = regexp.MustCompile(`/sell\s+(?P<pair>\w+)\s+(?P<amount>\d+(?:\.\d+)?)(?P<percent>%)?`)
)

// Telegram implements the core.NotifierWithStart interface
type Telegram struct {
	settings        *core.Settings
	orderController *order.Controller
	defaultMenu     *tb.ReplyMarkup
	client          *tb.Bot
	log             core.Logger
}

// Option is a function that configures a telegram instance
type Option func(telegram *Telegram)

// NewTelegram creates and initializes a new Telegram service
func NewTelegram(controller *order.Controller, settings *core.Settings, log core.Logger, options ...Option) (
	core.NotifierWithStart,
	error,
) {
	menu := &tb.ReplyMarkup{ResizeReplyKeyboard: true}
	poller := &tb.LongPoller{Timeout: pollingTimeout}
	userMiddleware := newAuthMiddleware(poller, settings, log)

	client, err := initializeBotClient(settings, userMiddleware)
	if err != nil {
		return nil, err
	}

	setupKeyboard(menu)
	if err := setupCommands(client); err != nil {
		return nil, fmt.Errorf("failed to set commands: %w", err)
	}

	bot := &Telegram{
		orderController: controller,
		client:          client,
		settings:        settings,
		defaultMenu:     menu,
		log:             log,
	}

	// Apply custom options if provided
	for _, option := range options {
		option(bot)
	}

	registerHandlers(client, bot)

	return bot, nil
}

// initializeBotClient creates and configures the Telegram bot client
func initializeBotClient(settings *core.Settings, middleware *tb.MiddlewarePoller) (*tb.Bot, error) {
	client, err := tb.NewBot(tb.Settings{
		ParseMode: tb.ModeMarkdown,
		Token:     settings.Telegram.Token,
		Poller:    middleware,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	return client, nil
}

// newAuthMiddleware creates a middleware to validate authorized users
func newAuthMiddleware(poller *tb.LongPoller, settings *core.Settings, log core.Logger) *tb.MiddlewarePoller {
	return tb.NewMiddlewarePoller(poller, func(u *tb.Update) bool {
		if u.Message == nil || u.Message.Sender == nil {
			log.Error("message or sender is nil ", u)
			return false
		}

		if slices.Contains(settings.Telegram.Users, int(u.Message.Sender.ID)) {
			return true
		}

		log.Error("unauthorized user ", u.Message.Sender.ID)
		return false
	})
}

// setupKeyboard configures the reply keyboard layout
func setupKeyboard(menu *tb.ReplyMarkup) {
	// Define keyboard buttons
	var (
		statusBtn  = menu.Text("/status")
		profitBtn  = menu.Text("/profit")
		balanceBtn = menu.Text("/balance")
		startBtn   = menu.Text("/start")
		stopBtn    = menu.Text("/stop")
		buyBtn     = menu.Text("/buy")
		sellBtn    = menu.Text("/sell")
	)

	// Arrange keyboard layout
	menu.Reply(
		menu.Row(statusBtn, balanceBtn, profitBtn),
		menu.Row(startBtn, stopBtn, buyBtn, sellBtn),
	)
}

// setupCommands configures available bot commands
func setupCommands(client *tb.Bot) error {
	return client.SetCommands([]tb.Command{
		{Text: "/help", Description: "Display help instructions"},
		{Text: "/stop", Description: "Stop buy and sell coins"},
		{Text: "/start", Description: "Start buy and sell coins"},
		{Text: "/status", Description: "Check bot status"},
		{Text: "/balance", Description: "Wallet balance"},
		{Text: "/profit", Description: "Summary of last trade results"},
		{Text: "/buy", Description: "Open a buy order"},
		{Text: "/sell", Description: "Open a sell order"},
	})
}

// registerHandlers registers all command handlers
func registerHandlers(client *tb.Bot, bot *Telegram) {
	client.Handle("/help", bot.HelpHandle)
	client.Handle("/start", bot.StartHandle)
	client.Handle("/stop", bot.StopHandle)
	client.Handle("/status", bot.StatusHandle)
	client.Handle("/balance", bot.BalanceHandle)
	client.Handle("/profit", bot.ProfitHandle)
	client.Handle("/buy", bot.BuyHandle)
	client.Handle("/sell", bot.SellHandle)
}

// Start begins the Telegram bot and notifies all authorized users
func (t *Telegram) Start() {
	go t.client.Start()
	t.sendMessageWithOptions("Bot initialized.", t.defaultMenu)
}

// Notification methods
// -------------------

// Notify sends a message to all authorized users
func (t *Telegram) Notify(text string) {
	for _, user := range t.settings.Telegram.Users {
		_, err := t.client.Send(&tb.User{ID: int64(user)}, text)
		if err != nil {
			t.log.WithError(err).Error("failed to send notification")
		}
	}
}

// sendMessageWithOptions sends a message to all authorized users with additional options
func (t *Telegram) sendMessageWithOptions(text string, options ...any) {
	for _, user := range t.settings.Telegram.Users {
		_, err := t.client.Send(&tb.User{ID: int64(user)}, text, options...)
		if err != nil {
			t.log.WithError(err).Error("failed to send notification with options")
		}
	}
}

// sendMessage sends a message to a specific user
func (t *Telegram) sendMessage(to *tb.User, text string, options ...any) {
	_, err := t.client.Send(to, text, options...)
	if err != nil {
		t.log.WithError(err).Error("failed to send message")
	}
}

// Command handlers
// ---------------

// BalanceHandle shows the balance of all assets
func (t *Telegram) BalanceHandle(m *tb.Message) {
	account, err := t.orderController.Account(context.Background())
	if err != nil {
		t.log.WithError(err).Error("failed to get account")
		t.OnError(err)
		return
	}

	message, err := t.formatBalanceMessage(account)
	if err != nil {
		t.OnError(err)
		return
	}

	t.sendMessage(m.Sender, message)
}

// HelpHandle displays available commands
func (t *Telegram) HelpHandle(m *tb.Message) {
	commands, err := t.client.GetCommands()
	if err != nil {
		t.log.WithError(err).Error("failed to get commands")
		t.OnError(err)
		return
	}

	lines := make([]string, 0, len(commands))
	for _, command := range commands {
		lines = append(lines, fmt.Sprintf("/%s - %s", command.Text, command.Description))
	}

	t.sendMessage(m.Sender, strings.Join(lines, "\n"))
}

// ProfitHandle shows trading results
func (t *Telegram) ProfitHandle(m *tb.Message) {
	if len(t.orderController.Results) == 0 {
		t.sendMessage(m.Sender, "No trades registered.")
		return
	}

	for pair, summary := range t.orderController.Results {
		t.sendMessage(m.Sender, fmt.Sprintf("*PAIR*: `%s`\n`%s`", pair, summary.String()))
	}
}

// BuyHandle processes buy commands
func (t *Telegram) BuyHandle(m *tb.Message) {
	match := buyRegexp.FindStringSubmatch(m.Text)
	if len(match) == 0 {
		t.sendMessage(m.Sender, "Invalid command.\nExamples of usage:\n`/buy BTCUSDT 100`\n\n`/buy BTCUSDT 50%`")
		return
	}

	err := t.processBuyOrder(m.Sender, match)
	if err != nil {
		t.OnError(err)
	}
}

// SellHandle processes sell commands
func (t *Telegram) SellHandle(m *tb.Message) {
	match := sellRegexp.FindStringSubmatch(m.Text)
	if len(match) == 0 {
		t.sendMessage(m.Sender, "Invalid command.\nExample of usage:\n`/sell BTCUSDT 100`\n\n`/sell BTCUSDT 50%`")
		return
	}

	err := t.processSellOrder(m.Sender, match)
	if err != nil {
		t.OnError(err)
	}
}

// StatusHandle displays the current bot status
func (t *Telegram) StatusHandle(m *tb.Message) {
	status := t.orderController.Status()
	t.sendMessage(m.Sender, fmt.Sprintf("Status: `%s`", status))
}

// StartHandle starts the bot operation
func (t *Telegram) StartHandle(m *tb.Message) {
	if t.orderController.Status() == order.StatusRunning {
		t.sendMessage(m.Sender, "Bot is already running.", t.defaultMenu)
		return
	}

	t.orderController.Start(context.Background())
	t.sendMessage(m.Sender, "Bot started.", t.defaultMenu)
}

// StopHandle stops the bot operation
func (t *Telegram) StopHandle(m *tb.Message) {
	if t.orderController.Status() == order.StatusStopped {
		t.sendMessage(m.Sender, "Bot is already stopped.", t.defaultMenu)
		return
	}

	t.orderController.Stop(context.Background())
	t.sendMessage(m.Sender, "Bot stopped.", t.defaultMenu)
}

// Event handlers
// -------------

// OnOrder notifies users about order status changes
func (t *Telegram) OnOrder(order core.Order) {
	title := t.getOrderStatusTitle(order)
	message := fmt.Sprintf("%s\n-----\n%s", title, order)
	t.Notify(message)
}

// getOrderStatusTitle returns a formatted title based on order status
func (t *Telegram) getOrderStatusTitle(order core.Order) string {
	switch order.Status {
	case core.OrderStatusTypeFilled:
		return fmt.Sprintf("✅ ORDER FILLED - %s", order.Pair)
	case core.OrderStatusTypeNew:
		return fmt.Sprintf("🆕 NEW ORDER - %s", order.Pair)
	case core.OrderStatusTypeCanceled, core.OrderStatusTypeRejected:
		return fmt.Sprintf("❌ ORDER CANCELED / REJECTED - %s", order.Pair)
	default:
		return fmt.Sprintf("ORDER UPDATE - %s", order.Pair)
	}
}

// OnError notifies users about errors
func (t *Telegram) OnError(err error) {
	var sb strings.Builder
	sb.WriteString("🛑 ERROR\n")

	var orderError *exchange.OrderError
	if errors.As(err, &orderError) {
		t.formatOrderError(&sb, orderError)
		t.Notify(sb.String())
		return
	}

	sb.WriteString("-----\n")
	sb.WriteString(err.Error())

	t.Notify(sb.String())
}

// Helper methods
// -------------

// formatOrderError formats order-specific errors
func (t *Telegram) formatOrderError(sb *strings.Builder, orderError *exchange.OrderError) {
	sb.WriteString("-----\n")
	fmt.Fprintf(sb, "Pair: %s\n", orderError.Pair)
	fmt.Fprintf(sb, "Quantity: %.4f\n", orderError.Quantity)
	sb.WriteString("-----\n")
	sb.WriteString(orderError.Err.Error())
}

// formatBalanceMessage creates a formatted balance message
func (t *Telegram) formatBalanceMessage(account core.Account) (string, error) {
	message := "*BALANCE*\n"
	quotesValue := make(map[string]float64)
	total := 0.0

	// Calculate balance for each pair
	for _, pair := range t.settings.Pairs {
		assetPair, quotePair := exchange.SplitAssetQuote(pair)
		assetBalance, quoteBalance := account.GetBalance(assetPair, quotePair)

		assetSize := assetBalance.Free + assetBalance.Lock
		quoteSize := quoteBalance.Free + quoteBalance.Lock

		quote, err := t.orderController.LastQuote(pair)
		if err != nil {
			return "", fmt.Errorf("failed to get last quote for %s: %w", pair, err)
		}

		assetValue := assetSize * quote
		quotesValue[quotePair] = quoteSize
		total += assetValue
		message += fmt.Sprintf("%s: `%.4f` ≅ `%.2f` %s \n", assetPair, assetSize, assetValue, quotePair)
	}

	// Add quote currencies to the total
	for quote, value := range quotesValue {
		total += value
		message += fmt.Sprintf("%s: `%.4f`\n", quote, value)
	}

	message += fmt.Sprintf("-----\nTotal: `%.4f`\n", total)
	return message, nil
}

// processBuyOrder handles the buy order creation logic
func (t *Telegram) processBuyOrder(sender *tb.User, match []string) error {
	command := extractCommandParams(buyRegexp, match)
	pair := strings.ToUpper(command["pair"])
	amount, err := strconv.ParseFloat(command["amount"], 64)
	if err != nil {
		return fmt.Errorf("failed to parse amount: %w", err)
	}

	if amount <= 0 {
		t.sendMessage(sender, "Invalid amount")
		return nil
	}

	// Handle percentage-based orders
	if command["percent"] != "" {
		amount, err = t.calculatePercentageQuoteAmount(pair, amount)
		if err != nil {
			return err
		}
	}

	// Create the buy order
	order, err := t.orderController.CreateOrderMarketQuote(
		context.Background(),
		core.SideTypeBuy,
		pair,
		amount,
	)

	if err != nil {
		return fmt.Errorf("failed to create buy order for %s: %w", pair, err)
	}

	t.log.Info("[TELEGRAM]: BUY ORDER CREATED: ", order)
	return nil
}

// processSellOrder handles the sell order creation logic
func (t *Telegram) processSellOrder(sender *tb.User, match []string) error {
	command := extractCommandParams(sellRegexp, match)
	pair := strings.ToUpper(command["pair"])
	amount, err := strconv.ParseFloat(command["amount"], 64)
	if err != nil {
		return fmt.Errorf("failed to parse amount: %w", err)
	}

	if amount <= 0 {
		t.sendMessage(sender, "Invalid amount")
		return nil
	}

	// Handle percentage-based orders
	if command["percent"] != "" {
		return t.createPercentageSellOrder(pair, amount)
	}

	// Create the sell order (quote-based)
	order, err := t.orderController.CreateOrderMarketQuote(context.Background(), core.SideTypeSell, pair, amount)
	if err != nil {
		return fmt.Errorf("failed to create quote-based sell order for %s: %w", pair, err)
	}

	t.log.Info("[TELEGRAM]: SELL ORDER CREATED: ", order)
	return nil
}

// calculatePercentageQuoteAmount calculates the quote amount based on a percentage
func (t *Telegram) calculatePercentageQuoteAmount(pair string, percentage float64) (float64, error) {
	_, quote, err := t.orderController.Position(context.Background(), pair)
	if err != nil {
		return 0, fmt.Errorf("failed to get position for %s: %w", pair, err)
	}

	return percentage * quote / 100.0, nil
}

// createPercentageSellOrder creates a sell order based on asset percentage
func (t *Telegram) createPercentageSellOrder(pair string, percentage float64) error {
	asset, _, err := t.orderController.Position(context.Background(), pair)
	if err != nil {
		return fmt.Errorf("failed to get position for %s: %w", pair, err)
	}

	amount := percentage * asset / 100.0
	order, err := t.orderController.CreateOrderMarket(context.Background(), core.SideTypeSell, pair, amount)
	if err != nil {
		return fmt.Errorf("failed to create percentage-based sell order for %s: %w", pair, err)
	}

	t.log.Info("[TELEGRAM]: SELL ORDER CREATED: ", order)
	return nil
}

// Helper function to extract named groups from regex matches
func extractCommandParams(regex *regexp.Regexp, match []string) map[string]string {
	command := make(map[string]string)
	for i, name := range regex.SubexpNames() {
		if i != 0 && name != "" {
			command[name] = match[i]
		}
	}
	return command
}

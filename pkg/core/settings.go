package core

// Settings represents the main configuration for the application
type Settings struct {
	Pairs    []string         // List of trading pairs to monitor
	Telegram TelegramSettings // Telegram notification settings
}

// TelegramSettings holds configuration for Telegram integration
type TelegramSettings struct {
	Enabled bool   // Whether Telegram notifications are enabled
	Token   string // Telegram bot token
	Users   []int  // List of authorized user IDs
}

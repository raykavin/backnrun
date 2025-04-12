package core

type Settings interface {
	GetPairs() []string
	GetTelegram() TelegramSettings
}

type TelegramSettings interface {
	IsEnabled() bool
	GetToken() string
	GetUsers() []int
}

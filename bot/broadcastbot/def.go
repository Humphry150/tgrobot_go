package bot

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

type botClient struct {
	api    *tgbotapi.BotAPI
	config *botConfig
}

type botConfig struct {
	Token             string   `toml:"token"`
	BroadcastChats    []int64  `toml:"boardcast_chats"`
	BroadcastInterval int64    `toml:"broadcast_interval"`
	Text              []string `toml:"text"`
}

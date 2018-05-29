package bot

import (
	"math/rand"
	"time"

	"bitbucket.org/magmeng/go-utils/log"
	"github.com/BurntSushi/toml"
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

func decodeConfig(path string) botConfig {
	var c botConfig
	_, err := toml.DecodeFile(path, &c)
	if err != nil {
		panic(err)
	}
	return c
}

func newBot(configPath string) *botClient {
	var b botClient
	config := decodeConfig(configPath)
	b.config = &config
	return &b
}

func (b *botClient) broadcast() {
	broadcastTicker := time.NewTicker(time.Duration(b.config.BroadcastInterval) * time.Second)
	rand.Seed(time.Now().Unix())
	n := rand.Intn(len(b.config.Text))
	for {
		for _, chat := range b.config.BroadcastChats {
			sendTextMessage(b.api, chat, b.config.Text[n], 0)
		}
		n++
		if n >= len(b.config.Text) {
			n -= len(b.config.Text)
		}
		<-broadcastTicker.C
	}
}

func (b *botClient) getUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updatesChan, err := b.api.GetUpdatesChan(u)
	if err != nil {
		panic(err)
	}

	for update := range updatesChan {
		if update.Message == nil {
			continue
		}
		log.Infofln("message: %+#v", *update.Message)
	}
}

var exitChan chan struct{}

func Run(configPath string) {
	b := newBot(configPath)
	api, err := tgbotapi.NewBotAPI(b.config.Token)
	if err != nil {
		panic(err)
	}
	b.api = api
	go b.getUpdates()
	go b.broadcast()
	<-exitChan
}

package main

import (
	"flag"

	"bitbucket.org/magmeng/hotbot/bot/broadcastbot"
)

var configPath = flag.String("c", "./config/broadcastbot.toml", "config")

func main() {
	flag.Parse()
	bot.Run(*configPath)
}

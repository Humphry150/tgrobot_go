package main

import (
	"flag"

	"bitbucket.org/magmeng/hotbot/bot/infobot"
)

func main() {
	flag.Parse()
	bot.Run("./config/infobot.toml")
}

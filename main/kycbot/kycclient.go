package main

import (
	"flag"

	"bitbucket.org/magmeng/hotbot/bot/kycbot"
)

var confPath = flag.String("c", "./config/kycbot.toml", "config")

func main() {
	flag.Parse()
	bot.NewBot(*confPath).Serve()
}

package main

import (
	"flag"

	"bitbucket.org/magmeng/hotbot/bot/mainbot"
)

var configPath = flag.String("c", "./config/coinhot.toml", "配置")

func main() {
	flag.Parse()
	bot.Run(*configPath)
}

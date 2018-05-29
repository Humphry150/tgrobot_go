package bot

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

func priceMethod(api *tgbotapi.BotAPI, message *tgbotapi.Message) {
	sendTextMessage(
		api,
		message.Chat.ID,
		price(),
		time.Duration(conf.DeleteBotMsgDelay)*time.Second,
		false,
		message.MessageID,
	)
}

func dividendsMethod(api *tgbotapi.BotAPI, message *tgbotapi.Message) {
	sendTextMessage(
		api,
		message.Chat.ID,
		dividends(),
		time.Duration(conf.DeleteBotMsgDelay)*time.Second,
		false,
		message.MessageID,
	)
}

func bindMethod(api *tgbotapi.BotAPI, message *tgbotapi.Message, helpText string) {
	args := strings.Split(message.CommandArguments(), " ")
	if len(args) < 1 || args[0] == "" {
		sendTextMessage(
			api,
			message.Chat.ID,
			helpText,
			time.Duration(conf.DeleteBotMsgDelay)*time.Second,
			false,
			message.MessageID,
		)
		return
	}
	text := bind(args[0], fmt.Sprintf("%d", message.From.ID), message.From.UserName, message.From.FirstName, message.From.LastName) + "\n\n(本消息2分钟后自动删除)"
	sendTextMessage(
		api,
		message.Chat.ID,
		text,
		time.Second*120,
		false,
		message.MessageID,
	)
}

func bindqMethod(api *tgbotapi.BotAPI, message *tgbotapi.Message, helpText string) {
	var text string
	reply := message.ReplyToMessage
	replyID := message.MessageID
	uid := message.From.ID

	if isUserInWhiteList(message.From.ID) && reply != nil {
		DebugLog("white list")
		replyID = reply.MessageID
		uid = reply.From.ID
	}

	text = bindq(fmt.Sprintf("%d", uid)) + "\n\n(本消息2分钟后自动删除)"
	sendTextMessage(
		api,
		message.Chat.ID,
		text,
		time.Second*120,
		false,
		replyID,
	)
}

func setAddressMethod(api *tgbotapi.BotAPI, message *tgbotapi.Message, helpText string) {
	if isChatToManage(message.Chat) {
		sendTextMessage(
			api,
			message.Chat.ID,
			"为了保护您的隐私，请私聊本机器宝宝并使用命令'/setaddress'来设置您的接受奖励地址，您的这条消息本机器宝宝帮您删除了。\n\n（本消息2分钟后自动删除）",
			time.Second*120,
			false,
			message.MessageID,
		)
		deleteMsg(api, message)
		return
	}

	if isChatGroup(message.Chat) {
		return
	}

	// 返回命令的 help
	args := strings.Split(message.CommandArguments(), " ")
	if len(args) < 1 || args[0] == "" {
		sendTextMessage(
			api,
			message.Chat.ID,
			helpText,
			time.Duration(conf.DeleteBotMsgDelay)*time.Second,
			false,
			message.MessageID,
		)
		return
	}

	// 检查地址
	address := args[0]
	if !isEthereumAddress(address) {
		text := fmt.Sprintf("请确保您的地址是一个标准的以太坊地址（标准以太坊地址若开头有'0x'，则总长度为%d）。推荐使用 [DragonEX](https://dragonex.im) 平台充值地址。", len("0x5a0b54d5dc17e0aadc383d2db43b0a0d3e029c4c"))
		sendTextMessage(
			api,
			message.Chat.ID,
			text,
			0,
			false,
			message.MessageID,
		)
		return
	}

	// 返回调用结果
	text := setAddress(fmt.Sprintf("%d", message.From.ID), address)
	sendTextMessage(
		api,
		message.Chat.ID,
		text,
		0,
		false,
		message.MessageID,
	)
}

func unbindMethod(api *tgbotapi.BotAPI, message *tgbotapi.Message, helpText string) {
	text := unbind(fmt.Sprintf("%d", message.From.ID)) + "\n\n(本消息2分钟后自动删除)"
	sendTextMessage(
		api,
		message.Chat.ID,
		text,
		time.Second*120,
		false,
		message.MessageID,
	)
}

func rollMethod(api *tgbotapi.BotAPI, message *tgbotapi.Message, helpText string) {
	args := commandArgs(message)

	if !isRollerChat(message.Chat.ID) {
		if len(args) < 1 {
			return
		}

		if !isRollerAdmin(message.From.ID) {
			return
		}

		if isChatGroup(message.Chat) {
			return
		}

		switch args[0] {
		case "set":
			var rules rollerRules
			if len(args) > 1 {
				data, err := json.RawMessage(args[1]).MarshalJSON()
				if err != nil {
					Errorf("set roller config failed. error: %v", err)
					sendTextMessage(api, message.Chat.ID, "摇奖配置失败", 0, false, message.MessageID)
					return
				}

				err = json.Unmarshal(data, &rules)
				if err != nil {
					Errorf("set roller config failed. error: %v", err)
					sendTextMessage(api, message.Chat.ID, "摇奖配置失败", 0, false, message.MessageID)
					return
				}

				conf.RollerConfig.Rules = &rules
				sendTextMessage(api, message.Chat.ID, "摇奖配置成功。", 0, false, message.MessageID)
			}

		case "read":
			if conf.RollerConfig.Rules == nil {
				sendTextMessage(api, int64(message.From.ID), "没有摇奖配置。", 0, false, message.MessageID)
				return
			}

			data, _ := json.Marshal(conf.RollerConfig.Rules)
			sendTextMessage(api, int64(message.From.ID), fmt.Sprintf("摇奖配置：`%s`", data), 0, false, message.MessageID)

		default:
			return
		}
	}

	switch {
	case len(args) < 1 || args[0] == "":
		// 普通 roll
		r := getRollGame(message.Chat.ID)
		if r != nil {
			sendTextMessage(
				api,
				message.Chat.ID,
				r.roll(message.From),
				0,
				false,
				message.MessageID,
			)
		} else {
			var announcement string
			if conf.RollerConfig.Rules != nil {
				announcement = conf.RollerConfig.Rules.GeneralRules.Announcement
			}
			sendTextMessage(
				api,
				message.Chat.ID,
				"活动还未开始。\n\n"+announcement,
				time.Second*30,
				false,
				message.MessageID,
			)
		}

	default:
		if !isRollerAdmin(message.From.ID) {
			return
		}

		switch args[0] {
		case "start":
			// 开始 roll
			startRoller(message.Chat.ID, message.MessageID, api)
		case "stop":
			// 结束 roll
			stopRoller(api, message.Chat.ID)
		}

	}

}

func contractMethod(api *tgbotapi.BotAPI, message *tgbotapi.Message, helpText string) {
	sendTextMessage(
		api,
		message.Chat.ID,
		contractRequest(),
		0,
		false,
		message.MessageID,
	)
}

// func frozenBalanceMethod(api *tgbotapi.BotAPI, message *tgbotapi.Message, helpText string) {
// 	sendTextMessage(
// 		api,
// 		message.Chat.ID,
// 		fronzenBalanceRequest(),
// 		0,
// 		message.MessageID,
// 	)
// }

func muteMethod(api *tgbotapi.BotAPI, msg *tgbotapi.Message, helpText string) {
	if !isChatGroup(msg.Chat) {
		return
	}

	if !isUserInWhiteList(msg.From.ID) {
		return
	}

	if msg.ReplyToMessage == nil {
		return
	}

	args := commandArgs(msg)
	var hours int64
	if len(args) != 0 {
		hours, _ = strconv.ParseInt(args[0], 0, 64)
	}

	if hours == 0 {
		hours = 24
	}
	mute(api, msg, hours)
}

func getMethod(api *tgbotapi.BotAPI, msg *tgbotapi.Message, helpText string) {
	if !isChatGroup(msg.Chat) {
		return
	}

	args := commandArgs(msg)
	if len(args) < 1 {
		sendTextMessage(api, msg.Chat.ID, helpText, 0, false, msg.MessageID)
		return
	}

	var rewardType string
	switch args[0] {
	case "注册奖励":
		rewardType = "register"
	case "交易奖励":
		rewardType = "trade"
	case "邀请奖励":
		rewardType = "invitation"
	// case "加入龙网群奖励":
	// 	rewardType = "telegram_group"
	case "实名奖励":
		rewardType = "identification"
	default:
		msg.Text = "/领取"
		instruction := botCommandInstruction(msg, helpText)
		sendTextMessage(api, msg.Chat.ID, instruction, 0, false, msg.MessageID)
		return
	}

	sendTextMessage(api, msg.Chat.ID, getRewardRequest(msg.From.ID, rewardType), 0, false, msg.MessageID)
}

// func sendCommandText(api *tgbotapi.BotAPI, message *tgbotapi.Message, helpText string) {
// 	args := strings.Split(message.CommandArguments(), " ")
// if len(c.Args) == 0 && (len(args) == 0 || args[0] == "") || (len(args) != 0 && len(c.Args) != 0 && strings.ToLower(args[0]) == strings.ToLower(c.Args[0])) {
// 		sendTextMessage(api, c.Text, message.Chat.ID, time.Duration(conf.DeleteBotMsgDelay)*time.Second, message.MessageID)
// 	}
// }

func botCommandHelp() string {
	for _, cmd := range conf.Commands {
		if equalString(cmd.Command, "help") {
			return cmd.Text
		}
	}
	panic("help text not found")
}

func (b BotCommand) instruction(msg *tgbotapi.Message) string {
	args := commandArgs(msg)
	if len(b.Args) == 0 && (len(args) == 0 || len(args) == 1 && args[0] == "") || len(args) != 0 && len(b.Args) != 0 && equalString(args[0], b.Args[0]) {
		return b.Text
	}
	return ""
}

func botCommandInstruction(message *tgbotapi.Message, helpText string) string {
	for _, cmd := range conf.Commands {
		if !equalString(cmd.Command, getCommand(message)) {
			continue
		}

		if ins := cmd.instruction(message); ins != "" {
			return ins
		}
	}
	return helpText
}

func commandArgs(msg *tgbotapi.Message) []string {
	var args []string
	if msg.IsCommand() {
		args = strings.Split(msg.CommandArguments(), " ")
	} else {
		commands := strings.Split(msg.Text, " ")
		if len(commands) > 1 {
			args = commands[1:]
		}
	}
	var resp []string
	for _, arg := range args {
		if arg != "" {
			resp = append(resp, arg)
		}
	}
	return resp
}

func isMethod(msg *tgbotapi.Message) bool {
	for _, cmd := range conf.Commands {
		if equalString(cmd.Command, getCommand(msg)) {
			return equalString(cmd.Type, "method")
		}
	}
	return false
}

func isCommand(message *tgbotapi.Message) bool {
	return message.IsCommand() || strings.HasPrefix(strings.Split(message.Text, " ")[0], "/")
}

func getCommand(message *tgbotapi.Message) string {
	if message.IsCommand() {
		return message.Command()
	}

	return strings.TrimPrefix(strings.Split(message.Text, " ")[0], "/")
}

func handleInstantCommand(api *tgbotapi.BotAPI, message *tgbotapi.Message, helpText string) {
	instruction := botCommandInstruction(message, helpText)

	if isCommand(message) {
		if isMethod(message) {
			switch getCommand(message) {
			case "price":
				priceMethod(api, message)
			case "dividends":
				dividendsMethod(api, message)
			case "bind":
				bindMethod(api, message, instruction)
			case "bindq":
				bindqMethod(api, message, instruction)
				// 此方法只对私聊使用
			case "setaddress":
				setAddressMethod(api, message, instruction)
			case "unbind":
				unbindMethod(api, message, instruction)
			case "roll":
				rollMethod(api, message, instruction)
			case "contract":
				contractMethod(api, message, helpText)
			case "mute":
				muteMethod(api, message, helpText)
			case "领取":
				getMethod(api, message, instruction)
			default:
				sendTextMessage(
					api,
					message.Chat.ID,
					helpText,
					time.Duration(conf.DeleteBotMsgDelay)*time.Second,
					false,
					message.MessageID,
				)
			}
		} else {
			sendTextMessage(
				api,
				message.Chat.ID,
				instruction,
				0,
				false,
				message.MessageID,
			)
		}
	}

}

func handleQueueingCommand(api *tgbotapi.BotAPI, msg *tgbotapi.Message, helpText string) string {
	instruction := botCommandInstruction(msg, helpText)
	switch msg.Command() {
	// case "roll":
	// return roll(rollingInfo{uid: msg.From.ID, username: msg.From.UserName, firstname: msg.From.FirstName, lastname: msg.From.LastName, gid: msg.Chat.ID})
	default:
		return instruction
	}
}

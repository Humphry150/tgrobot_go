package bot

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"bitbucket.org/magmeng/go-utils/log"
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

func price(...string) string {
	text := "当前价格信息：\n\n"
	format := "|%6s|%12s|\n"
	line := fmt.Sprintf("%s\n", strings.Repeat("-", 21))
	var body string
	for _, coin := range []string{"cht", "btc", "eth"} {
		v, ok := coins[coin]
		if ok {
			body += fmt.Sprintf(format, strings.ToUpper(coin), fmt.Sprintf("%.4f", v.Price))
			body += line
		}
	}

	if body == "" {
		return "请稍后再查询"
	}

	head := line + fmt.Sprintf(format, "Coin", "Price(CNY)") + line
	return text + "`" + head + body + "`"
}

func dividends(...string) string {
	now := time.Now().In(BeiJing)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	btc, _ := big.NewFloat(0).SetString(dividendBTC)
	eth, _ := big.NewFloat(0).SetString(dividendETH)
	if dividendBTC == "" || dividendETH == "" {
		return fmt.Sprintf("分红数据尚未初始化，请稍后查询。查询分红规则请使用命令`/faq 分红规则`。")
	}
	if dividendDate.Before(today.AddDate(0, 0, -1)) {
		return fmt.Sprintf("%15s\n`» %.8fBTC\n» %.8fETH`\n\n今日分红数据在今日发放分红之后可查询到，以上为最近一次分红数据。查询分红规则请使用命令`/faq 分红规则`。", "每持有`10000CHT`可同时获得以下分红:", btc.Mul(btc, big.NewFloat(1e4)), eth.Mul(eth, big.NewFloat(1e4)))
	}
	return fmt.Sprintf("%15s\n`» %.8fBTC\n» %.8fETH`\n\n查询分红规则请使用命令`/faq 分红规则`。", "每持有`10000CHT`可同时获得以下分红:", btc.Mul(btc, big.NewFloat(1e4)), eth.Mul(eth, big.NewFloat(1e4)))
}

// 查询绑定
func bindq(args ...string) string {
	if len(args) < 1 {
		return "机器宝宝出故障了..."
	}

	uid := args[0]
	if uid == "" {
		return "此账号无法查询。"
	}
	status, address := queryBindRequest(uid)
	switch status {
	case BIND_STATUS_ALREADY_BIND:
		if address == "" {
			return "该账号已绑定，未设置接收奖励的地址。可以【私聊】[CoinHot机器宝宝](https://t.me/coinhot_bot)使用`/setaddress`设置接收活动奖励的地址。推荐使用 [DragonEX](https://dragonex.im) 平台充值地址。"
		}
		return "该账号已经绑定并且已设置接收奖励的地址。可以【私聊】[CoinHot机器宝宝](https://t.me/coinhot_bot)使用`/setaddress`修改您接收活动奖励的地址。推荐使用 [DragonEX](https://dragonex.im) 平台充值地址。"

	case BIND_STATUS_NOT_BIND:
		return "该账号还未绑定，请使用命令`/bind`查询如何绑定。"
	default:
		return "查询失败，请呼叫客服，或稍后再试。"
	}
}

func bind(args ...string) string {
	if len(args) < 5 {
		return "机器宝宝出故障了..."
	}

	code := args[0]
	uid := args[1]
	uName := args[2]
	firstname := args[3]
	lastname := args[4]
	status, address := queryBindRequest(uid)
	switch status {
	case BIND_STATUS_ALREADY_BIND:
		if address == "" {
			return "您已经绑定，无须再次绑定。另外，您可以【私聊】[CoinHot机器宝宝](https://t.me/coinhot_bot)使用`/setaddress`设置您接收活动奖励的地址。推荐使用 [DragonEX](https://dragonex.im) 平台充值地址。"
		}
		return "您已经绑定并且已经设置了接收奖励的地址，无须再次绑定账号。若要修改接收奖励的地址，请【私聊】[CoinHot机器宝宝](https://t.me/coinhot_bot)使用`/setaddress`修改您接收活动奖励的地址。推荐使用 [DragonEX](https://dragonex.im) 平台充值地址。"
	case BIND_STATUS_UNKOWN:
		return "暂时无法绑定，请联系客服。"
	default:
	}

	return bindUserRequest(code, uid, uName, firstname, lastname)
}

func unbind(args ...string) string {
	if len(args) < 1 {
		return "机器宝宝出故障了..."
	}

	uid := args[0]
	status, _ := queryBindRequest(uid)
	switch status {
	case BIND_STATUS_NOT_BIND:
		return "您未绑定，无须解绑。另外，您可以使用'/bind'来绑定您的Telegram账号和CoinHot账号。"
	case BIND_STATUS_UNKOWN:
		return "暂时无法解绑，请联系客服。"
	default:
	}

	return unbindUserRequest(uid)
}

func setAddress(args ...string) string {
	if len(args) < 2 {
		return "机器宝宝出故障了..."
	}

	uid := args[0]
	if uid == "" {
		return "无法为您设置地址。"
	}

	status, _ := queryBindRequest(uid)
	switch status {
	case BIND_STATUS_NOT_BIND:
		return "您还未绑定，请先绑定。\n查询如何绑定使用命令`/bind`"
	case BIND_STATUS_UNKOWN:
		return "暂时查询不到您的绑定信息，请稍后再试。"
	default:
	}

	return setAddressRequest(uid, args[1])
}

func (r RollGame) roll(user *tgbotapi.User) string {
	ru := r.GetRollingUser(user.ID)
	var count int
	if ru == nil {
		log.Infofln("new user")
		// 是否注册绑定
		if !isRegisteredMember(user.ID) {
			return "请先使用命令`/bind`绑定账号并使用命令`/setaddress`设置接收活动奖励地址后再参与活动。"
		}
		count = getUserRollingCount()
		ru = newRollerUser(r.ChatID, user, count)
		r.RollingUsers.Add(user.ID, ru)
	}

	log.Infofln("user: %+#v", ru)
	// 是否还有机会
	if ru.IsRollingCountRunningOut() {
		return ""
	}

	currentRoll := ru.Roll(r.MaxNum)
	currentResult := currentRoll.BroadcastString()
	currentCheck := ru.CheckCountingString()
	currentTip := choseRollerTips(currentRoll.Num)
	r.writeHistory()
	return strings.Join([]string{strings.Join([]string{currentCheck, currentResult}, "，"), currentTip}, "\n\n")
}

func mute(api *tgbotapi.BotAPI, msg *tgbotapi.Message, hours int64) {
	restrictUser(api, msg.Chat.ID, msg.ReplyToMessage.From.ID, hours)
	sendTextMessage(
		api,
		msg.Chat.ID,
		fmt.Sprintf("用户%s无视群规，被禁言%d小时。", userString(fmt.Sprintf("%d", msg.ReplyToMessage.From.ID), msg.ReplyToMessage.From.UserName, msg.ReplyToMessage.From.FirstName, msg.ReplyToMessage.From.LastName), hours),
		0,
		false,
	)
}

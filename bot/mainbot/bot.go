package bot

import (
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/redis.v4"

	"bitbucket.org/magmeng/go-utils/but4print"
	"bitbucket.org/magmeng/go-utils/log"
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

var coins map[string]*cryptoCurrency

// dividend
var dividendBTC, dividendETH string
var dividendDate time.Time

var conf BotConfig
var redisClient *redis.Client
var exitChan chan struct{}

var testOn = flag.Bool("test", false, "test mode")

type chatMessage struct {
	gid  int64
	msgs []*tgbotapi.Message
}

// 绑定用户云网帐号
func doBindingYW(api *tgbotapi.BotAPI, msg *tgbotapi.Message, code string) {
	//先查看本地缓存, 如果用户的电报号没有领取过,则进行领取
	if (!isYWUserRewarded(msg.Chat.ID, code)) {
		// 判断用户的code是否已经领取过
		if !isYWCodeExist(code) {
			// 这里请求网络, 然后返回接口文档
			errString := ywreceiveCode(code)
			log.Infof("errString is: ", errString)

			if len(errString) > 0 {
				var text string
				// 表示有错误
				switch errString {
				case "bad code":
					text = "验证失败，请检查所填CODE码是否正确"
				case "code used":
					text = "验证失败，该CODE码已被使用"
				case "balance not enouth":
					text = "本轮活动奖励已发放完毕，谢谢参与"
				case "verify_failed":
					text = "实名认证失败，请在实名认证通过后再提交CODE码领取奖励"
				case "verify_ing":
					text = "实名认证审核中，请在实名认证通过后再提交CODE码领取奖励"
				case "verify_not_sub":
					text = "该账号未进行实名认证，请在实名认证通过后再提交CODE码领取奖励[https://yunex.io/safe/auth/id]"
				//case "args":
				case "activity not ok":
					text = "活动异常"
				case "activity not begin or already end":
					text = "活动尚未开始或已结束，请留意官方公告"
				default:
					text = "系统错误"
				}
				sendTextMessage(api, msg.Chat.ID, text, 0, false)
			}else {
				sendTextMessage(api, msg.Chat.ID, "验证成功，奖励将在两天内发放至Yunex账号，请及时查收", 0, false)
				// 成功, 则记录数据
				recordYWRewardedUser(msg.Chat.ID, code)
			}
		}else {
			sendTextMessage(api, msg.Chat.ID, "验证失败，该CODE码已被使用", 0, false)
		}
	}else {
		//已经领取过, 直接回复文案 "该账号已领取过奖励，请勿重复领取"
		sendTextMessage(api, msg.Chat.ID, "该账号已领取过奖励，请勿重复领取\n", 0, false)
		record := redisClient.Get(strings.Join([]string{"YWCODE",toString(msg.Chat.ID), code}, "_")).Val()
		sendTextMessage(api, msg.Chat.ID, record,0, false)
	}
}

func isYWUserRewarded(chatID int64, code string) bool {
	return strings.Contains(redisClient.Get(strings.Join([]string{"YWCODE",toString(chatID)}, "_")).Val(), "recorded")
}

func isYWCodeExist(code string) bool {
	return redisClient.Get("YWCODE_" + code).Val() == code
}

func recordYWRewardedUser(chatID int64, code string) {
	redisClient.Set(strings.Join([]string{"YWCODE",toString(chatID)}, "_"), "recorded" + code, 0)
	redisClient.Set("YWCODE_" + code, code, 0)
}

// 将需要队列处理的消息全部丢进队列，机器人每隔一段时间对消息进行处理
func messageAligner(messageChan chan *tgbotapi.Message, api *tgbotapi.BotAPI, stopChan chan struct{}, helpText string) {
	// 用户消息去重
	userQueue := make(map[int]*tgbotapi.Message)
	var groupMessages []*chatMessage
	// 计时器
	t := time.NewTicker(time.Second * 2)

	handleMessageFunc := func() {
		// 统计不同群消息
		groupMessageMap := make(map[int64][]*tgbotapi.Message)
		for _, v := range userQueue {
			groupMessageMap[v.Chat.ID] = append(groupMessageMap[v.Chat.ID], v)
		}

		// 汇总
		for k, v := range groupMessageMap {
			var c chatMessage
			c.gid = k
			c.msgs = v
			groupMessages = append(groupMessages, &c)
		}

		// 执行
		for _, v := range groupMessages {
			var g sync.WaitGroup
			var msgs []string
			for _, gmsg := range v.msgs {
				g.Add(1)
				go func(v *chatMessage, msg *tgbotapi.Message) {
					m := handleQueueingCommand(api, msg, helpText)
					msgs = append(msgs, m)
					g.Done()
				}(v, gmsg)
			}
			g.Wait()
			sendTextMessage(api, v.gid, strings.Join(msgs, "\n\n"), 0, false)
		}

		// 初始化
		userQueue = make(map[int]*tgbotapi.Message)
		groupMessages = nil
	}

	for {
		select {
		case <-stopChan:
			handleMessageFunc()
			return
		case m := <-messageChan:
			if m != nil {
				userQueue[m.From.ID] = m
			}
		case <-t.C:
			handleMessageFunc()
		}
	}
}

func isEthereumAddress(address string) bool {
	addressExample := "0x5a0b54d5dc17e0aadc383d2db43b0a0d3e029c4c"
	lowerAddress := strings.ToLower(address)
	if !strings.HasPrefix(lowerAddress, "0x") {
		lowerAddress = "0x" + lowerAddress
	}

	return len(lowerAddress) == len(addressExample)
}

func validUserName(uid, uname, firstname, lastname string) string {
	name := firstname + lastname

	if name == "" {
		name = uname
	}

	if name == "" {
		name = uid
	}
	return name
}

func userString(uid, uname, firstname, lastname string) string {
	return fmt.Sprintf("[%s](tg://user?id=%s)", validUserName(uid, uname, firstname, lastname), uid)
}

func deleteUserMessage(api *tgbotapi.BotAPI, msg *tgbotapi.Message, msgType string) {
	text := fmt.Sprintf("用户：%s，消息类型：%s\n\n防止广告滋生，本群禁止发送各种奇怪文件或非法链接。", userString(fmt.Sprintf("%d", msg.From.ID), msg.From.UserName, msg.From.FirstName, msg.From.LastName), msgType)

	if msgType == "url" || msgType == "广告" {
		text += "\n\n由于推送广告，该用户暂时被禁言1天。\n"
	}
	sendTextMessage(api, msg.Chat.ID, text, time.Duration(conf.DeleteBotMsgDelay)*time.Second, false, msg.MessageID)
	deleteMsg(api, msg)
}

func welcomeMsg(members []tgbotapi.User, extraMsg string) string {
	msg := fmt.Sprintf("欢迎新朋友(Welcome new friends)\n")
	var membersSli []string
	for _, member := range members {
		if !member.IsBot {
			membersSli = append(membersSli, userString(fmt.Sprintf("%d", member.ID), member.UserName, member.FirstName, member.LastName))
		}
	}
	msg += strings.Join(membersSli, ",")
	msg += "\n"
	msg += extraMsg
	return msg
}

func isChatGroup(chat *tgbotapi.Chat) bool {
	return chat != nil && (chat.IsSuperGroup() || chat.IsGroup())
}

// 检查 chatID 是否在白名单内，一般先检查是否为群，如果未群的话再检查是否为白名单的群
// 只有在白名单内的 chat，机器人才会去处理 command。
func isChatToManage(chat *tgbotapi.Chat) bool {
	if !isChatGroup(chat) {
		return false
	}

	for _, id := range conf.ChatsToManage {
		if chat.ID == id {
			return true
		}
	}

	return false
}

func isUserInWhiteList(uid int) bool {
	for _, id := range conf.WhiteList {
		if id == uid {
			return true
		}
	}
	return false
}

func isRollerAdmin(uid int) bool {
	for _, id := range conf.RollerConfig.Admins {
		if id == uid {
			return true
		}
	}
	return false
}

// 是否需要机器人在对话中检查消息类型，用来过滤广告
func messageMustBeCheck(message *tgbotapi.Message) bool {

	if !isChatToManage(message.Chat) {
		return false
	}

	if message.From != nil {
		if isUserInWhiteList(message.From.ID) {
			return false
		}
	}

	return true
}

// 统计 msg 中的entities有多少个 url
func countMessageURLs(entities []tgbotapi.MessageEntity) int {
	var n int
	for _, entity := range entities {
		if entity.Type == "url" {
			n++
		}
	}
	return n
}

// 统计text中出现了多少次白名单中的url
func countURLTotal(text string) int {
	var n int
	for _, u := range conf.URLWhiteList {
		n += strings.Count(text, u)
	}
	return n
}

// 禁言用户
func restrictUser(api *tgbotapi.BotAPI, chatID int64, userID int, lastHours int64) {
	var chatConfig tgbotapi.ChatMemberConfig
	chatConfig.ChatID = chatID
	chatConfig.UserID = userID

	var restrictConfig tgbotapi.RestrictChatMemberConfig
	restrictConfig.ChatMemberConfig = chatConfig
	canWeb := false
	canMedia := false
	canMessage := false
	canOthers := false
	restrictConfig.CanAddWebPagePreviews = &canWeb
	restrictConfig.CanSendMediaMessages = &canMedia
	restrictConfig.CanSendMessages = &canMessage
	restrictConfig.CanSendOtherMessages = &canOthers

	// 暂时禁止一天
	restrictConfig.UntilDate = time.Now().Add(time.Duration(lastHours) * time.Hour).Unix()

	api.RestrictChatMember(restrictConfig)
}

func handleBlackList(api *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if strings.Contains(msg.Text, "乐币") ||
		strings.Contains(strings.ToLower(msg.Text), "dft") ||
		strings.Contains(msg.Text, "币场") {
		deleteUserMessage(api, msg, "广告")
		restrictUser(api, msg.Chat.ID, msg.From.ID, 24)
	}
}

func handleURLMessage(api *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if msg.Entities != nil {
		n := countMessageURLs(*msg.Entities)
		m := countURLTotal(msg.Text)

		if n != m {
			deleteUserMessage(api, msg, "url")
			//restrictUser(api, msg.Chat.ID, msg.From.ID, 24)
		}
	}
}

func handlePhotoMessage(api *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// 删除图片消息
	if msg.Photo != nil {
		deleteUserMessage(api, msg, "photo")
	}
}

func handleDocumentMessage(api *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// 删除文件消息
	if msg.Document != nil {
		deleteUserMessage(api, msg, "document")
	}
}

func isNewMembersMessage(msg *tgbotapi.Message) bool {
	return msg.NewChatMembers != nil && len(*msg.NewChatMembers) > 0
}

func handleNewMemberMessage(api *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if isNewMembersMessage(msg) {
		// 配置中读取欢迎消息，发送之后5分钟内删除
		sendTextMessage(
			api,
			msg.Chat.ID,
			welcomeMsg(*msg.NewChatMembers, conf.WelcomeExtraMsg),
			time.Minute*5,
			false,
			msg.MessageID,
		)
	}
}

func isPrivateChat(msg *tgbotapi.Message) bool {
	return int64(msg.From.ID) == msg.Chat.ID
}

func checkMessage(api *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// handlePhotoMessage(api, msg)
	handleDocumentMessage(api, msg)
	handleURLMessage(api, msg)
	//handleBlackList(api, msg)
	// handleNewMemberMessage(api, msg)
}

func handleCommand(api *tgbotapi.BotAPI, msg *tgbotapi.Message, helpText string) {
	if isPrivateChat(msg) {
		//if msg.Command() == "start" {
		//	isPrivateChatRequest(fmt.Sprintf("%d", msg.From.ID))
		//}
		if msg.Command() == "receive" && len(msg.CommandArguments()) > 0 {
			doBindingYW(api, msg, msg.CommandArguments())
			return
		}
	}

	handleInstantCommand(api, msg, helpText)
}

func isSelfJoinMessage(msg *tgbotapi.Message) bool {
	return msg.NewChatMembers != nil && len(*msg.NewChatMembers) == 1 && (*msg.NewChatMembers)[0].ID == msg.From.ID
}

func isInviteNewMembersMessage(msg *tgbotapi.Message) bool {
	return isNewMembersMessage(msg) && !isSelfJoinMessage(msg)
}

// 记录邀请关系
func recordInviteship(msg *tgbotapi.Message) {
	if isInviteNewMembersMessage(msg) {
		inviter := msg.From.ID
		var invitees []int
		for _, m := range *msg.NewChatMembers {
			invitees = append(invitees, m.ID)
		}
		recordInviteshipRequest(inviter, invitees, msg.Chat.ID, msg.Chat.Title)
	}
}

func handleMessage(api *tgbotapi.BotAPI, msg *tgbotapi.Message, helpText string) {
	if msg == nil {
		return
	}

	// 配置了chats_to_manage的群, 就会被管理..
	if messageMustBeCheck(msg) {
		checkMessage(api, msg)
	}

	//if isChatToManage(msg.Chat) {
	//	recordInviteship(msg)
	//}

	handleCommand(api, msg, helpText)
}

func sendMsgBot(api *tgbotapi.BotAPI, helpText string) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := api.GetUpdatesChan(u)
	if err != nil {
		panic(err)
	}

	for update := range updates {
		DebugLog("sendMsgBot")
		if update.Message != nil {
			DebugLog("[%s] %s", update.Message.From.UserName, update.Message.Text)
			DebugLog("command: %s Args: %s", update.Message.Command(), update.Message.CommandArguments())
		}

		// answerCallback(api, update.CallbackQuery)
		handleMessage(api, update.Message, helpText)
	}
}

func tipsBot(api *tgbotapi.BotAPI, chatID int64) {
	l := len(conf.Tips[fmt.Sprintf("%d", chatID)])
	for {
		now := time.Now()
		past := todayPast(now)
		quo := int64(past.Seconds()/float64(conf.TipsInterval)) + 1
		nextTipsTime := todayStart(now).Add(time.Duration(quo*conf.TipsInterval) * time.Second)
		sleepTime := nextTipsTime.Sub(now)
		DebugLog("sleep %v", sleepTime.Seconds())
		time.Sleep(sleepTime)
		i := int(quo) % l
		sendTextMessage(api, chatID, conf.Tips[fmt.Sprintf("%d", chatID)][i], 0, false)
	}
}

var messageAlignerChan = make(chan *tgbotapi.Message, 1)
var messageAlignerStopChan = make(chan struct{}, 1)

func run() {
	api, err := tgbotapi.NewBotAPI(conf.Token)

	if err != nil {
		panic(err)
	}
	// go botServe(api)
	api.Debug = *debugOn
	DebugLog("Authorized on account %s", api.Self.UserName)
	helpText := botCommandHelp()
	go sendMsgBot(api, helpText)

	// 以下功能不需要
	//go messageAligner(messageAlignerChan, api, messageAlignerStopChan, helpText)

	//for k := range conf.Tips {
	//	id, _ := strconv.ParseInt(k, 0, 64)
	//	go tipsBot(api, id)
	//}

}

func init() {
	log.SetErrorColor(but.COLOR_RED, false)
	log.SetInfoColor(but.COLOR_CYAN, false)
	log.SetWarnColor(but.COLOR_YELLOW, false)
}

func initRedis() {
	var opts redis.Options
	opts.Addr = conf.Redis.Addr
	opts.DB = conf.Redis.DB
	opts.Password = conf.Redis.Password
	redisClient = redis.NewClient(&opts)
	log.Infofln("redis ping: %s", redisClient.Ping().Val())
}

func Run(configPath string) {
	_, err := toml.DecodeFile(configPath, &conf)
	if err != nil {
		panic(err)
	}

	//go dragonexPrice()
	//go conhotDividend()
	initRedis()
	//go RollerMonitor()
	//initRegisterdMembers()
	run()

	<-exitChan
}

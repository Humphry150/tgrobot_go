package bot

import (
	"fmt"
	"strings"
	"time"

	gomap "bitbucket.org/magmeng/go-utils/go-map"
	"bitbucket.org/magmeng/go-utils/log"
	"github.com/BurntSushi/toml"
	"github.com/achillesss/gorequest"
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

type botClient struct {
	api              *tgbotapi.BotAPI
	config           botConfig
	messagesToDelete *gomap.GoMap
}

type botConfig struct {
	Token              string     `toml:"token"`
	KycChatID          int64      `toml:"kyc_chat_id"`
	ExternalAPIAddress string     `toml:"external_api_address"`
	KYCResults         [][]string `toml:"kyc_results"`
}

func NewBot(path string) *botClient {
	var b botClient
	_, err := toml.DecodeFile(path, &b.config)
	if err != nil {
		panic(err)
	}
	b.messagesToDelete = gomap.NewMap(make(deleteQueue))
	go b.messagesToDelete.Handler()
	return &b
}

func (b *botClient) getUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updatesChan, err := b.api.GetUpdatesChan(u)
	if err != nil {
		panic(err)
	}

	for update := range updatesChan {
		b.checkCallbackQuery(update)
	}
}

var exitChan chan struct{}

func (b *botClient) Serve() {
	api, err := tgbotapi.NewBotAPI(b.config.Token)
	if err != nil {
		panic(err)
	}
	b.api = api
	b.api.Debug = true
	log.Infofln("Authorized on account %s", b.api.Self.UserName)
	b.botServe()
	go b.getUpdates()
	<-exitChan
}

var kycCallbackDataMap = map[string]string{
	"0": "通过",
	"2": "身份证信息不匹配",
	"3": "身份证照片不合规",
	"4": "手持照不合规",
}

var kycCallbackReasonMap = map[string]string{
	"0": "",
	"1": "",
	"2": "身份证信息不匹配",
	"3": "身份证照片不清晰或不合规",
	"4": "手持照不清晰或不合规",
}

func (b botClient) checkCallbackQuery(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		log.Infofln("update: %+#v", update)
		log.Infofln("callback query: %+#v", update.CallbackQuery.ChatInstance)
		log.Infofln("callback query: %+#v", update.CallbackQuery.Data)
		log.Infofln("callback query: %+#v", *update.CallbackQuery.From)
		log.Infofln("callback query: %+#v", update.CallbackQuery.GameShortName)
		log.Infofln("callback query: %+#v", update.CallbackQuery.ID)
		log.Infofln("callback query: %+#v", update.CallbackQuery.InlineMessageID)
		log.Infofln("callback query: %+#v", *update.CallbackQuery.Message)
		log.Infofln("callback query: %+#v", *update.CallbackQuery.Message.Photo)

		results := strings.Split(update.CallbackQuery.Data, "_")

		switch results[0] {
		case "1":
			msg := tgbotapi.NewEditMessageReplyMarkup(b.config.KycChatID, update.CallbackQuery.Message.MessageID, b.initKycButtons(results[1]))
			b.api.Send(msg)
			b.cancelDelete(update.CallbackQuery.Message.MessageID)
		default:
			msg := tgbotapi.NewEditMessageReplyMarkup(b.config.KycChatID, update.CallbackQuery.Message.MessageID, tgbotapi.NewInlineKeyboardMarkup(
				[]tgbotapi.InlineKeyboardButton{
					tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("« 撤销 •%s", kycCallbackDataMap[results[0]]), joinKYCIDInstruction("1", results[1])),
				},
			))
			b.api.Send(msg)

			b.addDelete(update.CallbackQuery.Message.MessageID, results[1], results[0], validUserName(toString(update.CallbackQuery.From.ID), update.CallbackQuery.From.UserName, update.CallbackQuery.From.FirstName, update.CallbackQuery.From.LastName))
		}
	}
}

type messageToDelete struct {
	deleteTicker *time.Ticker
	messageID    int
	cancelChan   chan struct{}

	callbackFunc interface{}
	callbackArgs []interface{}
}

type deleteQueue map[int]*messageToDelete

func (b *botClient) delete(m *messageToDelete) {
	log.Infofln("add delete %d", m.messageID)
	select {
	case <-m.cancelChan:
		log.Infofln("cancel delete %d", m.messageID)
		return
	case <-m.deleteTicker.C:
		log.Infofln("delete %d", m.messageID)
		resp := NewCaller(m.callbackFunc, m.callbackArgs...).Call(false)
		if len(resp) == 0 || resp[0].Interface() == nil {
			deleteMsg(b.api, b.config.KycChatID, m.messageID)
			b.messagesToDelete.Delete(m.messageID)
		} else {
			sendTextMessage(b.api, b.config.KycChatID, fmt.Sprintf("返回KYC结果失败：`%v`", resp[0].Interface()), 0)
		}
	}
}

func (b *botClient) cancelDelete(messageID int) {
	var m *messageToDelete
	b.messagesToDelete.Query(messageID, &m)
	if m == nil {
		return
	}
	close(m.cancelChan)
	b.messagesToDelete.Delete(messageID)
}

func (b *botClient) addDelete(messageID int, kycID, result, operator string) {
	var msg *messageToDelete
	b.messagesToDelete.Query(messageID, &msg)
	if msg != nil {
		return
	}

	var m messageToDelete
	m.deleteTicker = time.NewTicker(time.Second * 120)
	m.messageID = messageID
	m.cancelChan = make(chan struct{})
	m.callbackFunc = confirmKYCResult
	m.callbackArgs = []interface{}{
		kycID,
		kycCallbackReasonMap[result],
		operator,
	}

	b.messagesToDelete.Add(messageID, &m)
	go b.delete(&m)
}

func makeRequest(method, url string, p map[string]string) *gorequest.SuperAgent {
	req := gorequest.New().CustomMethod(method, url)
	for k, v := range p {
		req.Param(k, v)
	}

	return req
}

func confirmKYCResult(kycID, reason, operator string) error {
	u := "https://coinhot.io/inner/accounts/telegram/verify/identification/"
	p := map[string]string{
		"identification_id": kycID,
		"operator":          operator,
		"verified":          "0",
		"reason":            reason,
	}

	if reason == "" {
		p["verified"] = "1"
	}

	req := makeRequest("GET", u, p)
	_, body, err := req.End()
	log.Infofln("body: %s", body)
	if err != nil {
		return err[0]
	}

	if body != "{}" {
		return fmt.Errorf("%s", body)
	}

	return nil
}

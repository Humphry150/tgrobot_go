package bot

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/magmeng/go-utils/log"
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

// 使用机器人

// 删除消息
func deleteMsg(api *tgbotapi.BotAPI, chatID int64, messageID int) {
	api.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: chatID, MessageID: messageID})
}

// 文字消息
func newTextMessage(chatID int64, text string, replyMsgID ...int) tgbotapi.Chattable {
	msg := tgbotapi.NewMessage(chatID, text)
	if len(replyMsgID) != 0 {
		msg.ReplyToMessageID = replyMsgID[0]
	}
	msg.ParseMode = "MarkDown"
	return msg
}

// 图片消息
// 本地图片地址
// 图片 bytes
func newPhotoMessageFromBytes(chatID int64, title string, data []byte, replyMsgID ...int) tgbotapi.Chattable {
	var photo tgbotapi.FileBytes
	photo.Name = title
	photo.Bytes = data
	msg := tgbotapi.NewPhotoUpload(chatID, photo)
	if len(replyMsgID) != 0 {
		msg.ReplyToMessageID = replyMsgID[0]
	}
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("test_text", "test_data"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("test_text", "test_data"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("test_text", "test_data"),
		),
	)
	return msg
}

func newButtonRow(texts ...[]string) []tgbotapi.InlineKeyboardButton {
	var row []tgbotapi.InlineKeyboardButton
	for _, text := range texts {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(text[0], text[1]))
	}
	return row
}

func newButtons(texts ...[][]string) tgbotapi.InlineKeyboardMarkup {
	var buttons [][]tgbotapi.InlineKeyboardButton
	for _, t := range texts {
		row := newButtonRow(t...)
		buttons = append(buttons, row)
	}
	return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

func joinKYCIDInstruction(instruction, kycID string) string {
	return strings.Join([]string{instruction, kycID}, "_")
}

func (b botClient) initKycButtons(kycID string) tgbotapi.InlineKeyboardMarkup {
	var texts [][][]string
	for _, r := range b.config.KYCResults {
		texts = append(texts, [][]string{[]string{r[0], joinKYCIDInstruction(r[1], kycID)}})
	}
	return newButtons(texts...)
}

func (b botClient) newKYCMessage(title string, data []byte, kycID string) tgbotapi.Chattable {
	var photo tgbotapi.FileBytes
	photo.Name = title
	photo.Bytes = data
	msg := tgbotapi.NewPhotoUpload(b.config.KycChatID, photo)

	msg.ReplyMarkup = b.initKycButtons(kycID)
	return msg
}

func newPhotoMessageFromReader(chatID int64, title string, data io.Reader, size int64, replyMsgID ...int) tgbotapi.Chattable {
	var photo tgbotapi.FileReader
	photo.Name = title
	photo.Reader = data
	photo.Size = size
	msg := tgbotapi.NewPhotoUpload(chatID, photo)
	if len(replyMsgID) != 0 {
		msg.ReplyToMessageID = replyMsgID[0]
	}
	return msg
}

func sendMessage(api *tgbotapi.BotAPI, msg tgbotapi.Chattable, deleteAfter time.Duration) error {
	m, err := api.Send(msg)
	if err != nil {
		return err
	}

	if deleteAfter != 0 {
		go func() {
			time.Sleep(deleteAfter)
			deleteMsg(api, m.Chat.ID, m.MessageID)
		}()
	}
	return nil
}

func sendTextMessage(api *tgbotapi.BotAPI, chatID int64, text string, deleteAfter time.Duration, replyMsgID ...int) error {
	if text == "" {
		return fmt.Errorf("nil text")
	}

	msg := newTextMessage(chatID, text, replyMsgID...)
	return sendMessage(api, msg, deleteAfter)
}

func sendPhotoMessageFromBytes(api *tgbotapi.BotAPI, chatID int64, title string, data []byte, deleteAfter time.Duration, replyMsgID ...int) error {
	if data == nil {
		return fmt.Errorf("nil data")
	}

	msg := newPhotoMessageFromBytes(chatID, title, data, replyMsgID...)
	return sendMessage(api, msg, deleteAfter)
}

func (b botClient) sendKYCMessageFromBytes(title string, data []byte, kycID string) error {
	if data == nil {
		return fmt.Errorf("nil data")
	}

	msg := b.newKYCMessage(title, data, kycID)
	return sendMessage(b.api, msg, 0)
}

func (b botClient) sendTextMessageHandler(w http.ResponseWriter, r *http.Request) {
	text := r.FormValue("text")
	chatID := r.FormValue("chat_id")
	chatIDNum, _ := strconv.ParseInt(chatID, 0, 64)
	err := sendTextMessage(b.api, chatIDNum, text, 0)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte("{}"))
}

func (b botClient) sendPhotoMessageFromBytesHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(r.ContentLength)
	title := r.FormValue("title")
	data, _ := ioutil.ReadAll(r.Body)
	chatID := r.FormValue("chat_id")
	chatIDNum, _ := strconv.ParseInt(chatID, 0, 64)
	err := sendPhotoMessageFromBytes(b.api, chatIDNum, title, data, 0)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte("{}"))
}

func (b botClient) sendKYCMessageFromFileHandler(w http.ResponseWriter, r *http.Request) {
	KYCID := r.FormValue("kyc_id")
	if KYCID == "" {
		w.Write([]byte("invalid kyc_id"))
		return
	}

	r.ParseMultipartForm(r.ContentLength)
	file, header, err := r.FormFile("photo")
	if err != nil {
		log.Errorf("form file error: %v", err)
		w.Write([]byte(err.Error()))
		return
	}

	data, _ := ioutil.ReadAll(file)
	go func() {
		n := 3
		for n > 0 {
			err = b.sendKYCMessageFromBytes(header.Filename, data, KYCID)
			if err == nil {
				log.Infofln("send kyc photo success")
				break
			}
			if err != nil {
				log.Errorfln("send photo error: %v", err)
				n--
			}
		}
	}()

	w.Write([]byte("{}"))
}

func (b botClient) sendPhotoMessageFromFileHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(r.ContentLength)
	file, header, err := r.FormFile("photo")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	data, _ := ioutil.ReadAll(file)
	err = sendPhotoMessageFromBytes(b.api, b.config.KycChatID, header.Filename, data, 0)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	w.Write([]byte("{}"))
}

func (b botClient) botServe() {
	l, err := net.Listen("tcp", b.config.ExternalAPIAddress)
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	// mux.HandleFunc("/message/kyc/file/send", b.sendPhotoMessageFromFileHandler)
	mux.HandleFunc("/message/kyc/file/send", b.sendKYCMessageFromFileHandler)
	go http.Serve(l, mux)
	log.Infofln("serve send message on %s", b.config.ExternalAPIAddress)
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

func toString(v interface{}) string {
	switch a := v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", a)
	case float32, float64:
		return fmt.Sprintf("%f", a)
	default:
		return fmt.Sprintf("%#v", a)
	}
}

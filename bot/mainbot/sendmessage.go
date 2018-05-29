package bot

import (
	"fmt"
	"io"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

// 使用机器人

// 删除消息
func deleteMsg(api *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	if msg.Chat != nil {
		api.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: msg.Chat.ID, MessageID: msg.MessageID})
	}
}

// 文字消息
func newTextMessage(chatID int64, text string, replyMarkup interface{}, replyMsgID ...int) tgbotapi.Chattable {
	msg := tgbotapi.NewMessage(chatID, text)
	if len(replyMsgID) != 0 {
		msg.ReplyToMessageID = replyMsgID[0]
	}
	msg.ParseMode = "MarkDown"
	msg.ReplyMarkup = replyMarkup
	return msg
}

// 图片消息
// 本地图片地址
func newPhotoMessageFromPath(chatID int64, path string, replyMsgID ...int) tgbotapi.Chattable {
	msg := tgbotapi.NewPhotoUpload(chatID, path)
	if len(replyMsgID) != 0 {
		msg.ReplyToMessageID = replyMsgID[0]
	}
	return msg
}

// 图片 bytes
func newPhotoMessageFromBytes(chatID int64, title string, data []byte, markup interface{}, replyMsgID ...int) tgbotapi.Chattable {
	var photo tgbotapi.FileBytes
	photo.Name = title
	photo.Bytes = data
	msg := tgbotapi.NewPhotoUpload(chatID, photo)
	if len(replyMsgID) != 0 {
		msg.ReplyToMessageID = replyMsgID[0]
	}
	msg.ReplyMarkup = markup
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

func sendMessage(api *tgbotapi.BotAPI, msg tgbotapi.Chattable, deleteAfter time.Duration, mustPin bool) error {
	m, err := api.Send(msg)
	if err != nil {
		return err
	}

	if mustPin {
		pingMessage(api, m.Chat.ID, m.MessageID)
	}

	if deleteAfter != 0 {
		go func() {
			time.Sleep(deleteAfter)
			deleteMsg(api, &m)
		}()
	}
	return nil
}

func sendTextMessage(api *tgbotapi.BotAPI, chatID int64, text string, deleteAfter time.Duration, mustPin bool, replyMsgID ...int) error {
	if text == "" {
		return fmt.Errorf("nil text")
	}

	msg := newTextMessage(chatID, text, nil, replyMsgID...)
	return sendMessage(api, msg, deleteAfter, mustPin)
}

func getRewardKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("注册奖励", "register"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("交易奖励", "trade"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("加入龙网群", "join_dragonex"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("实名认证", "kyc"),
		},
		[]tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData("活动奖励", "activity"),
		},
	)
}
func rollerKeyboardRewardCount() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		[]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("/roll reward 1"),
		},
		[]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("/roll reward 3"),
		},
		[]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("/roll reward 5"),
		},
		[]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("/roll reward 10"),
		},
	)
}
func rollerKeyboardRankLength() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		[]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("/roll rank 20"),
		},
		[]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("/roll rank auto"),
		},
	)
}

func rollerKeyboardRollingRange() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		[]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("/roll range 10000"),
		},
	)
}

func rollerKeyboardRollerProccessing() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		[]tgbotapi.KeyboardButton{
			tgbotapi.NewKeyboardButton("/roll"),
		},
	)
}

func sendTextMessageHideKeyBoard(api *tgbotapi.BotAPI, chatID int64, text string, deleteAfter time.Duration, mustPin bool, replyMsgID ...int) error {
	if text == "" {
		return fmt.Errorf("nil text")
	}

	keyboard := tgbotapi.NewHideKeyboard(false)
	msg := newTextMessage(chatID, text, keyboard, replyMsgID...)
	return sendMessage(api, msg, deleteAfter, mustPin)
}
func sendTextMessageStartRoll(api *tgbotapi.BotAPI, chatID int64, text string, deleteAfter time.Duration, mustPin bool, replyMsgID ...int) error {
	if text == "" {
		return fmt.Errorf("nil text")
	}

	keyboard := rollerKeyboardRollerProccessing()
	msg := newTextMessage(chatID, text, keyboard, replyMsgID...)
	return sendMessage(api, msg, deleteAfter, mustPin)
}

func sendTextMessageSetRewardCount(api *tgbotapi.BotAPI, chatID int64, text string, deleteAfter time.Duration, mustPin bool, replyMsgID ...int) error {
	if text == "" {
		return fmt.Errorf("nil text")
	}

	keyboard := rollerKeyboardRewardCount()
	msg := newTextMessage(chatID, text, keyboard, replyMsgID...)
	return sendMessage(api, msg, deleteAfter, mustPin)
}

func sendTextMessageSetRankLength(api *tgbotapi.BotAPI, chatID int64, text string, deleteAfter time.Duration, mustPin bool, replyMsgID ...int) error {
	if text == "" {
		return fmt.Errorf("nil text")
	}

	keyboard := rollerKeyboardRankLength()
	msg := newTextMessage(chatID, text, keyboard, replyMsgID...)
	return sendMessage(api, msg, deleteAfter, mustPin)
}

func sendTextMessageInitRoller(api *tgbotapi.BotAPI, chatID int64, text string, deleteAfter time.Duration, mustPin bool, replyMsgID ...int) error {
	if text == "" {
		return fmt.Errorf("nil text")
	}

	keyboard := rollerKeyboardRollingRange()
	msg := newTextMessage(chatID, text, keyboard, replyMsgID...)
	return sendMessage(api, msg, deleteAfter, mustPin)
}

func sendPhotoMessageFromPath(api *tgbotapi.BotAPI, chatID int64, path string, deleteAfter time.Duration, mustPin bool, replyMsgID ...int) error {
	msg := newPhotoMessageFromPath(chatID, path, replyMsgID...)
	return sendMessage(api, msg, deleteAfter, mustPin)
}

func sendPhotoMessageFromBytes(api *tgbotapi.BotAPI, chatID int64, title string, data []byte, markup interface{}, deleteAfter time.Duration, mustPin bool, replyMsgID ...int) error {
	if data == nil {
		return fmt.Errorf("nil data")
	}

	msg := newPhotoMessageFromBytes(chatID, title, data, markup, replyMsgID...)
	return sendMessage(api, msg, deleteAfter, mustPin)
}

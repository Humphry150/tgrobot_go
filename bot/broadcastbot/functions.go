package bot

import (
	"fmt"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

// 文字消息
func newTextMessage(chatID int64, text string, replyMsgID ...int) tgbotapi.Chattable {
	msg := tgbotapi.NewMessage(chatID, text)
	if len(replyMsgID) != 0 {
		msg.ReplyToMessageID = replyMsgID[0]
	}
	msg.ParseMode = "MarkDown"
	return msg
}

func sendTextMessage(api *tgbotapi.BotAPI, chatID int64, text string, deleteAfter time.Duration, replyMsgID ...int) error {
	if text == "" {
		return fmt.Errorf("nil text")
	}

	msg := newTextMessage(chatID, text, replyMsgID...)
	return sendMessage(api, msg, deleteAfter)
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

// 删除消息
func deleteMsg(api *tgbotapi.BotAPI, chatID int64, messageID int) {
	api.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: chatID, MessageID: messageID})
}

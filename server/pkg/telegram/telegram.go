// telegram 通知模块，发送消息到 telegram bot
package telegram

import (
	"errors"
	"log"
	"strconv"

	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Telegram struct {
	token  string
	chatId int64
}

func New(token string, chaId int64) *Telegram {
	return &Telegram{
		token:  token,
		chatId: chaId,
	}
}

// 从环境变量里面读取配置，发送通知到 telegram bot
func SendText(txtMsg string) error {

	token := os.Getenv("TELEGRAM_APITOKEN")
	chatIdStr := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chatIdStr == "" { // skip
		log.Printf("telegram notify token or chatId is empty")
		return errors.New("token or chatId is empty")
	}

	i, err := strconv.Atoi(chatIdStr)
	if err != nil {
		log.Printf("telegram notify parse chatId err: %v", err)
		return err
	}
	chaId := int64(i)

	tg := New(token, chaId)
	err = tg.SendText(txtMsg)
	if err != nil {
		log.Printf("telegram notify err: %v", err)
		return err
	}

	return nil
}

// 通知 telegram bot 启动消息的函数
func (t *Telegram) SendText(txtMsg string) error {

	bot, err := tgbotapi.NewBotAPI(t.token)
	if err != nil {
		panic(err)
	}

	// bot.Debug = true
	msg := tgbotapi.NewMessage(t.chatId, txtMsg)
	if _, err := bot.Send(msg); err != nil {
		return err
	}

	return nil
}

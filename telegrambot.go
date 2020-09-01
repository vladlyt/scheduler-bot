package main

import (
	"fmt"
	"github.com/Syfaro/telegram-bot-api"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const NEW_COMMAND_USAGE = `
Send this command with reply of message.
Expected usage of this command: /store {day [1-7]} {class [1-7]}.
For example: /store 1 2 (Set reply message on Monday at 2 class).
`

func getClassTimeReminder(class int) (int, int) {
	switch class {
	case 1:
		return 8, 15
	case 2:
		return 10, 10
	case 3:
		return 12, 5
	case 4:
		return 14, 0
	case 5:
		return 15, 55
	case 6:
		return 18, 15
	case 7:
		return 20, 5
	default:
		return 0, 0
	}
}

type Reminder struct {
	Weekday   int
	Class     int
	ChatId    int64
	MessageID int
}

func SendMessage(bot *tgbotapi.BotAPI, reminder Reminder) {
	t := time.Now()
	nowWeekday := int(t.Weekday())
	if nowWeekday == 0 {
		nowWeekday = 7
	}

	nowH, nowM, _ := t.Clock()
	classH, classM := getClassTimeReminder(reminder.Class)

	weekday := reminder.Weekday - nowWeekday
	if reminder.Weekday < nowWeekday || (
		nowWeekday == reminder.Weekday && (
			classH < nowH || (
				classH == nowH && classM < nowM))) {
		weekday += 7
	}

	newT := t.AddDate(0, 0, weekday)
	t = time.Date(newT.Year(), newT.Month(), newT.Day(), classH, classM, 0, 0, newT.Location())

	bot.Send(
		tgbotapi.NewMessage(
			reminder.ChatId,
			fmt.Sprintf(
				"Ok, will notify you in %.1f hours, master",
				time.Until(t).Hours(),
			),
		),
	)

	<-time.After(time.Until(t))
	forward := tgbotapi.NewForward(reminder.ChatId, reminder.ChatId, reminder.MessageID)
	bot.Send(forward)
}

func telegramBot(bot *tgbotapi.BotAPI) {
	webhookEndpoint := fmt.Sprintf("%s:%s/%s", os.Getenv("DOMAIN"), "443", bot.Token)
	fmt.Println("webhookEndpoint", webhookEndpoint)
	_, err := bot.SetWebhook(tgbotapi.NewWebhook(webhookEndpoint))
	if err != nil {
		log.Fatal(err)
	}
	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Fatal(err)
	}
	if info.LastErrorDate != 0 {
		log.Printf("Telegram callback failed: %s", info.LastErrorMessage)
	}
	updates := bot.ListenForWebhook("/" + bot.Token)
	go http.ListenAndServe("0.0.0.0:"+os.Getenv("PORT"), nil)


	fmt.Println("Starting my master...")
	for update := range updates {
		fmt.Println("Message", update.Message.Text)
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "store":
				args := strings.Split(update.Message.CommandArguments(), " ")
				if len(args) != 2 {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, NEW_COMMAND_USAGE))
					continue
				}
				weekday, err := strconv.Atoi(args[0])
				if err != nil || weekday < 0 || weekday > 7 {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, NEW_COMMAND_USAGE))
					continue
				}
				class, err := strconv.Atoi(args[1])
				if err != nil || class < 0 || class > 7 {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, NEW_COMMAND_USAGE))
					continue
				}
				if update.Message.ReplyToMessage == nil {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, NEW_COMMAND_USAGE))
					continue
				}

				go SendMessage(bot, Reminder{
					Weekday:   weekday,
					Class:     class,
					ChatId:    update.Message.Chat.ID,
					MessageID: update.Message.MessageID,
				})
			default:
				continue
			}
		}
	}
}

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		panic(err)
	}

	telegramBot(bot)
}

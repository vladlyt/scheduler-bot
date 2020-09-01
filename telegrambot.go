package main

import (
	"errors"
	"fmt"
	"github.com/Syfaro/telegram-bot-api"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	HTTPS_PORT = "443"
)

type Reminder struct {
	Weekday   int
	Class     int
	ChatId    int64
	MessageID int
}

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

func sendReminder(bot *tgbotapi.BotAPI, reminder *Reminder) {
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
		// next week
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

	// sleep for some time
	<-time.After(time.Until(t))

	bot.Send(
		tgbotapi.NewForward(reminder.ChatId, reminder.ChatId, reminder.MessageID),
	)
}

func getReminderFromCommand(msg *tgbotapi.Message) (*Reminder, error) {
	args := strings.Split(msg.CommandArguments(), " ")
	if len(args) != 2 {
		return nil, errors.New("Two arguments required (weekday[1-7] and class[1-7])")
	}
	weekday, err := strconv.Atoi(args[0])
	if err != nil || weekday < 0 || weekday > 7 {
		return nil, errors.New("Weekday must be an integer 1-7")
	}
	class, err := strconv.Atoi(args[1])
	if err != nil || class < 0 || class > 7 {
		return nil, errors.New("Class must be an integer 1-7")
	}
	if msg.ReplyToMessage == nil {
		return nil, errors.New("Use reply on message")
	}
	return &Reminder{
		Weekday:   weekday,
		Class:     class,
		ChatId:    msg.Chat.ID,
		MessageID: msg.MessageID,
	}, nil
}

func runTelegramBot(bot *tgbotapi.BotAPI) {
	_, err := bot.SetWebhook(
		tgbotapi.NewWebhook(fmt.Sprintf("%s:%s/%s", os.Getenv("DOMAIN"), HTTPS_PORT, bot.Token)),
	)
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

	for update := range updates {
		if update.Message == nil {
			continue
		}
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "store":
				reminder, err := getReminderFromCommand(update.Message)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
					continue
				}
				go sendReminder(bot, reminder)
			case "help":
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, `
Send /store command with reply of message.
Expected usage of /store command: /store {day [1-7]} {class [1-7]}.
For example: /store 1 2 (Set reply message on Monday at 2 class).
`))
			}
		}
	}
}

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		panic(err)
	}

	runTelegramBot(bot)
}

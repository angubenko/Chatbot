package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"sync"
)

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	var mux sync.Mutex
	quizInProgress := make(map[int64]*Quiz)

	doneQuizOnChatID := make(chan int64)
	go func() {
		for {
			chanID := <-doneQuizOnChatID
			mux.Lock()
			delete(quizInProgress, chanID)
			mux.Unlock()
		}
	}()

	messagesToSend := make(chan Message)
	go func() {
		for {
			message := <-messagesToSend
			_, err = bot.Send(tgbotapi.NewMessage(message.chatID, message.message))
			if err != nil {
				log.Println(err)
			}
		}
	}()

	updates, err := bot.GetUpdatesChan(tgbotapi.UpdateConfig{Offset: 0, Timeout: 60})
	for update := range updates {
		if update.Message == nil {
			continue
		}
		chatID := update.Message.Chat.ID

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				{
					messagesToSend <- Message{
						chatID:  chatID,
						message: "Starting quiz...",
					}
					mux.Lock()
					if _, ok := quizInProgress[chatID]; ok {
						_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Quiz is already in progress."))
					} else {
						var quiz = Quiz{ChatID: chatID, OutgoingMessages: messagesToSend, DoneQuiz: doneQuizOnChatID}
						quizInProgress[chatID] = &quiz
						go quizInProgress[chatID].serveQuiz()
					}
					mux.Unlock()
				}
			case "help":
				{
					messagesToSend <- Message{
						chatID: chatID,
						message: "This is a quiz Bot. Use: \r " +
							"start to start game \r " +
							"score to check score",
					}
				}
			case "score":
				{
					messagesToSend <- Message{
						chatID:  chatID,
						message: "user score",
					}
				}
			}
			continue
		}

		mux.Lock()
		if quiz, ok := quizInProgress[chatID]; ok {
			quiz.IncomingAnswers <- UserAnswer{name: update.Message.From.UserName, answer: update.Message.Text}
		}
		mux.Unlock()
	}
}

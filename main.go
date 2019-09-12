package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
	"strconv"
	"sync"
)

const (
	cacheFile string = "userScore.cache"
)

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	addUserScore := make(chan string)
	scoreTracker := ScoreTracker{cacheFile: cacheFile, scoreUpdates: addUserScore}
	err = scoreTracker.start()
	if err != nil {
		log.Panic(err)
	}

	var quizInProgressMux sync.Mutex
	quizInProgress := make(map[int64]*Quiz)

	doneQuizOnChatID := make(chan int64)
	go func() {
		for {
			chanID := <-doneQuizOnChatID
			quizInProgressMux.Lock()
			delete(quizInProgress, chanID)
			quizInProgressMux.Unlock()
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
					quizInProgressMux.Lock()
					if _, ok := quizInProgress[chatID]; ok {
						_, _ = bot.Send(tgbotapi.NewMessage(chatID, "Quiz is already in progress."))
					} else {
						var quiz = Quiz{ChatID: chatID, OutgoingMessages: messagesToSend,
							DoneQuiz: doneQuizOnChatID, AddScore: addUserScore}
						quizInProgress[chatID] = &quiz
						go quizInProgress[chatID].serveQuiz()
					}
					quizInProgressMux.Unlock()
				}
			case "stop":
				{
					quizInProgressMux.Lock()
					if quiz, ok := quizInProgress[chatID]; ok {
						quiz.IncomingAnswers <- UserAnswer{name: update.Message.From.UserName, answerType: Stop}
					} else {
						messagesToSend <- Message{
							chatID:  chatID,
							message: "No quiz to stop",
						}
					}
					quizInProgressMux.Unlock()
				}
			case "skip":
				{
					quizInProgressMux.Lock()
					if quiz, ok := quizInProgress[chatID]; ok {
						quiz.IncomingAnswers <- UserAnswer{name: update.Message.From.UserName, answerType: Skip}
					} else {
						messagesToSend <- Message{
							chatID:  chatID,
							message: "No quiz running",
						}
					}
					quizInProgressMux.Unlock()
				}
			case "help":
				{
					messagesToSend <- Message{
						chatID: chatID,
						message: "This is a Quiz Bot. Use: \r\n " +
							"/start to start game \r\n" +
							"/score to check score",
					}
				}
			case "score":
				{
					message := strconv.Itoa(scoreTracker.getScore(update.Message.From.UserName))
					messagesToSend <- Message{
						chatID:  chatID,
						message: message,
					}
				}
			}
			continue
		}

		quizInProgressMux.Lock()
		if quiz, ok := quizInProgress[chatID]; ok {
			quiz.IncomingAnswers <- UserAnswer{name: update.Message.From.UserName, answerType: Reply, answer: update.Message.Text}
		}
		quizInProgressMux.Unlock()
	}
}

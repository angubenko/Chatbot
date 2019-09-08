package main

import (
	"encoding/json"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"
)

const (
	cacheFile string = "userScore.cache"
)

func loadUserScoreFromCache() (map[string]int, error) {
	bytes, err := ioutil.ReadFile(cacheFile)
	userScore := make(map[string]int)
	err = json.Unmarshal(bytes, &userScore)
	return userScore, err
}

func updateUserScore(userScore map[string]int, addUserScore chan string, userScoreMux *sync.Mutex) {
	for {
		userName := <-addUserScore
		userScoreMux.Lock()
		userScore[userName] += 1
		jsonData, _ := json.Marshal(userScore)
		ioutil.WriteFile(cacheFile, jsonData, 0644)
		userScoreMux.Unlock()
	}
}

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

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

	var userScoreMux sync.Mutex
	userScore, err := loadUserScoreFromCache()
	if err != nil {
		log.Println("error: ", err)
		userScore = make(map[string]int)
	}
	addUserScore := make(chan string)
	go updateUserScore(userScore, addUserScore, &userScoreMux)

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
					userScoreMux.Lock()
					var message string
					if score, ok := userScore[update.Message.From.UserName]; ok {
						message = "Your score is " + strconv.Itoa(score)
					} else {
						message = "Sorry, you haven't played this game yet."
					}
					userScoreMux.Unlock()
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
			quiz.IncomingAnswers <- UserAnswer{name: update.Message.From.UserName, answer: update.Message.Text}
		}
		quizInProgressMux.Unlock()
	}
}

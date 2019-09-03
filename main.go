package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type Test struct {
	ResponseCode int        `json:"response_code"`
	Results      []Question `json:"results"`
}

type Question struct {
	Category         string   `json:"category"`
	Type             string   `json:"type"`
	Difficulty       string   `json:"difficulty"`
	Question         string   `json:"question"`
	CorrectAnswer    string   `json:"correct_answer"`
	IncorrectAnswers []string `json:"incorrect_answers"`
}

func getQuestions() ([]Question, error) {
	resp, err := http.Get("https://opentdb.com/api.php?amount=10")
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {

	}
	var data Test
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return nil, err
	}
	return data.Results, nil
}

func runQuiz() {
	questions, err := getQuestions()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(questions)
}

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	updatesChan, err := bot.GetUpdatesChan(tgbotapi.UpdateConfig{Offset: 0, Timeout: 60})
	quizStarted := false
	for update := range updatesChan {
		if update.Message == nil {
			continue
		}
		if update.Message.IsCommand() && !quizStarted {
			chatID := update.Message.Chat.ID
			switch update.Message.Command() {
			case "start":
				{
					bot.Send(tgbotapi.NewMessage(chatID, "Starting quiz..."))
					runQuiz()
				}
			case "top":
				{
					bot.Send(tgbotapi.NewMessage(chatID, "Leaderboard"))
				}
			default:
				{
					bot.Send(tgbotapi.NewMessage(chatID, "Help"))
				}
			}
		}
	}
}

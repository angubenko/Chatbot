package main

import (
	"encoding/json"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"html"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
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

func updateScore(userName string) error {

	return nil
}

func getScore(userName string) (int, error) {

	return -1, nil
}

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	updates, err := bot.GetUpdatesChan(tgbotapi.UpdateConfig{Offset: 0, Timeout: 60})
	var question Question
	done := make(chan bool)
	userScore := make(map[string]int)
	for update := range updates {
		if update.Message == nil {
			continue
		}
		chatID := update.Message.Chat.ID

		// execute commands
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				{
					go func() {
						bot.Send(tgbotapi.NewMessage(chatID, "Starting quiz..."))
						bot.Send(tgbotapi.NewMessage(chatID, "Reply to bot to submit your answer."))
						questions, _ := getQuestions()
						for _, q := range questions {
							question = q
							message := html.UnescapeString(question.Question) + "\r\n"
							answers := question.IncorrectAnswers
							answers = append(answers, html.UnescapeString(question.CorrectAnswer))
							rand.Seed(time.Now().UnixNano())
							rand.Shuffle(len(answers), func(i, j int) {
								answers[i], answers[j] = answers[j], answers[i]
							})
							for _, a := range answers {
								message += "-  " + a + "\r\n"
							}
							bot.Send(tgbotapi.NewMessage(chatID, message))
							<-done
							question = Question{}
						}
					}()
				}
			case "help":
				{
					bot.Send(tgbotapi.NewMessage(chatID, "This is a quiz bot. Use: \r "+
						"start to start game \r "+
						"score to check score"))
				}
			case "score":
				{
					if score, ok := userScore[update.Message.From.UserName]; ok {
						bot.Send(tgbotapi.NewMessage(chatID, strconv.Itoa(score)))
					} else {
						bot.Send(tgbotapi.NewMessage(chatID, "You haven't played this game"))
					}
				}
			}
		}

		if question.Question != "" && !update.Message.IsCommand() {
			if strings.ToLower(update.Message.Text) == strings.ToLower(html.UnescapeString(strings.TrimSpace(question.CorrectAnswer))) {
				bot.Send(tgbotapi.NewMessage(chatID, "Correct!"))
				userScore[update.Message.From.UserName] += 1
				done <- true
			} else if strings.ToLower(update.Message.Text) == "skip" {
				done <- true
			} else {
				bot.Send(tgbotapi.NewMessage(chatID, "Incorrect, try again"))
			}
		}
	}
}

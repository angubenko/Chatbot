package main

import (
	"encoding/json"
	"html"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type questionRequestResponse struct {
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

type UserAnswer struct {
	name   string
	answer string
}

type Message struct {
	chatID  int64
	message string
}

type Quiz struct {
	ChatID           int64
	Questions        []Question
	OutgoingMessages chan Message
	IncomingAnswers  chan UserAnswer
	DoneQuiz         chan int64
}

func (q *Quiz) serveQuiz() {
	q.Questions, _ = q.requestQuestions()
	q.IncomingAnswers = make(chan UserAnswer)
	defer close(q.IncomingAnswers)

	for _, question := range q.Questions {
		func() {
			timer := time.NewTimer(30 * time.Second)
			defer timer.Stop()

			message := question.Question + "\r\n" + q.getAllAnswers(question.IncorrectAnswers, question.CorrectAnswer)
			q.OutgoingMessages <- Message{chatID: q.ChatID, message: message}
			for {
				select {
				case userAnswer := <-q.IncomingAnswers:
					if sameAnswer(userAnswer.answer, question.CorrectAnswer) {
						q.OutgoingMessages <- Message{chatID: q.ChatID, message: "Correct"}
					} else {
						q.OutgoingMessages <- Message{chatID: q.ChatID,
							message: "Wrong, correct answer is " + question.CorrectAnswer}
					}
					return
				case <-timer.C:
					q.OutgoingMessages <- Message{chatID: q.ChatID,
						message: "No one answered the question. The correct answer is " +
							"\"" + question.CorrectAnswer + "\""}
					return
				}
			}
		}()
	}
	q.DoneQuiz <- q.ChatID
	q.OutgoingMessages <- Message{chatID: q.ChatID, message: "Quiz has ended!"}
}

func (q *Quiz) requestQuestions() ([]Question, error) {
	resp, err := http.Get("https://opentdb.com/api.php?amount=10")
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(resp.Body)

	var data questionRequestResponse
	err = json.Unmarshal(bytes, &data)
	for i := range data.Results {
		data.Results[i].Question = html.UnescapeString(data.Results[i].Question)
		data.Results[i].Category = html.UnescapeString(data.Results[i].Category)
		data.Results[i].Type = html.UnescapeString(data.Results[i].Type)
		data.Results[i].Difficulty = html.UnescapeString(data.Results[i].Difficulty)
		data.Results[i].CorrectAnswer = html.UnescapeString(data.Results[i].CorrectAnswer)
		for j := range data.Results[i].IncorrectAnswers {
			data.Results[i].IncorrectAnswers[j] = html.UnescapeString(data.Results[i].IncorrectAnswers[j])
		}
	}
	return data.Results, err
}

func (q *Quiz) getAllAnswers(incorrectAnswers []string, correctAnswer string) string {
	shuffledAnswers := incorrectAnswers
	shuffledAnswers = append(shuffledAnswers, correctAnswer)
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(shuffledAnswers), func(i, j int) {
		shuffledAnswers[i], shuffledAnswers[j] = shuffledAnswers[j], shuffledAnswers[i]
	})

	var answers string
	for _, answer := range shuffledAnswers {
		answers += "- " + answer + "\r\n"
	}
	return answers
}

func sameAnswer(userAnswer string, correctAnswer string) bool {
	return strings.ToLower(strings.TrimSpace(userAnswer)) == strings.ToLower(html.UnescapeString(strings.TrimSpace(correctAnswer)))
}

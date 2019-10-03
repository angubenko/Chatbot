package main

import (
	"encoding/json"
	"html"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const (
	timeoutInSeconds time.Duration = 15
)

type Quiz struct {
	ChatID           int64
	Questions        []Question
	OutgoingMessages chan Message
	IncomingAnswers  chan UserAnswer
	DoneQuiz         chan int64
	AddScore         chan UserID
}

func (q *Quiz) serveQuiz() {
	q.Questions, _ = q.requestQuestions()
	q.IncomingAnswers = make(chan UserAnswer)
	defer close(q.IncomingAnswers)
	log.Println("Started quiz for chat ", q.ChatID)
	earlyTermination := false

	for _, question := range q.Questions {
		if earlyTermination {
			break
		}

		func() {
			timer := time.NewTimer(timeoutInSeconds * time.Second)
			defer timer.Stop()

			userTried := make(map[string]bool)

			answersList, answersMessage := q.getAllAnswers(question.IncorrectAnswers, question.CorrectAnswer)
			q.OutgoingMessages <- Message{chatID: q.ChatID, message: question.Question + "\r\n" + answersMessage}

			for {
				select {
				case userAnswer := <-q.IncomingAnswers:
					switch userAnswer.answerType {
					case Reply:
						answerIdx, _ := strconv.Atoi(userAnswer.answer)
						if answerIdx == 0 || answerIdx > len(answersList) {
							continue
						}

						if _, ok := userTried[userAnswer.name]; ok {
							q.OutgoingMessages <- Message{chatID: q.ChatID,
								message: "You already attempted answering this question."}
							continue
						}

						userTried[userAnswer.name] = true
						if answersList[answerIdx-1] == question.CorrectAnswer {
							q.OutgoingMessages <- Message{chatID: q.ChatID, message: "Correct!"}
							q.AddScore <- UserID{
								chatID: q.ChatID,
								name:   userAnswer.name,
							}
							return
						} else {
							q.OutgoingMessages <- Message{chatID: q.ChatID,
								message: "Wrong answer"}
						}
					case Stop:
						log.Println("Stopping the quiz on channel ", q.ChatID)
						earlyTermination = true
						return
					}
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

func (q *Quiz) getAllAnswers(incorrectAnswers []string, correctAnswer string) ([]string, string) {
	shuffledAnswers := incorrectAnswers
	shuffledAnswers = append(shuffledAnswers, correctAnswer)
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(shuffledAnswers), func(i, j int) {
		shuffledAnswers[i], shuffledAnswers[j] = shuffledAnswers[j], shuffledAnswers[i]
	})

	var message string
	for i, answer := range shuffledAnswers {
		message += strconv.Itoa(i+1) + ". " + answer + "\r\n"
	}

	return shuffledAnswers, message
}

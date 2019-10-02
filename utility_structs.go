package main

import (
	"strconv"
	"strings"
)

type AnswerType int

const (
	Reply AnswerType = iota
	Skip
	Stop
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
	name       string
	answerType AnswerType
	answer     string
}

type Message struct {
	chatID  int64
	message string
}

type UserID struct {
	chatID   int64
	userName string
}

func (u UserID) MarshalText() (text []byte, err error) {
	return []byte(strconv.Itoa(int(u.chatID)) + "-" + u.userName), nil
}

func (u *UserID) UnmarshalText(text []byte) error {
	parsedText := strings.Split(string(text), "-")
	chatId, err := strconv.Atoi(parsedText[0])
	u.chatID = int64(chatId)
	u.userName = parsedText[1]
	return err
}

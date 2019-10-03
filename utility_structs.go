package main

type AnswerType int

const (
	Reply AnswerType = iota
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
	chatID int64
	name   string
}

type UserScore struct {
	name  string
	score int
}

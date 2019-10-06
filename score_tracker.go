package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"sync"
)

const (
	nTopPerformers int = 5
)

type ScoreTracker struct {
	addUserScore    chan UserID
	userScoreByChat map[int64]map[string]int
	mux             sync.Mutex
	cacheFile       string
}

func NewScoreTracker(addUserScore chan UserID, cacheFile string) (ScoreTracker, error) {
	if addUserScore == nil || cacheFile == "" {
		return ScoreTracker{}, errors.New("error: addUserScore channel and cacheFile are required")
	}
	userScoreByChat := make(map[int64]map[string]int)

	return ScoreTracker{
		addUserScore:    addUserScore,
		userScoreByChat: userScoreByChat,
		mux:             sync.Mutex{},
		cacheFile:       cacheFile,
	}, nil
}

func (st *ScoreTracker) start() {
	err := st.loadFromCache()
	if err != nil {
		log.Println("warning: couldn't load from cache, creating new cache")
	}
	go st.trackScore()
}

func (st *ScoreTracker) loadFromCache() error {
	bytes, err := ioutil.ReadFile(cacheFile)
	err = json.Unmarshal(bytes, &st.userScoreByChat)
	return err
}

func (st *ScoreTracker) trackScore() {
	for {
		userID, ok := <-st.addUserScore
		if !ok {
			return
		}

		st.mux.Lock()
		if _, ok := st.userScoreByChat[userID.chatID]; ok {
			if _, ok := st.userScoreByChat[userID.chatID][userID.name]; ok {
				st.userScoreByChat[userID.chatID][userID.name] += 1
			} else {
				st.userScoreByChat[userID.chatID][userID.name] = 1
			}
		} else {
			st.userScoreByChat[userID.chatID] = make(map[string]int)
			st.userScoreByChat[userID.chatID][userID.name] = 1
		}

		jsonData, _ := json.Marshal(st.userScoreByChat)
		err := ioutil.WriteFile(tmpCacheFile, jsonData, 0644)
		err = os.Rename(tmpCacheFile, cacheFile)
		if err != nil {
			log.Println("error: error occurred during saving cache to disk")
		}
		st.mux.Unlock()
	}
}

func (st *ScoreTracker) getScore(userName string, chatID int64) int {
	st.mux.Lock()
	defer st.mux.Unlock()
	var score int
	if _, ok := st.userScoreByChat[chatID]; ok {
		if val, ok := st.userScoreByChat[chatID][userName]; ok {
			score = val
		}
	}
	return score
}

func (st *ScoreTracker) getTopPerformersByChatID(chatID int64) []UserScore {
	userScoresOnChat := make(map[string]int)
	st.mux.Lock()
	if val, ok := st.userScoreByChat[chatID]; ok {
		userScoresOnChat = val
	}
	st.mux.Unlock()

	userScoreSorted := make([]UserScore, 0, nTopPerformers)
	if userScoresOnChat != nil {
		for name, score := range userScoresOnChat {
			userScoreSorted = append(userScoreSorted, UserScore{name, score})
		}
	}
	sort.Slice(userScoreSorted, func(i, j int) bool {
		return userScoreSorted[i].score >= userScoreSorted[j].score
	})

	if nTopPerformers < len(userScoreSorted) {
		return userScoreSorted[:nTopPerformers]
	}
	return userScoreSorted
}

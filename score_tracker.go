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

// ScoreTracker waits for updates on a user provided channel scoreUpdates.
// Whenever update occurs, ScoreTracker updates value in score map and updates cache.

const (
	nTopPerformers int = 5
)

type ScoreTracker struct {
	scoreUpdates    chan UserID
	userScoreByChat map[int64]map[string]int
	mux             sync.Mutex
	cacheFile       string
}

func (st *ScoreTracker) start() error {
	if st.scoreUpdates == nil || st.cacheFile == "" {
		return errors.New("error: scoreUpdates channel and cacheFile must be set")
	}
	err := st.loadFromCache()
	if err != nil {
		log.Println("error: couldn't load from cache, creating new cache")
		st.userScoreByChat = make(map[int64]map[string]int)
	}
	go st.trackScore()
	return nil
}

func (st *ScoreTracker) loadFromCache() error {
	bytes, err := ioutil.ReadFile(cacheFile)
	err = json.Unmarshal(bytes, &st.userScoreByChat)
	return err
}

func (st *ScoreTracker) trackScore() {
	for {
		userID, ok := <-st.scoreUpdates
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
		ioutil.WriteFile(tmpCacheFile, jsonData, 0644)
		os.Rename(tmpCacheFile, cacheFile)
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

package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"sync"
)

// ScoreTracker waits for updates on a user provided channel scoreUpdates.
// Whenever update occurs, ScoreTracker updates value in score map and updates cache.
type ScoreTracker struct {
	scoreUpdates chan UserID
	score        map[UserID]int
	mux          sync.Mutex
	cacheFile    string
}

func (st *ScoreTracker) start() error {
	if st.scoreUpdates == nil || st.cacheFile == "" {
		return errors.New("error: scoreUpdates channel and cacheFile must be set")
	}
	err := st.loadFromCache()
	if err != nil {
		log.Println("error: couldn't load from cache")
		st.score = make(map[UserID]int)
	}
	go st.trackScore()
	return nil
}

func (st *ScoreTracker) loadFromCache() error {
	bytes, err := ioutil.ReadFile(cacheFile)
	err = json.Unmarshal(bytes, &st.score)
	return err
}

func (st *ScoreTracker) trackScore() {
	for {
		userID, ok := <-st.scoreUpdates
		if !ok {
			return
		}
		st.mux.Lock()
		if _, ok := st.score[userID]; ok {
			st.score[userID] += 1
		} else {
			st.score[userID] = 1
		}
		jsonData, _ := json.Marshal(st.score)
		ioutil.WriteFile(cacheFile, jsonData, 0644)
		st.mux.Unlock()
	}
}

func (st *ScoreTracker) getScore(userName string, chatID int64) int {
	st.mux.Lock()
	var score int
	if val, ok := st.score[UserID{
		chatID:   chatID,
		userName: userName,
	}]; ok {
		score = val
	}
	st.mux.Unlock()
	return score
}

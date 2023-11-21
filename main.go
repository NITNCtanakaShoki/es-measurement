package main

import (
	"bytes"
	json2 "encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

const BaseURL = "http://127.0.0.1:6000"
const User1 = "user1"
const User2 = "user2"

func main() {
	title := "CHUNK_1024_APP_M200_C50_DB_M100_C50"
	logFile, err := os.OpenFile(fmt.Sprintf("%s.log", title), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	client := &http.Client{}
	logger := log.New(logFile, "LOG: ", log.LstdFlags|log.Lshortfile)

	reset(client, logger)

	prepareUser(client, User1, logger)
	prepareUser(client, User2, logger)

	unit := 100
	measureFrequency := 100
	count := 0
	c := make(chan uint8, unit)

	for i := 0; i < 100; i++ {
		for j := 0; j < measureFrequency; j++ {
			for k := 0; k < unit; k++ {
				go func() {
					for l := 0; l < 10; l++ {
						if err := sendRandom(client, logger); err == nil {
							c <- 1
							return
						}
					}
					panic("failed to send")
				}()
			}
			for k := 0; k < unit; k++ {
				<-c
				count++
			}
		}
		if err := measure(client, logger, count); err != nil {
			logger.Printf("failed to measure")
		}
	}
}

func reset(client *http.Client, logger *log.Logger) {
	url := fmt.Sprintf("%s/reset", BaseURL)
	req, err := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Authentication", "Reset-Force")
	if err != nil {
		logger.Fatalf("failed to reset: %s", err.Error())
		return
	}
	res, err := client.Do(req)
	if err != nil {
		logger.Fatalf("failed to reset: %s", err.Error())
		return
	}
	if res.StatusCode != 200 {
		// print res.body
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			logger.Fatalf("failed to reset: %s", err.Error())
			return
		}
		logger.Fatalf("failed to reset: %d, %s", res.StatusCode, string(bodyBytes))
		return
	}
}

func prepareUser(client *http.Client, username string, logger *log.Logger) {
	url := fmt.Sprintf("%s/user/%s", BaseURL, username)
	res, err := client.Post(url, "application/json", nil)
	if err != nil {
		logger.Fatalf("failed to prepare user: %s", err.Error())
		return
	}
	if res.StatusCode != 201 {
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			logger.Fatalf("failed to prepare user: %s", err.Error())
			return
		}
		logger.Fatalf("failed to prepare user: %d, %s", res.StatusCode, string(bodyBytes))
		return
	}
}

type SendJSON struct {
	Point int `json:"point"`
}

func measure(client *http.Client, logger *log.Logger, count int) error {
	if err := requestLog(client, logger); err != nil {
		return err
	}
	url := fmt.Sprintf("%s/user/%s", BaseURL, User1)
	start := time.Now()
	res, err := client.Get(url)
	if err != nil {
		logger.Println(err.Error())
		return err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	costDuration := time.Since(start)
	logger.Println(fmt.Sprintf("measure: count: %d, status: %d, point: %s, time: %dms, %s", count, res.StatusCode, string(b), costDuration.Milliseconds(), time.Now().Format("2006-01-02T15:04:05+09:00")))
	return nil
}

func requestLog(client *http.Client, logger *log.Logger) error {
	url := fmt.Sprintf("%s/user/%s/log", BaseURL, User1)
	res, err := client.Get(url)
	if err != nil {
		logger.Println(err.Error())
		return err
	}
	defer res.Body.Close()
	return nil
}

func sendRandom(client *http.Client, logger *log.Logger) error {
	from, to := randomFromTo()
	url := fmt.Sprintf("%s/send/%s/%s", BaseURL, from, to)
	content := SendJSON{rand.Intn(1_000_000)}
	marshalled, err := json2.Marshal(content)
	if err != nil {
		logger.Fatalf("failed to marshal: %s", err.Error())
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(marshalled))
	if err != nil {
		logger.Printf("failed to create request: %s", err.Error())
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		logger.Printf("failed to send request: %s", err.Error())
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		logger.Printf("failed to send request: %d", resp.StatusCode)
		return fmt.Errorf("failed to send request")
	}
	return nil
}

func randomFromTo() (string, string) {
	from := rand.Intn(2)
	if from == 0 {
		return User1, User2
	}
	return User2, User1
}

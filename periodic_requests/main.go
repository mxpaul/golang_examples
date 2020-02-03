package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Service struct {
	ErrChan    chan error
	UUIDChan   chan string
	WG         *sync.WaitGroup
	HttpClient *http.Client
}

func NewService() *Service {
	s := &Service{}
	s.ErrChan = make(chan error)
	s.UUIDChan = make(chan string)
	return s
}

func (s *Service) ReapService() {
	close(s.ErrChan)
	close(s.UUIDChan)
}

func (s *Service) NewClient(ctx context.Context) {
	httpTransport := &http.Transport{}
	//s.HttpClient = &http.Client{Transport: httpTransport, Timeout: time.Second * 3}
	s.HttpClient = &http.Client{Transport: httpTransport}
}

func (s *Service) MakeHttpRequest(ctx context.Context) {
	go func() {
		log.Printf("Make http request")
		webUrl := "http://httpbin.org/uuid"
		req, err := http.NewRequest("GET", webUrl, nil)
		if err != nil {
			s.ErrChan <- fmt.Errorf("NewRequest GET %s: %v", webUrl, err)
			return
		}
		req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 5.1.1; SM-G928X Build/LMY47X) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.83 Mobile Safari/537.36")
		req.Header.Add("Referer", webUrl)

		ctx, cancelTimeout := context.WithTimeout(ctx, 1000*time.Millisecond)
		defer cancelTimeout()
		req = req.WithContext(ctx)

		resp, err := s.HttpClient.Do(req)
		if err != nil {
			s.ErrChan <- fmt.Errorf("request error: %v", err)
			return
		}
		defer resp.Body.Close()

		if wantCode := 200; resp.StatusCode != wantCode {
			s.ErrChan <- fmt.Errorf("status not %d(%s)", wantCode, resp.Status)
			return
		}
		if wantCType, ctype := "application/json", resp.Header.Get("Content-Type"); ctype != wantCType {
			s.ErrChan <- fmt.Errorf("content-type is not %s(%s)", wantCType, ctype)
			return
		}

		var result struct{ Uuid string }
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			s.ErrChan <- fmt.Errorf("JSON decode error:(%s)", err)
			return
		}
		s.UUIDChan <- result.Uuid

	}()
}

func (s *Service) RequestSender(ctx context.Context) {
	s.WG = &sync.WaitGroup{}
	s.WG.Add(1)
	defer s.WG.Done()

	s.MakeHttpRequest(ctx)
	for {
		select {
		case <-time.After(3 * time.Second):
			s.MakeHttpRequest(ctx)
		case uuid := <-s.UUIDChan:
			log.Printf("Got uuid: %v", uuid)
		case err := <-s.ErrChan:
			log.Printf("Got error: %s", err)
		case <-ctx.Done():
			log.Printf("RequestSender break")
			return
		}
	}

}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("Start main")
	srv := NewService()
	ctx, Shutdown := context.WithCancel(context.Background())
	srv.NewClient(ctx)

	log.Printf("create signal notification channel")
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Start http sender service")
	go srv.RequestSender(ctx)

	log.Printf("Wait for signal")
	GotSignal := <-sigchan
	log.Print("")
	log.Printf("Got signal %v", GotSignal)
	Shutdown()
	log.Printf("Waiting WG")
	srv.WG.Wait()

	srv.ReapService()
	log.Printf("Exit")
}

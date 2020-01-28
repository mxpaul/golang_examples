package main

import (
	"context"
	"fmt"
	//"io/ioutil"
	"log"
	"net/http"
	//"os"
	"encoding/json"
	"time"

	"github.com/cretz/bine/process"
	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
)

// Hack for non-versioned modules (go-libtor)
var creator = libtor.Creator

type LibTorWrapper struct{}

func (LibTorWrapper) New(ctx context.Context, args ...string) (process.Process, error) {
	return creator.New(ctx, args...)
}

// End of hack

// Start tor with some defaults + elevated verbosity
func RunTorNode() (*tor.Tor, error) {
	log.Print("Starting TOR node, please wait a bit...")
	//t, err := tor.Start(nil, &tor.StartConf{ProcessCreator: LibTorWrapper{}, DebugWriter: os.Stderr})
	t, err := tor.Start(nil, &tor.StartConf{ProcessCreator: LibTorWrapper{}})
	return t, err
}

func CreateTunneledHTTPClient(t *tor.Tor) (*http.Client, error) {
	onion, err := t.Dialer(context.Background(), &tor.DialConf{})
	if err != nil {
		return nil, err
	}

	torTransport := &http.Transport{DialContext: onion.DialContext}
	client := &http.Client{Transport: torTransport, Timeout: time.Second * 3}
	return client, nil
}

func MakeHTTPRequest(client *http.Client) (*http.Response, error) {
	var webUrl = "https://httpbin.org/ip"
	//var webUrl = "https://httpbin.org/headers"
	//var webUrl = "https://httpbin.org/gzip"
	log.Printf("Send request to %s", webUrl)
	req, err := http.NewRequest("GET", webUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("NewRequest GET %s: %v", webUrl, err)
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; Android 5.1.1; SM-G928X Build/LMY47X) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.83 Mobile Safari/537.36")
	req.Header.Add("Referer", webUrl)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %v", err)
	}
	return resp, nil
}

func ProcessHTTPResponse(resp *http.Response) error {
	if wantCode := 200; resp.StatusCode != wantCode {
		return fmt.Errorf("status not %d(%s)", wantCode, resp.Status)
	}
	if wantCType, ctype := "application/json", resp.Header.Get("Content-Type"); ctype != wantCType {
		return fmt.Errorf("content-type is not %s(%s)", wantCType, ctype)
	}

	var result struct{ Origin string }
	//var result struct{ Headers map[string]string }
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("JSON decode error:(%s)", err)
	}

	log.Printf("Your ip: %s", result.Origin)
	//log.Printf("response: %+v", result)
	return nil
}

func main() {
	t, err := RunTorNode()
	if err != nil {
		log.Fatalf("Failed to start tor: %v", err)
	}
	defer t.Close()

	client, err := CreateTunneledHTTPClient(t)
	if err != nil {
		log.Fatalf("Failed to create onion dialer: %v", err)
	}

	resp, err := MakeHTTPRequest(client)
	if err != nil {
		log.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	err = ProcessHTTPResponse(resp)
	if err != nil {
		log.Fatalf("Error processing HTTP response: %v", err)
	}
}

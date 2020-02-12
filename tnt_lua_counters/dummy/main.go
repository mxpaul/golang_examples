package main

import (
	"github.com/tarantool/go-tarantool"
	"log"
)

func main() {
	opts := tarantool.Opts{SkipSchema: true}
	conn, err := tarantool.Connect("127.0.0.1:3301", opts)
	if err != nil {
		log.Fatalf("Connection refused: %s", err)
	}
	defer conn.Close()
	//resp, err := conn.Eval("return require('counter'):get(1)", []interface{}{})
	resp, err := conn.Call("counter:get", []interface{}{1})
	if err != nil {
		log.Fatalf("Error: %s Code: %v", err, resp.Code)
	}
	log.Printf("Success: %+v", resp)
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type PubSubData struct {
	From     string   `json:"from"`
	Seqno    string   `json:"seqno"`
	TopicIDs []string `json:"topicIDs"`
	Data     string   `json:"data"`
}

func Subscribe(ctx context.Context, data chan<- PubSubData, api, topic string) {
	var r io.ReadCloser
	defer func() {
		if r != nil {
			r.Close()
		}
	}()

	init := func() bool {
		if r != nil {
			r.Close()
			r = nil
		}

		for {
			req, err := http.NewRequestWithContext(
				ctx,
				http.MethodPost,
				fmt.Sprintf("%v/pubsub/pub?arg=%v", api, topic),
				nil,
			)
			if err != nil {
				panic(err)
			}

			rsp, err := http.DefaultClient.Do(req)
			if err != nil {
				if isContextError(err) {
					return false
				}

				log.Printf("Error: subscribe: %v", err)
				select {
				case <-ctx.Done():
					return false
				case <-time.After(10 * time.Second):
					continue
				}
			}

			r = rsp.Body
			return true
		}
	}

	if !init() {
		return
	}
	d := json.NewDecoder(r)

	for {
		var next PubSubData
		err := d.Decode(&next)
		if err != nil {
			if isContextError(err) {
				return
			}

			if !init() {
				return
			}

			continue
		}

		select {
		case <-ctx.Done():
			return

		case data <- next:
		}
	}
}

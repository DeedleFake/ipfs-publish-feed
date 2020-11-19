package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/ipfs/go-cid"
)

func apiURL(api, endpoint string, args ...interface{}) string {
	q := make(url.Values, len(args)/2)
	for i := 0; i < len(args); i += 2 {
		q.Add(fmt.Sprint(args[i]), fmt.Sprint(args[i+1]))
	}

	return fmt.Sprintf("%v/api/v0/%v?%v", api, endpoint, q.Encode())
}

// PubSubData is the structure of data returned from the pubsub
// system's subscribe endpoint.
type PubSubData struct {
	From     string   `json:"from"`
	Seqno    string   `json:"seqno"`
	TopicIDs []string `json:"topicIDs"`
	Data     string   `json:"data"`
}

// Subscribe subscribes to a pubsub topic, returning the received data
// one piece at a time to the provided channel. It stops when the
// context is canceled.
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
				apiURL(api, "pubsub/sub", "arg", topic),
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

			log.Printf("Error: decode data: %v", err)
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

var sizeSuffixes = []string{
	"B",
	"KB",
	"MB",
	"GB",
}

type FileSize uint

func (fs FileSize) String() string {
	var sizes, sep string
	for i := 0; (i < len(sizeSuffixes)) && (fs > 0); i++ {
		size := uint(fs % 1000)
		fs /= 1000

		sizes = fmt.Sprintf("%v%v%v%v", size, sizeSuffixes[i], sep, sizes)
		sep = " "
	}

	return sizes
}

type FileStat struct {
	Hash           string   `json:"Hash"`
	Size           FileSize `json:"Size"`
	CumulativeSize FileSize `json:"CumulativeSize"`
	Blocks         int      `json:"Blocks"`
	Type           string   `json:"Type"`
}

func Stat(ctx context.Context, api string, c cid.Cid) (FileStat, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		apiURL(api, "files/stat", "arg", fmt.Sprintf("/ipfs/%v", c)),
		nil,
	)
	if err != nil {
		panic(err)
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return FileStat{}, fmt.Errorf("do request: %w", err)
	}
	defer rsp.Body.Close()

	buf, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return FileStat{}, fmt.Errorf("read body: %w", err)
	}

	var stat FileStat
	err = json.Unmarshal(buf, &stat)
	if err != nil {
		return stat, fmt.Errorf("unmarshal: %w", err)
	}
	return stat, nil
}

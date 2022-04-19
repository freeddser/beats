// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package submitter

import (
	"bufio"
	"bytes"
	"context"
	json2 "encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"net/http"
	"os"
	"runtime"
	"time"
)

type submitter struct {
	log      *logp.Logger
	out      *os.File
	observer outputs.Observer
	writer   *bufio.Writer
	codec    codec.Codec
	index    string
}

type submitterEvent struct {
	Timestamp time.Time `json:"@timestamp" struct:"@timestamp"`

	// Note: stdlib json doesn't support inlining :( -> use `codec: 2`, to generate proper event
	Fields interface{} `struct:",inline"`
}

func init() {
	outputs.RegisterType("submitter", makesubmitter)
}

func makesubmitter(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	//config := defaultConfig
	err := cfg.Unpack(&defaultConfig)
	if err != nil {
		return outputs.Fail(err)
	}

	var enc codec.Codec
	if defaultConfig.Codec.Namespace.IsSet() {
		enc, err = codec.CreateEncoder(beat, defaultConfig.Codec)
		if err != nil {
			return outputs.Fail(err)
		}
	} else {
		enc = json.New(beat.Version, json.Config{
			Pretty:     defaultConfig.Pretty,
			EscapeHTML: false,
		})
	}
	index := beat.Beat
	c, err := newsubmitter(index, observer, enc)
	if err != nil {
		return outputs.Fail(fmt.Errorf("submitter output initialization failed with: %v", err))
	}
	// check stdout actually being available
	if runtime.GOOS != "windows" {
		if _, err = c.out.Stat(); err != nil {
			err = fmt.Errorf("submitter output initialization failed with: %v", err)
			return outputs.Fail(err)
		}
	}

	return outputs.Success(defaultConfig.BatchSize, 0, c)
}

func newsubmitter(index string, observer outputs.Observer, codec codec.Codec) (*submitter, error) {
	c := &submitter{log: logp.NewLogger("submitter"), out: os.Stdout, codec: codec, observer: observer, index: index}
	c.writer = bufio.NewWriterSize(c.out, 8*1024)
	return c, nil
}

func (c *submitter) Close() error { return nil }
func (c *submitter) Publish(_ context.Context, batch publisher.Batch) error {
	fmt.Println("---- module:submitter ----")
	st := c.observer
	events := batch.Events()
	st.NewBatch(len(events))
	dropped := 0
	for i := range events {
		ok := c.publishEvent(&events[i])
		if !ok {
			dropped++
		}
	}

	c.writer.Flush()
	batch.ACK()

	st.Dropped(dropped)
	st.Acked(len(events) - dropped)

	return nil
}

var nl = []byte("\n")

func (c *submitter) publishEvent(event *publisher.Event) bool {
	serializedEvent, err := c.codec.Encode(c.index, &event.Content)
	if err != nil {
		if !event.Guaranteed() {
			return false
		}

		c.log.Errorf("Unable to encode event: %+v", err)
		c.log.Debugf("Failed event: %v", event)
		return false
	}

	if err := c.writeBuffer(serializedEvent); err != nil {
		c.observer.WriteError(err)
		c.log.Errorf("Unable to publish events to submitter: %+v", err)
		return false
	}

	if err := c.writeBuffer(nl); err != nil {
		c.observer.WriteError(err)
		c.log.Errorf("Error when appending newline to event: %+v", err)
		return false
	}

	c.observer.WriteBytes(len(serializedEvent) + 1)
	return true
}

func (c *submitter) writeBuffer(buf []byte) error {
	written := 0

	if len(buf) > 1 {
		var data Data
		err := json2.Unmarshal(buf, &data)
		if err != nil {
			fmt.Println("json error:", err)
		}
		//CALL HTTP POST API
		//todo you can handle the error here
		go Post(defaultConfig.PostAPI, &data, nil, nil)
	}

	for written < len(buf) {
		n, err := c.writer.Write(buf[written:])
		if err != nil {
			return err
		}

		written += n
	}
	return nil
}

func (c *submitter) String() string {
	return "submitter"
}

func Post(url string, body interface{}, params map[string]string, headers map[string]string) (*http.Response, error) {
	//add post body
	var bodyJson []byte
	var req *http.Request
	if body != nil {
		var err error
		bodyJson, err = json2.Marshal(body)
		if err != nil {
			fmt.Println(err)
			return nil, errors.New("http post body to json failed")
		}
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyJson))
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("new request is fail: %v \n")
	}
	req.Header.Set("Content-type", "application/json")
	//add params
	q := req.URL.Query()
	if params != nil {
		for key, val := range params {
			q.Add(key, val)
		}
		req.URL.RawQuery = q.Encode()
	}
	//add headers
	if headers != nil {
		for key, val := range headers {
			req.Header.Add(key, val)
		}
	}
	//http client
	client := &http.Client{}
	fmt.Printf("HTTP client:%s URL : %s Body:%s\n", http.MethodPost, req.URL.String(), bodyJson)
	return client.Do(req)
}

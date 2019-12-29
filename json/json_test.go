// Copyright (C) 2019 rameshvk. All rights reserved.
// Use of this source code is governed by a MIT-style license
// that can be found in the LICENSE file.

package json_test

import (
	"bytes"
	gojson "encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/tvastar/http/json"
)

func Example() {
	// input and output can be strongly typed below
	makeCall := func(url string, query, input, output interface{}) (*http.Response, error) {
		req, err := json.NewRequest(
			"POST",
			url,
			json.Query(query),
			json.Body(input),
		)
		if err != nil {
			return nil, err
		}

		client := &http.Client{
			Transport: json.Transport{
				Result:    output,
				Transport: http.DefaultTransport,
			},
		}
		return client.Do(req)
	}

	// actual test

	server := simpleTestServer(func(query string, jsonBody interface{}) interface{} {
		return map[string]interface{}{
			"Query": query,
			"Body":  jsonBody,
		}
	})
	defer server.Close()

	input := map[string]interface{}{"hello": 42}
	var output interface{}

	res, err := makeCall(server.URL, sampleQuery(), input, &output)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	greeting, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	// both of these should agree
	fmt.Printf("%s", greeting)
	fmt.Println(output)

	// Output:
	// {"Body":{"hello":42},"Query":"foo=42&heya=true"}
	// map[Body:map[hello:42] Query:foo=42&heya=true]
}

func ExampleNewRequest() {
	r, err := json.NewRequest(
		"POST",
		"http://localhost:99/boo?x=1",
		json.Query(sampleQuery()),
		json.Body(map[string]interface{}{"hello": 42}),
	)
	var buf bytes.Buffer
	err2 := r.Write(&buf)
	fmt.Println("Error:", err, err2)
	fmt.Println(strings.Replace(buf.String(), "\r", "", 100))

	// Output:
	// Error: <nil> <nil>
	// POST /boo?foo=42&heya=true&x=1 HTTP/1.1
	// Host: localhost:99
	// User-Agent: Go-http-client/1.1
	// Content-Length: 13
	// Content-Type: application/json
	//
	// {"hello":42}
}

func sampleQuery() interface{} {
	return struct {
		Foo  int  `url:"foo"`
		Heya bool `url:"heya"`
	}{42, true}
}

func simpleTestServer(fn func(query string, body interface{}) interface{}) *httptest.Server {
	// create a server that parse body as JSON and passes that to the fn
	// the response is then written back as JSON
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body interface{}
		err := gojson.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer
		enc := gojson.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		err = enc.Encode(fn(r.URL.Query().Encode(), body))
		if err != nil {
			panic(err)
		}
		w.Header().Add("Content-Type", "application/json")
		_, err = w.Write(buf.Bytes())
		if err != nil {
			panic(err)
		}
	}))
}

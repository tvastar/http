// Copyright (C) 2019 rameshvk. All rights reserved.
// Use of this source code is governed by a MIT-style license
// that can be found in the LICENSE file.

// Package json adds composition friendly JSON support to HTTP clients
//
// A HTTP client request can be constructed using NewRequest which
// takes fluent functional args to specify the Body and Query
// parameters:
//
//     // queryArgs can be any type that can be serialized by
//     // github.com/google/go-querystring/query"
//     queryArgs := struct {
//         Foo int
//         Hoo string
//     }{ 42, "something" }
//
//     // body can be any type that can be serialied to JSON
//     body := struct {
//         Hey string
//     }{ "Hey!" }
//
//     // make a HTTP request with these
//     req, err := json.NewRequest(
//         "GET",
//         url,
//         json.Query(queryArgs),
//         json.Body(body),
//     )
//
// The response can then be sent using standard http.Client
// mechanisms.
//
// This package also exposes a `Transport` type for parsing
// `application/json` content-types.  The middleware pattern is built
// on top of the standard http.RoundTripper interface which allows
// other processing activity to be properly chained.  For instance, it
// is possible to properly chain Retries (assuming that was also
// implemented as a http.RoundTripper):
//
//      // output can be any JSON decodable type
//      var output struct { ... }
//      client := &http.Client{
//          Transport: json.Transport{
//              Result: &output,
//              // Transports can be chained!
//              Transport: retry.Transport{
//                  // See github.com/tvastar/http/retry for details
//                  // of how to use the retry transport
//                  Transport: http.DefaultTransport,
//              }
//          },
//      }
//      res, err := client.Do(req)
//      // if there was no error, output will be filled in!
package json

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"mime"
	"net/http"

	"github.com/google/go-querystring/query"
)

// NewRequest creates a new http Request with the provided options.
//
// See Body and Query for interesting JSON options.
func NewRequest(method, url string, options ...Option) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	for kk := 0; kk < len(options) && err == nil; kk++ {
		req, err = options[kk](req)
	}
	return req, err
}

// Option is an option to pass to NewRequest.
//
// Custom options can either mutate the request or create a new
// request and return that
type Option func(req *http.Request) (*http.Request, error)

// Body updates the request to use the JSON encoding of the provided
// value. It also sets the Content-Type value to "application/json"
func Body(v interface{}) Option {
	return func(req *http.Request) (*http.Request, error) {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(v)
		req.ContentLength = int64(buf.Len())
		req.Body = ioutil.NopCloser(&buf)
		req.Header.Set("Content-Type", "application/json")
		return req, err
	}
}

// Query updates the query with the provided args using standard URL
// encoding schemes.  Existing query values are retained.
func Query(v interface{}) Option {
	return func(req *http.Request) (*http.Request, error) {
		val, err := query.Values(v)
		log.Println("Got", val, err)
		orig := req.URL.Query()
		for k, entries := range val {
			for _, entry := range entries {
				orig.Add(k, entry)
			}
		}
		req.URL.RawQuery = orig.Encode()
		return req, err
	}
}

// Transport implements a JSON-decoder HTTP transport.
//
// This wraps another transport and only parses the response.
// If the response has the content type `application/json`, then the
// response body is decoded into `Result`.  Note that `Result` must be
// a reference type (i.e. something that can be passed to json.Unmarshal).
type Transport struct {
	Result    interface{}       // where the result is stored
	Transport http.RoundTripper // the base transport
}

// RoundTrip implements the http.RoundTripper interface
func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := t.Transport.RoundTrip(req)
	if err != nil {
		return res, err
	}

	ct, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if ct != "application/json" || err != nil {
		// do not treat it as JSON
		return res, nil
	}

	defer res.Body.Close()
	buf, err := ioutil.ReadAll(res.Body)
	res.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
	if err != nil {
		return res, err
	}

	return res, json.Unmarshal(buf, t.Result)
}

// Copyright (C) 2019 rameshvk. All rights reserved.
// Use of this source code is governed by a MIT-style license
// that can be found in the LICENSE file.

// Package retry implements a http.RoundTripper that retries requests
//
// This package uses a configurable ExponentialBackoff mechanism:
//
//     r := retry.Transport {
//         Backoff: backoff.NewExponentialBackoff(),
//         ShouldRetry: /* optional custom retry check function */,
//         Transport: /* chain transports! */,
//     }
//     client := http.Client{Transport: r}
//     res, err := client.Do(http.NewRequest("GET", url, nil))
package retry

import (
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
)

// Transport implements the http Retry middleware
type Transport struct {
	Backoff     *backoff.ExponentialBackOff
	ShouldRetry func(res *http.Response, err error, lastAttempt bool) (error, bool)
	Transport   http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface
func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := *t.Backoff
	b := &copy

	shouldRetry := func(res *http.Response, err error, lastAttempt bool) (error, bool) {
		return err, err != nil && !lastAttempt
	}

	if t.ShouldRetry != nil {
		shouldRetry = t.ShouldRetry
	}

	for {
		res, err := t.Transport.RoundTrip(req)
		delay := b.NextBackOff()
		lastAttempt := delay == backoff.Stop
		if err, ok := shouldRetry(res, err, lastAttempt); !ok {
			return res, err
		}
		t := time.NewTimer(delay)
		select {
		case <-t.C:
		case <-req.Context().Done():
			return res, req.Context().Err()
		}
	}
}

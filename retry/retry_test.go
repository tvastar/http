// Copyright (C) 2019 rameshvk. All rights reserved.
// Use of this source code is governed by a MIT-style license
// that can be found in the LICENSE file.

package retry_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/tvastar/http/retry"

	"github.com/cenkalti/backoff"
)

func Example() {
	count := 0
	r := retry.Transport{
		Backoff: backoff.NewExponentialBackOff(),
		ShouldRetry: func(res *http.Response, err error, lastAttempt bool) (error, bool) {
			count++
			return err, err != nil && !lastAttempt
		},
		Transport: http.DefaultTransport,
	}
	r.Backoff.MaxElapsedTime = time.Second

	client := http.Client{Transport: r}
	req, err := http.NewRequest("GET", "x.boo.bohemian/a/b/c/d", nil)
	if err != nil {
		panic(err)
	}
	_, err = client.Do(req)
	fmt.Println("Got:", count, err == nil)

	// Output: Got: 3 false
}

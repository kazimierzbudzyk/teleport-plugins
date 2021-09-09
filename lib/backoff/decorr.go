/*
Copyright 2021 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backoff

import (
	"context"
	"math/rand"
	"time"

	"github.com/gravitational/trace"
)

// Decorr is a "decorrelated jitter" inspired by https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/.
type decorr struct {
	base  int64
	cap   int64
	mul   int64
	sleep int64
}

// Decorr initializes an alogoritm.
func Decorr(base, cap time.Duration) Backoff {
	return DecorrWithMul(base, cap, 3)
}

// DecorrWithMul initializes a backoff alogoritm with a give multiplier.
func DecorrWithMul(base, cap time.Duration, mul int64) Backoff {
	return &decorr{
		base:  int64(base),
		cap:   int64(cap),
		mul:   mul,
		sleep: int64(base),
	}
}

func (backoff *decorr) Do(ctx context.Context) error {
	backoff.sleep = backoff.base + rand.Int63n(backoff.sleep*backoff.mul-backoff.base)
	if backoff.sleep > backoff.cap {
		backoff.sleep = backoff.cap
	}
	select {
	case <-time.After(time.Duration(backoff.sleep)):
		return nil
	case <-ctx.Done():
		return trace.Wrap(ctx.Err())
	}
}
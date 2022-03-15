// SPDX-License-Identifier: Apache-2.0
// Copyright © 2022 Wrangle Ltd

package testutils

import (
	"testing"
	"time"
)

func Retry(t *testing.T, dur time.Duration, count int, cond func() bool, message string, args ...interface{}) {
	t.Helper()
	for i := 0; i < count; i++ {
		if v := cond(); v {
			return
		}
		time.Sleep(dur)
	}
	if message == "" {
		message = "retry limit exceed"
	}
	t.Fatalf(message, args...)
}

/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkLockFree(b *testing.B) {
	var (
		counter = uint64(0)
	)
	eh := &longEventHandler[uint64]{}
	disruptor := NewLockfree[uint64](1024*1024, Uint32Array, eh, &SleepBlockStrategy{
		t: time.Microsecond,
	})
	disruptor.Start()
	producer := disruptor.Producer()
	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			for j := 0; j < SchPerGo; j++ {
				x := atomic.AddUint64(&counter, 1)
				err := producer.Write(x)
				if err != nil {
					panic(err)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	b.StopTimer()
	time.Sleep(time.Second * 1)
	disruptor.Close()
}

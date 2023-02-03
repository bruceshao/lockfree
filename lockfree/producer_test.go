/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	GoSize   = 5000
	SchPerGo = 10000
)

type longEventHandler[T uint64] struct {
	count int32
	ts    time.Time
}

func (h *longEventHandler[T]) OnEvent(v uint64) {
	atomic.AddInt32(&h.count, 1)
	if h.count == 1 {
		h.ts = time.Now()
	}
	if v%1000000 == 0 {
		fmt.Printf("read %d\n", v)
	}
	if h.count == GoSize*SchPerGo {
		tl := time.Since(h.ts)
		fmt.Printf("read time = %d ms\n", tl.Milliseconds())
	}
}

func TestAA(t *testing.T) {
	var (
		t1_10us     = uint64(0) // 1-10微秒
		t10_100us   = uint64(0) // 10-100微秒
		t100_1000us = uint64(0) // 100-1000微秒
		t1_10ms     = uint64(0) // 1-10毫秒
		t10_100ms   = uint64(0) // 10-100毫秒
		t100_ms     = uint64(0) // 大于100毫秒
		slower      = uint64(0)
		counter     = uint64(0)
	)
	eh := &longEventHandler[uint64]{}
	//queue, err := NewProducer[uint64](1024*1024, 1, eh, &SleepWaitStrategy{
	//	t: time.Nanosecond * 1,
	//})
	disruptor := NewSerialDisruptor[uint64](1024*1024, eh, &SchedWaitStrategy{})
	disruptor.Start()
	producer := disruptor.Producer()
	var wg sync.WaitGroup
	wg.Add(GoSize)
	totalS := time.Now()
	for i := 0; i < GoSize; i++ {
		go func() {
			for j := 0; j < SchPerGo; j++ {
				x := atomic.AddUint64(&counter, 1)
				ts := time.Now()
				err := producer.Write(x)
				if err != nil {
					panic(err)
				}
				tl := time.Since(ts)
				ms := tl.Microseconds()
				if ms > 1 {
					atomic.AddUint64(&slower, 1)
					if ms < 10 { // t1_10us
						atomic.AddUint64(&t1_10us, 1)
					} else if ms < 100 {
						atomic.AddUint64(&t10_100us, 1)
					} else if ms < 1000 {
						atomic.AddUint64(&t100_1000us, 1)
					} else if ms < 10000 {
						atomic.AddUint64(&t1_10ms, 1)
					} else if ms < 100000 {
						atomic.AddUint64(&t10_100ms, 1)
					} else {
						atomic.AddUint64(&t100_ms, 1)
					}
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	totalL := time.Since(totalS)
	fmt.Printf("write total time = [%d ms]\n", totalL.Milliseconds())
	fmt.Println("----- write complete -----")
	time.Sleep(time.Second * 3)
	disruptor.Close()
	fmt.Printf("slow ratio = %.2f \n", float64(slower)*100.0/float64(counter))
	fmt.Printf("quick ratio = %.2f \n", float64(counter-slower)*100.0/float64(counter))
	fmt.Printf("[<1us][%d] \n", counter-slower)
	fmt.Printf("[1-10us][%d] \n", t1_10us)
	fmt.Printf("[10-100us][%d] \n", t10_100us)
	fmt.Printf("[100-1000us][%d] \n", t100_1000us)
	fmt.Printf("[1-10ms][%d] \n", t1_10ms)
	fmt.Printf("[10-100ms][%d] \n", t10_100ms)
	fmt.Printf("[>100ms][%d] \n", t100_ms)
}

func TestB(t *testing.T) {
	var x = uint64(100)
	typeOf := reflect.TypeOf(x)
	fmt.Println(typeOf)
	fmt.Println(os.Getpagesize())
}

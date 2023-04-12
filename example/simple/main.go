/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bruceshao/lockfree/lockfree"
)

var (
	goSize    = 10000
	sizePerGo = 10000

	total = goSize * sizePerGo
)

func main() {
	// lockfree计时
	now := time.Now()

	// 创建事件处理器
	handler := &eventHandler[uint64]{
		signal: make(chan struct{}, 0),
		now:    now,
	}

	// 创建消费端串行处理的Lockfree
	lf := lockfree.NewLockfree[uint64](
		1024*1024,
		handler,
		lockfree.NewSleepBlockStrategy(time.Millisecond),
	)

	// 启动Lockfree
	if err := lf.Start(); err != nil {
		panic(err)
	}

	// 获取生产者对象
	producer := lf.Producer()

	// 并发写入
	var wg sync.WaitGroup
	wg.Add(goSize)
	for i := 0; i < goSize; i++ {
		go func(start int) {
			for j := 0; j < sizePerGo; j++ {
				err := producer.Write(uint64(start*sizePerGo + j + 1))
				if err != nil {
					panic(err)
				}
			}
			wg.Done()
		}(i)
	}

	// wait for producer
	wg.Wait()

	fmt.Printf("producer has been writed, write count: %v, time cost: %v \n", total, time.Since(now).String())

	// wait for consumer
	handler.wait()

	// 关闭Lockfree
	lf.Close()
}

type eventHandler[T uint64] struct {
	signal   chan struct{}
	gcounter uint64
	now      time.Time
}

func (h *eventHandler[T]) OnEvent(v uint64) {
	cur := atomic.AddUint64(&h.gcounter, 1)
	if cur == uint64(total) {
		fmt.Printf("eventHandler has been consumed already, read count: %v, time cose: %v\n", total, time.Since(h.now))
		close(h.signal)
		return
	}

	if cur%10000000 == 0 {
		fmt.Printf("eventHandler consume %v\n", cur)
	}
}

func (h *eventHandler[T]) wait() {
	<-h.signal
}

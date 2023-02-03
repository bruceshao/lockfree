/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package main

import (
	"fmt"
	"lockfree/lockfree"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	var (
		goSize    = 10
		sizePerGo = 10
		counter   = uint64(0)
	)

	eh := &longEventHandler[uint64]{}
	disruptor := lockfree.NewSerialDisruptor[uint64](1024*1024, eh, &lockfree.SchedWaitStrategy{})
	if err := disruptor.Start(); err != nil {
		panic(err)
	}
	producer := disruptor.Producer()
	var wg sync.WaitGroup
	wg.Add(goSize)
	for i := 0; i < goSize; i++ {
		go func() {
			for j := 0; j < sizePerGo; j++ {
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
	fmt.Println("----- write complete -----")
	time.Sleep(time.Second * 1)
	disruptor.Close()
}

type longEventHandler[T uint64] struct {
}

func (h *longEventHandler[T]) OnEvent(v uint64) {
	fmt.Printf("value = %v\n", v)
}

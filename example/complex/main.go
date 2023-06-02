/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package main

import (
	"fmt"
	"github.com/bruceshao/lockfree"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	fmt.Println("========== start write by discard ==========")
	writeByDiscard()
	fmt.Println("========== complete write by discard ==========")
	fmt.Println("========== start write by cursor ==========")
	writeByCursor()
	fmt.Println("========== complete write by cursor ==========")
}

func writeByDiscard() {
	var counter = uint64(0)
	// 写入超时，如何使用
	eh := &fixedSleepEventHandler[uint64]{
		sm: time.Millisecond * 10,
	}
	disruptor := lockfree.NewLockfree[uint64](2, 0, eh, lockfree.NewSleepBlockStrategy(time.Microsecond))
	disruptor.Start()
	producer := disruptor.Producer()
	// 假设有100个写g
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 10; j++ {
				v := atomic.AddUint64(&counter, 1)
				ww := producer.WriteWindow()
				if ww <= 0 {
					// 表示无法写入，丢弃
					fmt.Println("discard ", v)
					continue
				}
				// 表示可以写入，写入即可
				err := producer.Write(v)
				if err != nil {
					panic(err)
				}
				fmt.Println("write ", v)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	time.Sleep(3 * time.Second)
	disruptor.Close()
}

func writeByCursor() {
	var counter = uint64(0)
	// 写入超时，如何使用
	eh := &randomSleepEventHandler[uint64]{}
	disruptor := lockfree.NewLockfree[uint64](2, 0, eh, lockfree.NewSleepBlockStrategy(time.Microsecond))
	disruptor.Start()
	producer := disruptor.Producer()
	// 假设有100个写g
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			for j := 0; j < 10; j++ {
				v := atomic.AddUint64(&counter, 1)
				wc, exist, err := producer.WriteTimeout(v, time.Millisecond)
				if err != nil {
					return
				}
				if !exist {
					// 重复写入1次，写入不成功则丢弃重新写其他的
					if ok, _ := producer.WriteByCursor(v, wc); ok {
						continue
					}
					fmt.Println("discard ", v)
					// 重新生成值，一直等待写入
					v = atomic.AddUint64(&counter, 1)
					for {
						if ok, _ := producer.WriteByCursor(v, wc); ok {
							fmt.Println("write ", v, " with x times")
							break
						}
						// 写入不成功则休眠，防止CPU暴增
						time.Sleep(100 * time.Microsecond)
					}
				} else {
					fmt.Println("write ", v, " with 1 time")
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	time.Sleep(3 * time.Second)
	disruptor.Close()
}

type fixedSleepEventHandler[T uint64] struct {
	sm time.Duration
}

func (h *fixedSleepEventHandler[T]) OnEvent(v uint64) {
	time.Sleep(h.sm)
	fmt.Println("consumer ", v)
}

func (h *fixedSleepEventHandler[T]) OnBatchEvent(v []uint64) {
	for i := range v {
		h.OnEvent(v[i])
	}
}

type randomSleepEventHandler[T uint64] struct {
	count int32
}

func (h *randomSleepEventHandler[T]) OnEvent(v uint64) {
	// 每次处理都会进行随机休眠，可以导致消费端变慢
	intn := rand.Intn(1000)
	time.Sleep(time.Duration(intn * 1000))
	fmt.Println("consumer count ", atomic.AddInt32(&h.count, 1))
}

func (h *randomSleepEventHandler[T]) OnBatchEvent(v []uint64) {
	for i := range v {
		h.OnEvent(v[i])
	}
}

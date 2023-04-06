/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package main

import (
	"fmt"
	"github.com/bruceshao/lockfree/lockfree"
	"os"
	"runtime/pprof"
	"sync"
	"time"
)

var (
	goSize    = 10000
	sizePerGo = 20000
)

func main() {
	f, _ := os.OpenFile("cpu.pprof", os.O_CREATE|os.O_RDWR, 0644)
	defer f.Close()
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	//go start()
	var w = sync.WaitGroup{}
	w.Add(1)
	// lockfree
	go func() {
		//counter := uint64(0)
		t := time.Now()
		// 创建事件处理器
		eh := &longEventHandler[uint64]{}
		// 创建消费端串行处理的Disruptor
		disruptor := lockfree.NewLockfree[uint64](1024*1024, eh,
			lockfree.NewSleepBlockStrategy(time.Microsecond))
		// 启动Disruptor
		if err := disruptor.Start(); err != nil {
			panic(err)
		}
		// 获取生产者对象
		producer := disruptor.Producer()
		var wg sync.WaitGroup
		wg.Add(goSize)
		for i := 0; i < goSize; i++ {
			go func(start int) {
				for j := 0; j < sizePerGo; j++ {
					//x := atomic.AddUint64(&counter, 1)
					//if x%10000000 == 0 {
					//	fmt.Printf("write[%d]\n", x)
					//}
					//写入数据
					err := producer.Write(uint64(start*sizePerGo + j + 1))
					//err := producer.Write(x)
					if err != nil {
						panic(err)
					}
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
		fmt.Println("=====lockfree[", time.Now().Sub(t), "]=====")
		fmt.Println("----- lockfree write complete -----")
		time.Sleep(1 * time.Second)
		// 关闭Disruptor
		disruptor.Close()
		w.Done()
	}()
	// chan，单写多读
	//go func() {
	//	var c = make(chan uint64, 1000000)
	//	t := time.Now()
	//	var wg = sync.WaitGroup{}
	//	wg.Add(goSize)
	//	for i := 0; i < goSize; i++ {
	//		go func() {
	//			for {
	//				select {
	//				case v, ok := <-c:
	//					if !ok {
	//						goto END
	//					} else if v%10000000 == 0 {
	//						fmt.Println("chan one write[", v, "]")
	//					}
	//				}
	//			}
	//		END:
	//			wg.Done()
	//		}()
	//	}
	//	for i := 0; i < sizePerGo*goSize; i++ {
	//		c <- uint64(i)
	//	}
	//	close(c)
	//	wg.Wait()
	//	fmt.Println("=====chan one write[", time.Now().Sub(t), "]=====")
	//	fmt.Println("----- chan one write complete -----")
	//	w.Done()
	//}()
	// chan 多写单读
	//go func() {
	//	counter := uint64(0)
	//	var c = make(chan uint64, 1000000)
	//	t := time.Now()
	//	go func() {
	//		for {
	//			select {
	//			case v, ok := <-c:
	//				if !ok {
	//					return
	//				} else if v%10000000 == 0 {
	//					fmt.Println("chan many write[", v, "]")
	//				}
	//			}
	//		}
	//	}()
	//	var wg = sync.WaitGroup{}
	//	wg.Add(goSize)
	//	for i := 0; i < goSize; i++ {
	//		go func() {
	//			for j := 0; j < sizePerGo; j++ {
	//				x := atomic.AddUint64(&counter, 1)
	//				c <- x
	//			}
	//			wg.Done()
	//		}()
	//	}
	//	wg.Wait()
	//	close(c)
	//	fmt.Println("=====chan many write[", time.Now().Sub(t), "]=====")
	//	fmt.Println("----- chan many write complete -----")
	//	w.Done()
	//}()
	w.Wait()
	//time.Sleep(10 * time.Minute)
}

type longEventHandler[T uint64] struct {
}

func (h *longEventHandler[T]) OnEvent(v uint64) {
	if v%10000000 == 0 {
		fmt.Println("lockfree [", v, "]")
	}
}

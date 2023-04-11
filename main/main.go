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
	"sync/atomic"
	"time"
)

var (
	goSize      = 10000
	sizePerGo   = 10000
	cap         = 1024 * 1024
	timeAnalyse = false
)

func main() {
	f, _ := os.OpenFile("cpu.pprof", os.O_CREATE|os.O_RDWR, 0644)
	defer f.Close()
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	// 启动命令: .exe [all/lockfree/chan] [time]
	arg := "all"
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}
	if len(os.Args) > 2 {
		timeAnalyse = true
	}
	if arg == "all" {
		fmt.Println("start lockfree and channel test")
		// 表示全部启动
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			lockfreeMain()
		}()
		go func() {
			defer wg.Done()
			chanMain()
		}()
		wg.Wait()
	} else if arg == "chan" {
		fmt.Println("start channel test")
		// 表示启动chan
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			chanMain()
		}()
		wg.Wait()
	} else if arg == "lockfree" {
		fmt.Println("start lockfree test")
		// 表示启动lockfree
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			lockfreeMain()
		}()
		wg.Wait()
	}
	fmt.Println("all queue is over")
}

func lockfreeMain() {
	var (
		t1_10us     = uint64(0) // 1-10微秒
		t10_100us   = uint64(0) // 10-100微秒
		t100_1000us = uint64(0) // 100-1000微秒
		t1_10ms     = uint64(0) // 1-10毫秒
		t10_100ms   = uint64(0) // 10-100毫秒
		t100_ms     = uint64(0) // 大于100毫秒
		slower      = uint64(0)
	)
	// 创建事件处理器
	eh := &longEventHandler{}
	// 创建消费端串行处理的Lockfree
	lf := lockfree.NewLockfree(cap, eh, lockfree.NewSleepBlockStrategy(time.Millisecond))
	// 启动Lockfree
	if err := lf.Start(); err != nil {
		panic(err)
	}
	// lockfree计时
	ts := time.Now()
	// 获取生产者对象
	producer := lf.Producer()
	var wg sync.WaitGroup
	wg.Add(goSize)
	for i := 0; i < goSize; i++ {
		go func(start int) {
			if timeAnalyse {
				for j := 0; j < sizePerGo; j++ {
					//写入数据
					tb := time.Now()
					err := producer.Write(uint64(start*sizePerGo + j + 1))
					tl := time.Since(tb)
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
					if err != nil {
						panic(err)
					}
				}
			} else {
				for j := 0; j < sizePerGo; j++ {
					err := producer.Write(uint64(start*sizePerGo + j + 1))
					if err != nil {
						panic(err)
					}
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	fmt.Println("===== lockfree[", time.Now().Sub(ts), "] =====")
	fmt.Println("----- lockfree write complete -----")
	time.Sleep(3 * time.Second)
	// 关闭Lockfree
	lf.Close()
	if timeAnalyse {
		fmt.Printf("lockfree slow ratio = %.2f \n", float64(slower)*100.0/float64(goSize*sizePerGo))
		fmt.Printf("lockfree quick ratio = %.2f \n", float64(goSize*sizePerGo-int(slower))*100.0/float64(goSize*sizePerGo))
		fmt.Printf("lockfree [<1us][%d] \n", uint64(goSize*sizePerGo)-slower)
		fmt.Printf("lockfree [1-10us][%d] \n", t1_10us)
		fmt.Printf("lockfree [10-100us][%d] \n", t10_100us)
		fmt.Printf("lockfree [100-1000us][%d] \n", t100_1000us)
		fmt.Printf("lockfree [1-10ms][%d] \n", t1_10ms)
		fmt.Printf("lockfree [10-100ms][%d] \n", t10_100ms)
		fmt.Printf("lockfree [>100ms][%d] \n", t100_ms)
	}
}

func chanMain() {
	var (
		t1_10us     = uint64(0) // 1-10微秒
		t10_100us   = uint64(0) // 10-100微秒
		t100_1000us = uint64(0) // 100-1000微秒
		t1_10ms     = uint64(0) // 1-10毫秒
		t10_100ms   = uint64(0) // 10-100毫秒
		t100_ms     = uint64(0) // 大于100毫秒
		slower      = uint64(0)
	)
	c := make(chan uint64, cap)
	// 启动监听协程
	go func() {
		for {
			x, ok := <-c
			if !ok {
				return
			}
			if x%10000000 == 0 {
				fmt.Println("chan [", x, "]")
			}
		}
	}()
	// lockfree计时
	ts := time.Now()
	// 开始写入
	var wg sync.WaitGroup
	wg.Add(goSize)
	for i := 0; i < goSize; i++ {
		go func(start int) {
			if timeAnalyse {
				for j := 0; j < sizePerGo; j++ {
					//写入数据
					tb := time.Now()
					c <- uint64(start*sizePerGo + j + 1)
					tl := time.Since(tb)
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
			} else {
				for j := 0; j < sizePerGo; j++ {
					//写入数据
					c <- uint64(start*sizePerGo + j + 1)
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	fmt.Println("=====channel[", time.Now().Sub(ts), "]=====")
	fmt.Println("----- channel write complete -----")
	time.Sleep(3 * time.Second)
	// 关闭chan
	close(c)
	if timeAnalyse {
		fmt.Printf("channel slow ratio = %.2f \n", float64(slower)*100.0/float64(goSize*sizePerGo))
		fmt.Printf("channel  quick ratio = %.2f \n", float64(goSize*sizePerGo-int(slower))*100.0/float64(goSize*sizePerGo))
		fmt.Printf("channel [<1us][%d] \n", uint64(goSize*sizePerGo)-slower)
		fmt.Printf("channel [1-10us][%d] \n", t1_10us)
		fmt.Printf("channel [10-100us][%d] \n", t10_100us)
		fmt.Printf("channel [100-1000us][%d] \n", t100_1000us)
		fmt.Printf("channel [1-10ms][%d] \n", t1_10ms)
		fmt.Printf("channel [10-100ms][%d] \n", t10_100ms)
		fmt.Printf("channel [>100ms][%d] \n", t100_ms)
	}
}

type longEventHandler struct {
}

func (h *longEventHandler) OnEvent(v interface{}) {
	x := v.(uint64)
	if x%10000000 == 0 {
		fmt.Println("lockfree [", v, "]")
	}
}

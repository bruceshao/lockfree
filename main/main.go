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
	sizePerGo = 10000
	cap       = 1024 * 1024
)

func main() {
	f, _ := os.OpenFile("cpu.pprof", os.O_CREATE|os.O_RDWR, 0644)
	defer f.Close()
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}
	if len(arg) == 0 {
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
	// 创建事件处理器
	eh := &longEventHandler[uint64]{}
	// 创建消费端串行处理的Lockfree
	lf := lockfree.NewLockfree[uint64](cap, eh, lockfree.NewSleepBlockStrategy(time.Millisecond))
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
			for j := 0; j < sizePerGo; j++ {
				//写入数据
				err := producer.Write(uint64(start*sizePerGo + j + 1))
				if err != nil {
					panic(err)
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	fmt.Println("=====lockfree[", time.Now().Sub(ts), "]=====")
	fmt.Println("----- lockfree write complete -----")
	time.Sleep(3 * time.Second)
	// 关闭Lockfree
	lf.Close()
}

func chanMain() {
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
			for j := 0; j < sizePerGo; j++ {
				//写入数据
				c <- uint64(start*sizePerGo + j + 1)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	fmt.Println("=====channel[", time.Now().Sub(ts), "]=====")
	fmt.Println("----- channel write complete -----")
	time.Sleep(3 * time.Second)
	// 关闭Lockfree
	close(c)
}

type longEventHandler[T uint64] struct {
}

func (h *longEventHandler[T]) OnEvent(v uint64) {
	if v%10000000 == 0 {
		fmt.Println("lockfree [", v, "]")
	}
}

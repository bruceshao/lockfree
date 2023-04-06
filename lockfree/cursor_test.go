/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCursor(t *testing.T) {
	c := newCursor()
	ts := time.Now()
	for i := 0; i < 100000000; i++ {
		x := c.increment()
		if x%1000000 == 0 {
			fmt.Println(x)
		}
	}
	tl := time.Since(ts)
	fmt.Printf("time = %v\n", tl)
}

func TestCursor2(t *testing.T) {
	c := newCursor()
	var wg sync.WaitGroup
	wg.Add(10000)
	ts := time.Now()
	for i := 0; i < 10000; i++ {
		go func() {
			for j := 0; j < 10000; j++ {
				x := c.increment()
				if x%10000000 == 0 {
					fmt.Println(x)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Println(time.Since(ts))
}

func TestCursor3(t *testing.T) {
	var c uint64
	var wg sync.WaitGroup
	wg.Add(10000)
	ts := time.Now()
	for i := 0; i < 10000; i++ {
		go func() {
			for j := 0; j < 10000; j++ {
				x := atomic.AddUint64(&c, 1)
				if x%10000000 == 0 {
					fmt.Println(x)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Println(time.Since(ts))
}

type NoPad struct {
	a uint64
	b uint64
	c uint64
}

func (np *NoPad) Increase() {
	atomic.AddUint64(&np.a, 1)
	atomic.AddUint64(&np.b, 1)
	atomic.AddUint64(&np.c, 1)
}

type Pad struct {
	a   uint64
	_p1 [8]uint64
	b   uint64
	_p2 [8]uint64
	c   uint64
	_p3 [8]uint64
}

func (p *Pad) Increase() {
	atomic.AddUint64(&p.a, 1)
	atomic.AddUint64(&p.b, 1)
	atomic.AddUint64(&p.c, 1)
}

func BenchmarkPad_Increase(b *testing.B) {
	pad := &Pad{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			pad.Increase()
		}
	})
}
func BenchmarkNoPad_Increase(b *testing.B) {
	nopad := &NoPad{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			nopad.Increase()
		}
	})
}

/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"runtime"
	"sync/atomic"
)

// sequencer 序号产生器，维护读和写两个状态，写状态具体由内部游标（cursor）维护。
// 读取状态由自身维护，变量read即可
type sequencer struct {
	wc       *cursor
	ws       waitStrategy
	rc       uint64 // 读取游标，因为该值仅会被一个g修改，所以不需要使用cursor
	capacity uint64
}

func newSequencer(capacity int, ws waitStrategy) *sequencer {
	return &sequencer{
		wc:       newCursor(),
		ws:       ws,
		capacity: uint64(capacity),
	}
}

// next 获取下一个可写入的游标，游标可以直接获取，但需要判断该位置是否可以写入，
// 如果可以写入则返回该值，否则则需要等待，直到可以写入为止，判断的标准即rg（Read Goroutine）
// 已读取到的位置+容量大于将要写入的位置，防止出现覆盖的问题
func (s *sequencer) next() uint64 {
	var nextIdx = 0
	// 首先获取下个游标
	next := s.wc.increment()
	// 判断该值是否可以写入
	for {
		if nextIdx > NextWaitMax {
			// 如果多次没有获取到，可以调度出去，等待再次唤醒
			runtime.Gosched()
		}
		if !s.canWrite(next - 1) {
			s.ws.wait()
			nextIdx++
			continue
		}
		// 执行到此时表示可以写入
		return next - 1
	}
}

func (s *sequencer) canWrite(willWrite uint64) bool {
	// next即为打算获取的值
	r := s.nextRead()
	// 确保数据不会被覆盖
	if willWrite < r+s.capacity {
		return true
	}
	return false
}

// nextRead 获取下个要读取的位置
// 使用原子操作解决data race问题
func (s *sequencer) nextRead() uint64 {
	return atomic.LoadUint64(&s.rc)
}

// setRead 更新要读取的位置，只会有一个rg来操作
func (s *sequencer) setRead(expectVal, newVal uint64) bool {
	return atomic.CompareAndSwapUint64(&s.rc, expectVal, newVal)
}

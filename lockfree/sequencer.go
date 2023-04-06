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
	rc       uint64 // 读取游标，因为该值仅会被一个g修改，所以不需要使用cursor
	capacity uint64
}

func newSequencer(capacity int) *sequencer {
	return &sequencer{
		wc:       newCursor(),
		capacity: uint64(capacity),
	}
}

// next 获取下一个可写入的游标，游标可以直接获取，但需要判断该位置是否可以写入，
// 如果可以写入则返回该值，否则则需要等待，直到可以写入为止，判断的标准即rg（Read Goroutine）
// 已读取到的位置+容量大于将要写入的位置，防止出现覆盖的问题
func (s *sequencer) next() uint64 {
	// 首先获取下个游标
	next := s.wc.increment() - 1
	for i := 0; ; i++ {
		// 获取已读的位置
		r := atomic.LoadUint64(&s.rc)
		if next < r+s.capacity {
			// 可以写入数据
			return next
		}
		if i < spin {
			procyield(15)
		} else {
			runtime.Gosched()
		}
	}
}

// nextRead 获取下个要读取的位置
// 使用原子操作解决data race问题
func (s *sequencer) nextRead() uint64 {
	return atomic.LoadUint64(&s.rc)
}

func (s *sequencer) readIncrement() uint64 {
	return atomic.AddUint64(&s.rc, 1)
}

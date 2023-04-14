/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
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
		rc:       1,
		capacity: uint64(capacity),
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

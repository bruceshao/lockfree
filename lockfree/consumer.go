/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"fmt"
	"sync/atomic"
)

// consumer 消费者，这个消费者只会有一个g操作，这样处理的好处是可以不涉及并发操作，其内部不会涉及到任何锁
// 对于实际的并发操作由该g进行分配
type consumer[T any] struct {
	rbuf     *ringBuffer[T]
	abuf     *available
	seqer    *sequencer
	hdl      EventHandler[T]
	parallel bool
	mask     uint64 // 用于使用&代替%（取余）运算提高性能
	status   int32  // 运行状态
}

func newConsumer[T any](parallel bool, rbuf *ringBuffer[T], abuf *available, hdl EventHandler[T]) *consumer[T] {
	return &consumer[T]{
		rbuf:     rbuf,
		abuf:     abuf,
		seqer:    rbuf.sequer,
		hdl:      hdl,
		parallel: parallel,
		mask:     rbuf.capacity - 1,
		status:   READY,
	}
}

func (c *consumer[T]) start() error {
	if atomic.CompareAndSwapInt32(&c.status, READY, RUNNING) {
		go c.handle()
		return nil
	}
	return fmt.Errorf(StartErrorFormat, "Consumer")
}

func (c *consumer[T]) handle() {
	for {
		if c.closed() {
			return
		}
		// 获取需要获取的位置
		readSeq := c.seqer.nextRead()
		pos := int(readSeq & c.mask)
		// 判断是否可以获取到
		var (
			l     = 0
			count = 0
			b     = false
		)
		for {
			// bugfix 防止关闭时后续操作无法释放
			if c.closed() {
				return
			}
			if c.abuf.enabled(pos) {
				// 先获取到值
				v := c.rbuf.element(pos)
				// 设置read自增
				if c.seqer.setRead(readSeq, readSeq+1) {
					// 设置不可用
					c.abuf.disable(pos)
					// 交由协程队列来处理
					if c.parallel {
						go c.hdl.OnEvent(v)
					} else {
						c.hdl.OnEvent(v)
					}
					break
				}
			}
			// 未读取到值
			if l, b = wait(l, ReadWaitMax); b {
				count++
			}
			if count > BlockWaitMax {
				if !c.abuf.wait() {
					// 阻塞未成功的情况下，重置计数器
					// 阻塞成功的话，上一行代码会处理阻塞状态
					count = 0
					l = 0
					continue
				}
			}
		}
	}
}

func (c *consumer[T]) close() error {
	if atomic.CompareAndSwapInt32(&c.status, RUNNING, READY) {
		// 防止阻塞无法释放
		c.abuf.release()
		return nil
	}
	return fmt.Errorf(CloseErrorFormat, "Consumer")
}

func (c *consumer[T]) closed() bool {
	return c.status == READY
}

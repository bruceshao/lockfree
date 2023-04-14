/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"fmt"
	"runtime"
	"sync/atomic"
)

// consumer 消费者，这个消费者只会有一个g操作，这样处理的好处是可以不涉及并发操作，其内部不会涉及到任何锁
// 对于实际的并发操作由该g进行分配
type consumer[T any] struct {
	status int32 // 运行状态
	rbuf   *ringBuffer[T]
	seqer  *sequencer
	blocks blockStrategy
	hdl    EventHandler[T]
}

func newConsumer[T any](rbuf *ringBuffer[T], hdl EventHandler[T], sequer *sequencer, blocks blockStrategy) *consumer[T] {
	return &consumer[T]{
		rbuf:   rbuf,
		seqer:  sequer,
		hdl:    hdl,
		blocks: blocks,
		status: READY,
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
	// 判断是否可以获取到
	rc := c.seqer.nextRead()
	for {
		if c.closed() {
			return
		}
		var i = 0
		for {
			if c.closed() {
				return
			}
			// 看下读取位置的seq是否OK
			if v, exist := c.rbuf.contains(rc - 1); exist {
				rc = c.seqer.readIncrement()
				c.hdl.OnEvent(v)
				i = 0
				break
			}
			if i < spin {
				procyield(30)
			} else if i < spin+passiveSpin {
				runtime.Gosched()
			} else {
				c.blocks.block()
				i = 0
			}
			i++
		}
	}
}

func (c *consumer[T]) close() error {
	if atomic.CompareAndSwapInt32(&c.status, RUNNING, READY) {
		// 防止阻塞无法释放
		c.blocks.release()
		return nil
	}
	return fmt.Errorf(CloseErrorFormat, "Consumer")
}

// closed 判断是否已关闭
// 将直接判断调整为原子操作，解决data race问题
func (c *consumer[T]) closed() bool {
	return atomic.LoadInt32(&c.status) == READY
}

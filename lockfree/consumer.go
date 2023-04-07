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
	sd     stateDescriptor
	seqer  *sequencer
	blocks blockStrategy
	hdl    EventHandler[T]
	mask   uint64 // 用于使用&代替%（取余）运算提高性能
}

func newConsumer[T any](rbuf *ringBuffer[T], sd stateDescriptor, hdl EventHandler[T], blocks blockStrategy) *consumer[T] {
	return &consumer[T]{
		rbuf:   rbuf,
		sd:     sd,
		seqer:  rbuf.sequer,
		hdl:    hdl,
		blocks: blocks,
		mask:   rbuf.capacity - 1,
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
	readSeq := c.seqer.nextRead()
	for {
		if c.closed() {
			return
		}
		// 获取需要获取的位置
		pos := int(readSeq & c.mask)
		var i = 0
		for {
			if c.closed() {
				return
			}
			if c.sd.enabled(pos) {
				// 先获取到值
				v := c.rbuf.element(pos)
				// 设置read自增，并更新返回值
				readSeq = c.seqer.readIncrement()
				// 设置不可用
				c.sd.disable(pos)
				// 交由协程队列来处理
				c.hdl.OnEvent(v)
				i = 0
				break
			}
			if i < spin {
				procyield(30)
			} else if i < spin+passive_spin {
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

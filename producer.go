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

// Producer 生产者
// 核心方法是Write，通过调用Write方法可以将对象写入到队列中
type Producer[T any] struct {
	seqer    *sequencer
	rbuf     *ringBuffer[T]
	blocks   blockStrategy
	capacity uint64
	status   int32
}

func newProducer[T any](seqer *sequencer, rbuf *ringBuffer[T], blocks blockStrategy) *Producer[T] {
	return &Producer[T]{
		seqer:    seqer,
		rbuf:     rbuf,
		blocks:   blocks,
		capacity: rbuf.cap(),
		status:   READY,
	}
}

func (q *Producer[T]) start() error {
	if atomic.CompareAndSwapInt32(&q.status, READY, RUNNING) {
		return nil
	}
	return fmt.Errorf(StartErrorFormat, "Producer")
}

// Write 对象写入核心逻辑
// 首先会从序号产生器中获取一个序号，该序号由atomic自增，不会重复；
// 然后通过&运算获取该序号应该放入的位置pos；
// 通过循环的方式，判断对应pos位置是否可以写入内容，这个判断是通过available数组判断的；
// 如果无法写入则持续循环等待，直到可以写入为止，此处基于一种思想即写入的实时性，写入操作不需要等太久，因此此处是没有阻塞的，
// 仅仅通过调度让出的方式，进行一部分cpu让渡，防止持续占用cpu资源
// 获取到写入资格后将内容写入到ringbuffer，同时更新available数组，并且调用release，以便于释放消费端的阻塞等待
func (q *Producer[T]) Write(v T) error {
	if q.closed() {
		return ClosedError
	}
	next := q.seqer.wc.increment()
	for {
		// 判断是否可以写入
		r := atomic.LoadUint64(&q.seqer.rc) - 1
		if next <= r+q.capacity {
			// 可以写入数据，将数据写入到指定位置
			q.rbuf.write(next-1, v)
			// 释放，防止消费端阻塞
			q.blocks.release()
			return nil
		}
		runtime.Gosched()
                // 防止消费端无限阻塞
                q.blocks.release()
		// 再次判断是否已关闭
		if q.closed() {
			return ClosedError
		}
	}
}

func (q *Producer[T]) close() error {
	if atomic.CompareAndSwapInt32(&q.status, RUNNING, READY) {
		return nil
	}
	return fmt.Errorf(CloseErrorFormat, "Producer")
}

func (q *Producer[T]) closed() bool {
	return atomic.LoadInt32(&q.status) == READY
}

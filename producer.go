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
	"time"
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
		// 再次判断是否已关闭
		if q.closed() {
			return ClosedError
		}
	}
}

// WriteWindow 写入窗口
// 描述当前可写入的状态，如果不能写入则返回零值，如果可以写入则返回写入窗口大小
// 由于执行时不加锁，所以该结果是不可靠的，仅用于在并发环境很高的情况下，进行丢弃行为
func (q *Producer[T]) WriteWindow() int {
	next := q.seqer.wc.atomicLoad() + 1
	r := atomic.LoadUint64(&q.seqer.rc)
	if next < r+q.capacity {
		return int(r+q.capacity - next)
	}
	return - int(next - (r+q.capacity))
}

// WriteTimeout 在写入的基础上设定一个时间，如果时间到了仍然没有写入则会放弃本次写入，返回写入的位置和false
// 使用方需要调用 WriteByCursor 来继续写入该位置，因为位置已经被占用，是必须要写的，不能跳跃性写入
// 在指定时间内写入成功会返回true
// 三个返回项：写入位置、是否写入成功及是否有error
func (q *Producer[T]) WriteTimeout(v T, timeout time.Duration) (uint64, bool, error) {
	if q.closed() {
		return 0, false, ClosedError
	}
	next := q.seqer.wc.increment()
	// 创建定时器
	waiter := time.NewTimer(timeout)
	for {
		select {
		case <-waiter.C:
			// 超时触发，执行到此处表示未写入，返回对应结果即可
			return next, false, nil
		default:
			// 判断是否可以写入
			r := atomic.LoadUint64(&q.seqer.rc) - 1
			if next <= r+q.capacity {
				// 可以写入数据，将数据写入到指定位置
				q.rbuf.write(next-1, v)
				// 释放，防止消费端阻塞
				q.blocks.release()
				// 返回写入成功标识
				return next, true, nil
			}
			runtime.Gosched()
		}
		// 再次判断是否已关闭
		if q.closed() {
			return 0, false, ClosedError
		}
	}
}

// WriteByCursor 根据游标写入内容，wc是调用 WriteTimeout 方法返回false时对应的写入位置，
// 该位置有严格的含义，不要随意填值，否则会造成整个队列异常
// 函数返回值：是否写入成功和是否存在error，若返回false表示写入失败，可以继续调用重复写入
func (q *Producer[T]) WriteByCursor(v T, wc uint64) (bool, error) {
	if q.closed() {
		return false, ClosedError
	}
	// 判断是否可以写入
	r := atomic.LoadUint64(&q.seqer.rc) - 1
	if wc <= r+q.capacity {
		// 可以写入数据，将数据写入到指定位置
		q.rbuf.write(wc-1, v)
		// 释放，防止消费端阻塞
		q.blocks.release()
		// 返回写入成功标识
		return true, nil
	}
	return false, nil
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

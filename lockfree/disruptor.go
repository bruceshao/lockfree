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

// Disruptor 包装类，内部包装了生产者和消费者
type Disruptor[T any] struct {
	writer   *Producer[T]
	consumer *consumer[T]
	status   int32
}

// NewSerialDisruptor 创建串行化消费端的Disruptor
// 串行化消费端会直接调用EventHandler.OnEvent()方法，需要用户侧手动实现并发处理
func NewSerialDisruptor[T any](capacity int, handler EventHandler[T], writeWait waitStrategy) *Disruptor[T] {
	return NewDisruptor(false, capacity, handler, writeWait)
}

// NewParallelDisruptor 创建并行化消费端的Disruptor
// 并行化消费端会通过新启动一个goroutine的方式，调用EventHandler.OnEvent()方法，用户侧不再需要并发处理
func NewParallelDisruptor[T any](capacity int, handler EventHandler[T], writeWait waitStrategy) *Disruptor[T] {
	return NewDisruptor(true, capacity, handler, writeWait)
}

// NewDisruptor 自定义创建消费端的Disruptor
// parallel：表示是否并行化处理
// capacity：buffer的容量大小，类似于chan的大小，但要求必须是2^n，即2的指数倍
// handler：消费端的事件处理器
// writeWait：写入阻塞时等待策略，建议使用SchedWaitStrategy
func NewDisruptor[T any](parallel bool, capacity int, handler EventHandler[T], writeWait waitStrategy) *Disruptor[T] {
	seqer := newSequencer(capacity, writeWait)
	abuf := newAvailable(capacity)
	rbuf := newRingBuffer[T](capacity, seqer)
	cmer := newConsumer[T](parallel, rbuf, abuf, handler)
	writer := newProducer[T](seqer, abuf, rbuf)
	return &Disruptor[T]{
		writer:   writer,
		consumer: cmer,
		status:   READY,
	}
}

func (d *Disruptor[T]) Start() error {
	if atomic.CompareAndSwapInt32(&d.status, READY, RUNNING) {
		// 启动消费者
		if err := d.consumer.start(); err != nil {
			// 恢复现场
			atomic.CompareAndSwapInt32(&d.status, RUNNING, READY)
			return err
		}
		// 启动生产者
		if err := d.writer.start(); err != nil {
			// 恢复现场
			atomic.CompareAndSwapInt32(&d.status, RUNNING, READY)
			return err
		}
		return nil
	}
	return fmt.Errorf(StartErrorFormat, "Disruptor")
}

func (d *Disruptor[T]) Producer() *Producer[T] {
	return d.writer
}

func (d *Disruptor[T]) Running() bool {
	return d.status == RUNNING
}

func (d *Disruptor[T]) Close() error {
	if atomic.CompareAndSwapInt32(&d.status, RUNNING, READY) {
		// 关闭生产者
		if err := d.writer.close(); err != nil {
			// 恢复现场
			atomic.CompareAndSwapInt32(&d.status, READY, RUNNING)
			return err
		}
		// 关闭消费者
		if err := d.consumer.close(); err != nil {
			// 恢复现场
			atomic.CompareAndSwapInt32(&d.status, READY, RUNNING)
			return err
		}
		// 关闭成功
		return nil
	}
	return fmt.Errorf(CloseErrorFormat, "Disruptor")
}

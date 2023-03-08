/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

type e[T any] struct {
	val T
}

// ringBuffer 具体对象的存放区域，通过数组（定长切片）实现环状数据结构
// 其中e为具体对象，非指针，这样可以一次性进行内存申请
type ringBuffer[T any] struct {
	buf      []e[T]
	sequer   *sequencer
	capacity uint64
}

func newRingBuffer[T any](cap int, sequer *sequencer) *ringBuffer[T] {
	x := ringBuffer[T]{
		capacity: uint64(cap),
		buf:      make([]e[T], cap),
		sequer:   sequer,
	}
	return &x
}

func (r *ringBuffer[T]) write(pos int, v T) {
	r.buf[pos].val = v
}

func (r *ringBuffer[T]) element(pos int) T {
	return r.buf[pos].val
}

func (r *ringBuffer[T]) cap() uint64 {
	return r.capacity
}

// available 切片实现的map，通过index（或pos）标识每个位置为0或1
// 当长时间无法读取时会通过blockC进行阻塞，写线程完成时可释放该blockC
// 其内部buf实际是[]uint8，但由于[]uint8切片在寻址时会进行游标是否越界的判断，造成性能下降，
// 因此通过使用unsafe.Pointer直接对对应的值进行操作，从而避免越界判断，提升性能
// 之所以使用uint8是考虑到写并发的行为，防止bit操作导致数据异常（或靠锁解决）
// 由于存在data race问题，此处以调整为[]uint32，便于进行原子操作
//type available struct {
//	buf    availBuffer
//	blockC chan struct{}
//	block  uint32
//}
//
//func newAvailableWithType(capacity int, bufType AvailBufType) *available {
//	if bufType == ArrayBuf {
//		return newAvailable(capacity)
//	}
//	return newAvailableWithBitmap(capacity)
//}
//
//func newAvailable(capacity int) *available {
//	return &available{
//		buf:    NewArrayAvailBuf(capacity),
//		blockC: make(chan struct{}, 0),
//	}
//}
//
//func newAvailableWithBitmap(capacity int) *available {
//	return &available{
//		buf:    NewBitmapAvailBuf(capacity),
//		blockC: make(chan struct{}, 0),
//	}
//}
//
//// enable 设置pos位置为可读状态，读线程可读取
//// 将操作由uint8直接赋值调整为uint32的原子操作，解决data race问题
//func (a *available) enable(pos int) {
//	a.buf.enable(pos)
//}
//
//// enabled 返回pos位置是否可读，true为可读，此时可通过buffer获取对应元素
//// 解决data race
//func (a *available) enabled(pos int) bool {
//	return a.buf.enabled(pos)
//}
//
//// disable 设置pos位置为可写状态，写入线程可写入值
//func (a *available) disable(pos int) {
//	a.buf.disable(pos)
//}
//
//// disabled 返回pos位置是否可写，true为可写，此时写入线程可以写入值至buffer指定位置
//func (a *available) disabled(pos int) bool {
//	return a.buf.disabled(pos)
//}
//
//// wait 消费端由于长时间未获取到结果，阻塞等待
//func (a *available) wait() bool {
//	// 0：未阻塞；1：阻塞
//	if !atomic.CompareAndSwapUint32(&a.block, 0, 1) {
//		// 表示未设置成功
//		return false
//	}
//	// 等待信号
//	<-a.blockC
//	return true
//}
//
//// release 生产者端释放消费端的阻塞状态
//func (a *available) release() {
//	if atomic.CompareAndSwapUint32(&a.block, 1, 0) {
//		// 表示可以释放，即chan是等待状态
//		a.blockC <- struct{}{}
//	}
//	// 无法设置则不用关心
//	return
//}

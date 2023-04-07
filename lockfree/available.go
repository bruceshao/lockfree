/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"sync/atomic"
	"unsafe"
)

// SdrType 状态描述类型
type SdrType int

const (
	Uint32Array = SdrType(0)
	Uint8Array  = SdrType(1)
	Bitmap      = SdrType(2)
)

// stateDescriptor 状态描述符
type stateDescriptor interface {
	// enable 将对应位置设置为可读状态
	enable(pos int)
	// enabled 判断对应位置是否可读
	enabled(pos int) bool
	// disable 将对应位置置为可写
	disable(pos int)
	// disabled 判断对应位置是否可写
	disabled(pos int) bool
}

func NewStateDescriptor(capacity int, sdrType SdrType) stateDescriptor {
	if sdrType == Uint32Array {
		return newAvailableWithUint32(capacity)
	} else if sdrType == Uint8Array {
		return newAvailableWithUint8(capacity)
	} else if sdrType == Bitmap {
		return newAvailableWithBitmap(capacity)
	}
	return nil
}

// availableWithUint32 切片实现的map，通过index（或pos）标识每个位置为0或1
// 当长时间无法读取时会通过blockC进行阻塞，写线程完成时可释放该blockC
// 其内部buf实际是[]uint8，但由于[]uint8切片在寻址时会进行游标是否越界的判断，造成性能下降，
// 因此通过使用unsafe.Pointer直接对对应的值进行操作，从而避免越界判断，提升性能
// 之所以使用uint8是考虑到写并发的行为，防止bit操作导致数据异常（或靠锁解决）
// 由于存在data race问题，此处以调整为[]uint32，便于进行原子操作
type availableWithUint32 struct {
	buf unsafe.Pointer
}

func newAvailableWithUint32(capacity int) *availableWithUint32 {
	p := byteArrayPointerWithUint32(capacity)
	return &availableWithUint32{
		buf: p,
	}
}

// enable 设置pos位置为可读状态，读线程可读取
// 将操作由uint8直接赋值调整为uint32的原子操作，解决data race问题
func (a *availableWithUint32) enable(pos int) {
	atomic.StoreUint32((*uint32)(unsafe.Pointer(uintptr(a.buf)+uintptr(4*pos))), 1)
}

// enabled 返回pos位置是否可读，true为可读，此时可通过buffer获取对应元素
// 解决data race
func (a *availableWithUint32) enabled(pos int) bool {
	return atomic.LoadUint32((*uint32)(unsafe.Pointer(uintptr(a.buf)+uintptr(4*pos)))) == 1
}

// disable 设置pos位置为可写状态，写入线程可写入值
func (a *availableWithUint32) disable(pos int) {
	atomic.StoreUint32((*uint32)(unsafe.Pointer(uintptr(a.buf)+uintptr(4*pos))), 0)
}

// disabled 返回pos位置是否可写，true为可写，此时写入线程可以写入值至buffer指定位置
func (a *availableWithUint32) disabled(pos int) bool {
	return atomic.LoadUint32((*uint32)(unsafe.Pointer(uintptr(a.buf)+uintptr(4*pos)))) == 0
}

// availableWithUint8
// 借用runtime/internal/atomic.Store8和runtime/internal/atomic.Load8，可以实现uint8的原子控制
// 将内存进行了部分降低，相对于uint32而言
type availableWithUint8 struct {
	buf unsafe.Pointer
}

func newAvailableWithUint8(capacity int) *availableWithUint8 {
	p := byteArrayPointerWithUint8(capacity)
	return &availableWithUint8{
		buf: p,
	}
}

// enable 设置pos位置为可读状态，读线程可读取
// 将操作由uint8直接赋值调整为uint32的原子操作，解决data race问题
func (a *availableWithUint8) enable(pos int) {
	Store8((*uint8)(unsafe.Pointer(uintptr(a.buf)+uintptr(pos))), 1)
}

// enabled 返回pos位置是否可读，true为可读，此时可通过buffer获取对应元素
// 解决data race
func (a *availableWithUint8) enabled(pos int) bool {
	return Load8((*uint8)(unsafe.Pointer(uintptr(a.buf)+uintptr(pos)))) == 1
}

// disable 设置pos位置为可写状态，写入线程可写入值
func (a *availableWithUint8) disable(pos int) {
	Store8((*uint8)(unsafe.Pointer(uintptr(a.buf)+uintptr(pos))), 0)
}

// disabled 返回pos位置是否可写，true为可写，此时写入线程可以写入值至buffer指定位置
func (a *availableWithUint8) disabled(pos int) bool {
	return Load8((*uint8)(unsafe.Pointer(uintptr(a.buf)+uintptr(pos)))) == 0
}

type availableWithBitmap struct {
	buf []uint32
}

func newAvailableWithBitmap(capacity int) *availableWithBitmap {
	arrayCap := capacity / 32
	return &availableWithBitmap{
		buf: make([]uint32, arrayCap),
	}
}

// enable 设置pos位置为可读状态，读线程可读取
// 将操作由uint8直接赋值调整为uint32的原子操作，解决data race问题
func (a *availableWithBitmap) enable(pos int) {
	blkAt, bitAt := toAt(pos)
	Or(&a.buf[blkAt], uint32(1<<bitAt))
}

// enabled 返回pos位置是否可读，true为可读，此时可通过buffer获取对应元素
// 解决data race
func (a *availableWithBitmap) enabled(pos int) bool {
	blkAt, bitAt := toAt(pos)
	return Load(&a.buf[blkAt])&(1<<bitAt) > 0
}

// disable 设置pos位置为可写状态，写入线程可写入值
func (a *availableWithBitmap) disable(pos int) {
	blkAt, bitAt := toAt(pos)
	And(&a.buf[blkAt], ^uint32(1<<bitAt))
}

// disabled 返回pos位置是否可写，true为可写，此时写入线程可以写入值至buffer指定位置
func (a *availableWithBitmap) disabled(pos int) bool {
	blkAt, bitAt := toAt(pos)
	return Load(&a.buf[blkAt])&(1<<bitAt) == 0
}

func toAt(pos int) (int, int) {
	return pos >> 5, pos & 31
}

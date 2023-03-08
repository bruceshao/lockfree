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

const (
	ArrayBuf  AvailBufType = 0
	BitmapBuf AvailBufType = 1
)

type AvailBufType int

type availBuffer interface {
	enable(pos int)
	enabled(pos int) bool
	disable(pos int)
	disabled(pos int) bool
	wait() bool
	release()
}

func newAvailBuf(capacity int, bufType AvailBufType) availBuffer {
	if bufType == ArrayBuf {
		return newArrayAvailBuf(capacity)
	}
	return newBitmapAvailBuf(capacity)
}

type BitmapAvailBuf struct {
	bitm   *MultiBitmap
	blockC chan struct{}
	block  uint32
}

func newBitmapAvailBuf(capacity int) *BitmapAvailBuf {
	return &BitmapAvailBuf{
		bitm:   NewMultiBitmapByMax(capacity),
		blockC: make(chan struct{}, 0),
	}
}

func (b *BitmapAvailBuf) enable(pos int) {
	b.bitm.Set(pos)
}

func (b *BitmapAvailBuf) enabled(pos int) bool {
	return b.bitm.Contains(pos)
}

func (b *BitmapAvailBuf) disable(pos int) {
	b.bitm.Remove(pos)
}

func (b *BitmapAvailBuf) disabled(pos int) bool {
	return !b.bitm.Contains(pos)
}

// wait 消费端由于长时间未获取到结果，阻塞等待
func (b *BitmapAvailBuf) wait() bool {
	// 0：未阻塞；1：阻塞
	if !atomic.CompareAndSwapUint32(&b.block, 0, 1) {
		// 表示未设置成功
		return false
	}
	// 等待信号
	<-b.blockC
	return true
}

// release 生产者端释放消费端的阻塞状态
func (b *BitmapAvailBuf) release() {
	if atomic.CompareAndSwapUint32(&b.block, 1, 0) {
		// 表示可以释放，即chan是等待状态
		b.blockC <- struct{}{}
	}
	// 无法设置则不用关心
	return
}

type ArrayAvailBuf struct {
	buf    unsafe.Pointer
	blockC chan struct{}
	block  uint32
}

func newArrayAvailBuf(capacity int) *ArrayAvailBuf {
	return &ArrayAvailBuf{
		buf:    byteArrayPointerWithUint32(capacity),
		blockC: make(chan struct{}, 0),
	}
}

func (b *ArrayAvailBuf) enable(pos int) {
	atomic.StoreUint32((*uint32)(unsafe.Pointer(uintptr(b.buf)+uintptr(pos*4))), 1)
}

func (b *ArrayAvailBuf) enabled(pos int) bool {
	return atomic.LoadUint32((*uint32)(unsafe.Pointer(uintptr(b.buf)+uintptr(pos*4)))) == 1
}

func (b *ArrayAvailBuf) disable(pos int) {
	atomic.StoreUint32((*uint32)(unsafe.Pointer(uintptr(b.buf)+uintptr(pos*4))), 0)
}

func (b *ArrayAvailBuf) disabled(pos int) bool {
	return atomic.LoadUint32((*uint32)(unsafe.Pointer(uintptr(b.buf)+uintptr(pos*4)))) == 0
}

// wait 消费端由于长时间未获取到结果，阻塞等待
func (b *ArrayAvailBuf) wait() bool {
	// 0：未阻塞；1：阻塞
	if !atomic.CompareAndSwapUint32(&b.block, 0, 1) {
		// 表示未设置成功
		return false
	}
	// 等待信号
	<-b.blockC
	return true
}

// release 生产者端释放消费端的阻塞状态
func (b *ArrayAvailBuf) release() {
	if atomic.CompareAndSwapUint32(&b.block, 1, 0) {
		// 表示可以释放，即chan是等待状态
		b.blockC <- struct{}{}
	}
	// 无法设置则不用关心
	return
}

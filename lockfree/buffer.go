/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

type e[T any] struct {
	_   [64]byte
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

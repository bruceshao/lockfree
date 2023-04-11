/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

type e[T any] struct {
	c   uint64
	val T
}

// ringBuffer 具体对象的存放区域，通过数组（定长切片）实现环状数据结构
// 其中e为具体对象，非指针，这样可以一次性进行内存申请
type ringBuffer[T any] struct {
	buf     []e[T]
	sequer  *sequencer
	capMask uint64
}

func newRingBuffer[T any](cap int, sequer *sequencer) *ringBuffer[T] {
	x := ringBuffer[T]{
		capMask: uint64(cap) - 1,
		buf:     make([]e[T], cap),
		sequer:  sequer,
	}
	return &x
}

func (r *ringBuffer[T]) write(c uint64, v T) {
	x := &r.buf[c&r.capMask]
	x.val = v
	x.c = c + 1
}

func (r *ringBuffer[T]) element(c uint64) e[T] {
	return r.buf[c&r.capMask]
}

func (r *ringBuffer[T]) cap() uint64 {
	return r.capMask + 1
}

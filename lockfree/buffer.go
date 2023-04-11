/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import "sync/atomic"

type e struct {
	c   uint64
	val interface{}
}

// ringBuffer 具体对象的存放区域，通过数组（定长切片）实现环状数据结构
// 其中e为具体对象，非指针，这样可以一次性进行内存申请
type ringBuffer struct {
	// 增加默认的对象以便于return，处理data race问题
	tDefault struct{}
	buf      []e
	capMask  uint64
}

func newRingBuffer(cap int) *ringBuffer {
	x := ringBuffer{
		capMask: uint64(cap) - 1,
		buf:     make([]e, cap),
	}
	return &x
}

func (r *ringBuffer) write(c uint64, v interface{}) {
	x := &r.buf[c&r.capMask]
	x.val = v
	atomic.StoreUint64(&x.c, c+1)
}

func (r *ringBuffer) element(c uint64) interface{} {
	return r.buf[c&r.capMask]
}

func (r *ringBuffer) contains(c uint64) (interface{}, bool) {
	x := &r.buf[c&r.capMask]
	if atomic.LoadUint64(&x.c) == c+1 {
		v := x.val
		return v, true
	}
	return r.tDefault, false
}

func (r *ringBuffer) cap() uint64 {
	return r.capMask + 1
}

/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import "sync/atomic"

// cursor 游标，一直持续增长的一个uint64序列
// 该序列用于wg（Write Goroutine）获取对应写入到buffer中元素的位置操作
// 通过使用atomic操作避免锁，提高性能
// 通过使用padding填充的方式，填充前面和后面各使用7个uint64（缓存行填充），避免伪共享问题
type cursor struct {
	p1, p2, p3, p4, p5, p6, p7       uint64
	v                                uint64
	p9, p10, p11, p12, p13, p14, p15 uint64
}

func newCursor() *cursor {
	return &cursor{}
}

func (c *cursor) increment() uint64 {
	return atomic.AddUint64(&c.v, 1)
}

func (c *cursor) atomicLoad() uint64 {
	return atomic.LoadUint64(&c.v)
}

func (c *cursor) load() uint64 {
	return c.v
}

func (c *cursor) store(expectVal, newVal uint64) bool {
	return atomic.CompareAndSwapUint64(&c.v, expectVal, newVal)
}

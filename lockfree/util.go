/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"errors"
	"reflect"
	"runtime"
	"unsafe"
)

const (
	WriteWaitMax     = 7
	NextWaitMax      = 7
	ReadWaitMax      = 7
	BlockWaitMax     = 128
	READY            = 0 // 模块的状态之就绪态
	RUNNING          = 1 // 模块的状态之运行态
	StartErrorFormat = "start model [%s] error"
	CloseErrorFormat = "close model [%s] error"
)

var (
	ClosedError = errors.New("the queue has been closed")
)

// wait 等待，返回是否阻塞
func wait(loop int, waitMax int) (int, bool) {
	loop++
	if loop > waitMax {
		// 减为原来的一半
		loop = loop >> 1
		// 超过次数则阻塞等待调度
		runtime.Gosched()
		return loop, true
	}
	return loop, false
}

// byteArrayPointer 创建uint8切片，返回其对应实际内容（Data）的指针
func byteArrayPointer(capacity int) unsafe.Pointer {
	bytes := make([]uint8, capacity)
	rs := *(*reflect.SliceHeader)(unsafe.Pointer(&bytes))
	return unsafe.Pointer(rs.Data)
}

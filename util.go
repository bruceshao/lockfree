/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"errors"
	"runtime"
	_ "unsafe"
)

const (
	activeSpin       = 4
	passiveSpin      = 2
	READY            = 0 // 模块的状态之就绪态
	RUNNING          = 1 // 模块的状态之运行态
	StartErrorFormat = "start model [%s] error"
	CloseErrorFormat = "close model [%s] error"
)

var (
	ncpu        = runtime.NumCPU()
	spin        = 0
	ClosedError = errors.New("the queue has been closed")
)

func init() {
	if ncpu > 1 {
		spin = activeSpin
	}
}

//go:linkname procyield runtime.procyield
func procyield(cycles uint32)

//go:linkname osyield runtime.osyield
func osyield()

// minSuitableCap 最小的合适的数量
func minSuitableCap(v int) int {
	if v <= 0 {
		return 2
	}
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}

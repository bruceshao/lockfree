/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"runtime"
	"time"
)

// waitStrategy 等待策略
type waitStrategy interface {
	wait()
}

// SchedWaitStrategy 调度等待策略
// 调用runtime.Gosched()方法将当前g让渡出去
type SchedWaitStrategy struct {
}

func (w *SchedWaitStrategy) wait() {
	runtime.Gosched()
}

// SleepWaitStrategy 休眠等待策略
// 调用Sleep方法将当前g让渡出去
type SleepWaitStrategy struct {
	t time.Duration
}

func (w *SleepWaitStrategy) wait() {
	time.Sleep(w.t)
}

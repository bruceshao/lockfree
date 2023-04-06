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

// blockStrategy 阻塞策略
type blockStrategy interface {
	// block 阻塞
	block()

	// release 释放阻塞
	release()
}

// SchedBlockStrategy 调度等待策略
// 调用runtime.Gosched()方法将当前g让渡出去
type SchedBlockStrategy struct {
}

func (w *SchedBlockStrategy) block() {
	runtime.Gosched()
}

func (w *SchedBlockStrategy) release() {
}

// SleepBlockStrategy 休眠等待策略
// 调用Sleep方法将当前g让渡出去
type SleepBlockStrategy struct {
	t time.Duration
}

func NewSleepBlockStrategy(wait time.Duration) *SleepBlockStrategy {
	return &SleepBlockStrategy{
		t: wait,
	}
}

func (w *SleepBlockStrategy) block() {
	time.Sleep(w.t)
}

func (w *SleepBlockStrategy) release() {
}

// ProcYieldBlockStrategy 调度策略
type ProcYieldBlockStrategy struct {
	cycle uint32
}

func NewProcYieldBlockStrategy(cycle uint32) *ProcYieldBlockStrategy {
	return &ProcYieldBlockStrategy{
		cycle: cycle,
	}
}

func (w *ProcYieldBlockStrategy) block() {
	procyield(w.cycle)
}

func (w *ProcYieldBlockStrategy) release() {
}

// OSYieldBlockStrategy 调度策略
type OSYieldBlockStrategy struct {
}

func NewOSYieldWaitStrategy() *OSYieldBlockStrategy {
	return &OSYieldBlockStrategy{}
}

func (w *OSYieldBlockStrategy) block() {
	osyield()
}

func (w *OSYieldBlockStrategy) release() {
}

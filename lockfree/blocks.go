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

func (s *SchedBlockStrategy) block() {
	runtime.Gosched()
}

func (s *SchedBlockStrategy) release() {
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

func (s *SleepBlockStrategy) block() {
	time.Sleep(s.t)
}

func (s *SleepBlockStrategy) release() {
}

// ProcYieldBlockStrategy CPU空指令策略
type ProcYieldBlockStrategy struct {
	cycle uint32
}

func NewProcYieldBlockStrategy(cycle uint32) *ProcYieldBlockStrategy {
	return &ProcYieldBlockStrategy{
		cycle: cycle,
	}
}

func (s *ProcYieldBlockStrategy) block() {
	procyield(s.cycle)
}

func (s *ProcYieldBlockStrategy) release() {
}

// OSYieldBlockStrategy 操作系统调度策略
type OSYieldBlockStrategy struct {
}

func NewOSYieldWaitStrategy() *OSYieldBlockStrategy {
	return &OSYieldBlockStrategy{}
}

func (s *OSYieldBlockStrategy) block() {
	osyield()
}

func (s *OSYieldBlockStrategy) release() {
}

// ChanBlockStrategy chan阻塞策略
type ChanBlockStrategy struct {
}

func NewChanBlockStrategy() *ChanBlockStrategy {
	return &ChanBlockStrategy{}
}

func (s *ChanBlockStrategy) block() {
	osyield()
}

func (s *ChanBlockStrategy) release() {
}

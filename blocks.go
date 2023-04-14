/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// blockStrategy 阻塞策略
type blockStrategy interface {
	// block 阻塞
	block()

	// release 释放阻塞
	release()

	// close 防止资源泄露
	close()
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

func (s *SchedBlockStrategy) close() {
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

func (s *SleepBlockStrategy) close() {
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

func (s *ProcYieldBlockStrategy) close() {
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

func (s *OSYieldBlockStrategy) close() {
}

// ChanBlockStrategy chan阻塞策略
type ChanBlockStrategy struct {
	bc chan struct{}
	b  uint32
}

func NewChanBlockStrategy() *ChanBlockStrategy {
	return &ChanBlockStrategy{
		bc: make(chan struct{}),
	}
}

func (s *ChanBlockStrategy) block() {
	// 0：未阻塞；1：阻塞
	if !atomic.CompareAndSwapUint32(&s.b, 0, 1) {
		return
	}
	// 等待信号
	<-s.bc
}

func (s *ChanBlockStrategy) release() {
	if atomic.CompareAndSwapUint32(&s.b, 1, 0) {
		// 表示可以释放，即chan是等待状态
		s.bc <- struct{}{}
	}
	// 无法设置则不用关心
	return
}

func (s *ChanBlockStrategy) close() {
	close(s.bc)
}

// ConditionBlockStrategy condition 阻塞策略
type ConditionBlockStrategy struct {
	cond *sync.Cond
}

func NewConditionBlockStrategy() *ConditionBlockStrategy {
	return &ConditionBlockStrategy{
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (s *ConditionBlockStrategy) block() {
	s.cond.L.Lock()
	s.cond.Wait()
	s.cond.L.Unlock()
}

func (s *ConditionBlockStrategy) release() {
	s.cond.Broadcast()
}

func (s *ConditionBlockStrategy) close() {
}

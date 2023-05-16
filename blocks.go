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
	block(actual *atomic.Uint64, expected uint64)

	// release 释放阻塞
	release()
}

// SchedBlockStrategy 调度等待策略
// 调用runtime.Gosched()方法使当前 g 主动让出 cpu 资源。
type SchedBlockStrategy struct {
}

func (s *SchedBlockStrategy) block(actual *atomic.Uint64, expected uint64) {
	runtime.Gosched()
}

func (s *SchedBlockStrategy) release() {
}

// SleepBlockStrategy 休眠等待策略
// 调用 Sleep 方法使当前 g 主动让出 cpu 资源。
// sleep poll 参考值:
// 轮询时长为 10us 时，cpu 开销约 2-3% 左右。
// 轮询时长为 5us 时，cpu 开销约在 10% 左右。
// 轮询时长小于 5us 时，cpu 开销接近 100% 满载。
type SleepBlockStrategy struct {
	t time.Duration
}

func NewSleepBlockStrategy(wait time.Duration) *SleepBlockStrategy {
	return &SleepBlockStrategy{
		t: wait,
	}
}

func (s *SleepBlockStrategy) block(actual *atomic.Uint64, expected uint64) {
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

func (s *ProcYieldBlockStrategy) block(actual *atomic.Uint64, expected uint64) {
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

func (s *OSYieldBlockStrategy) block(actual *atomic.Uint64, expected uint64) {
	osyield()
}

func (s *OSYieldBlockStrategy) release() {
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

func (s *ChanBlockStrategy) block(actual *atomic.Uint64, expected uint64) {
	// 0：未阻塞；1：阻塞
	if atomic.CompareAndSwapUint32(&s.b, 0, 1) {
		// 设置成功的话，表示阻塞，需要进行二次判断
		if actual.Load() == expected {
			// 表示阻塞失败，因为结果是一致的，此处需要重新将状态调整回来
			if atomic.CompareAndSwapUint32(&s.b, 1, 0) {
				// 表示回调成功，直接退出即可
				return
			} else {
				// 表示有其他协程release了，则读取对应chan即可
				<-s.bc
			}
		} else {
			// 如果说结果不一致，则表示阻塞，等待被释放即可
			<-s.bc
		}
	}
	// 没有设置成功，不用关注
}

func (s *ChanBlockStrategy) release() {
	if atomic.CompareAndSwapUint32(&s.b, 1, 0) {
		// 表示可以释放，即chan是等待状态
		s.bc <- struct{}{}
	}
	// 无法设置则不用关心
	return
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

func (s *ConditionBlockStrategy) block(actual *atomic.Uint64, expected uint64) {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	if actual.Load() == expected {
		return
	}
	s.cond.Wait()
}

func (s *ConditionBlockStrategy) release() {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	s.cond.Broadcast()
}

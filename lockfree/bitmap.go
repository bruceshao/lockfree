/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import "unsafe"

type MultiBitmap struct {
	buf unsafe.Pointer
}

func NewMultiBitmapByMax(max int) *MultiBitmap {
	arrayCap := max / 32
	return &MultiBitmap{
		buf: byteArrayPointerWithUint32(arrayCap),
	}
}

func (m *MultiBitmap) Set(x int) {
	blkAt := x >> 5 << 2
	bitAt := uint32(1 << (x & 31))
	Or((*uint32)(unsafe.Pointer(uintptr(m.buf)+uintptr(blkAt))), bitAt)
}

func (m *MultiBitmap) Contains(x int) bool {
	blkAt := x >> 5 << 2
	bitAt := uint32(1 << (x & 31))
	return Load((*uint32)(unsafe.Pointer(uintptr(m.buf)+uintptr(blkAt))))&bitAt > 0
}

func (m *MultiBitmap) Remove(x int) {
	blkAt := x >> 5 << 2
	bitAt := ^uint32(1 << (x & 31))
	And((*uint32)(unsafe.Pointer(uintptr(m.buf)+uintptr(blkAt))), bitAt)
}

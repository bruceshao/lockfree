/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import _ "unsafe"

//go:linkname And runtime/internal/atomic.And
func And(ptr *uint32, val uint32)

//go:linkname Or runtime/internal/atomic.Or
func Or(ptr *uint32, val uint32)

//go:linkname Load runtime/internal/atomic.Load
func Load(volatile *uint32) uint32

//go:linkname Store8 runtime/internal/atomic.Store8
func Store8(ptr *uint8, val uint8)

//go:linkname Load8 runtime/internal/atomic.Load8
func Load8(ptr *uint8) uint8

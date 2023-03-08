/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

//go:noescape
func And(ptr *uint32, val uint32)

//go:noescape
func Or(ptr *uint32, val uint32)

//go:noescape
func Load(ptr *uint32) uint32

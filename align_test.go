/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"fmt"
	"testing"
)

func TestBufferAlign32(t *testing.T) {
	buf := newRingBuffer[uint32](1024)
	buf.write(0, 1)
	x := buf.element(0)
	fmt.Println(x)

	buf.write(1023, 2)
	x = buf.element(1023)
	fmt.Println(x)

	buf.write(1024, 3)
	x = buf.element(1024)
	fmt.Println(x)
}

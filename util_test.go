/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMinSuitableCap(t *testing.T) {
	x := minSuitableCap(-1)
	assert.Equal(t, 2, x)
	x = minSuitableCap(3)
	assert.Equal(t, 4, x)
	x = minSuitableCap(10)
	assert.Equal(t, 16, x)
	x = minSuitableCap(1023)
	assert.Equal(t, 1024, x)
	x = minSuitableCap(16)
	assert.Equal(t, 16, x)
}

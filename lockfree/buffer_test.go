/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

import (
	"fmt"
	"reflect"
	"testing"
	"time"
	"unsafe"
)

func TestA(t *testing.T) {
	sli := make([]uint8, 8)
	sh := *(*reflect.SliceHeader)(unsafe.Pointer(&sli))
	mem := unsafe.Pointer(sh.Data)
	Sets(mem)
	fmt.Printf("%v\n", sli[0])
	fmt.Printf("%v\n", sli[1])
	fmt.Printf("%v\n", sli[2])
	fmt.Printf("%v\n", sli[3])
	fmt.Printf("%v\n", sli[4])
	fmt.Printf("%v\n", sli[5])
	fmt.Printf("%v\n", sli[6])
	fmt.Printf("%v\n", sli[7])
}

func Sets(mem unsafe.Pointer) {
	for i := 0; i < 8; i++ {
		*(*uint8)(unsafe.Pointer(uintptr(mem) + uintptr(i))) = 1
	}
}

func TestX(t *testing.T) {
	loop := 1000000
	l := 1024 * 1024
	bytes := make([]byte, l)
	ts := time.Now()
	for i := 0; i < loop; i++ {
		bytes[i] = 0x01
	}
	tl := time.Since(ts)
	fmt.Println(tl.Microseconds())
}

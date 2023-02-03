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
	"time"
)

func TestM(t *testing.T) {
	l := 61
	l = l >> 1
	fmt.Println(l)
}

func TestCh(t *testing.T) {
	c := make(chan struct{}, 0)
	go func() {
		<-c
		fmt.Println(1)
	}()
	c <- struct{}{}
	time.Sleep(time.Second)
}

func TestCh1(t *testing.T) {
	c := make(chan struct{}, 0)
	go func() {
		time.Sleep(time.Second)
		fmt.Printf("wait %v\n", time.Now())
		<-c
		fmt.Printf("read %v\n", time.Now())
	}()
	fmt.Printf("start %v\n", time.Now())
	c <- struct{}{}
	fmt.Printf("write %v\n", time.Now())
	time.Sleep(2 * time.Second)
}

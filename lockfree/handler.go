/*
 * Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 */

package lockfree

// EventHandler 事件处理器接口
// 整个无锁队列中唯一需要用户实现的接口，该接口描述消费端收到消息时该如何处理
// 使用泛型，通过编译阶段确定事件类型，提高性能
type EventHandler interface {
	// OnEvent 用户侧实现，事件处理方法
	OnEvent(v interface{})
}

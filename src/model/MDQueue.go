/**
===========================================================================
 * 双向链表队列
 * Double linked list queue
===========================================================================
*/
package model

import (
	"container/list"
	"reflect"
	"sync"
)

type QueueS struct {
	list   *list.List
	lock   *sync.RWMutex
	maxLen int
}

//* 新建链栈链队列 */
func (queue *QueueS) New(maxLen ...int) *QueueS {
	maxLenInt := int(1 << 30)
	if len(maxLen) > 0 {
		maxLenInt = maxLen[0]
	}
	list := list.New()
	lock := new(sync.RWMutex)
	return &QueueS{list, lock, maxLenInt}
}

//* 清空链表 */
func (queue *QueueS) Renew() {
	defer queue.lock.Unlock()
	queue.lock.Lock()
	queue.list.Init()
}

//* 入队尾(入栈) */
func (queue *QueueS) Push(value interface{}) {
	if value == nil {
		return
	}
	defer queue.lock.Unlock()
	queue.lock.Lock()
	if queue.list.Len() == queue.maxLen {
		queue.list.Remove(queue.list.Front())
	}
	queue.list.PushBack(value)
}

//* 取队尾(出栈) */
func (queue *QueueS) Pop() interface{} {
	defer queue.lock.Unlock()
	queue.lock.Lock()
	e := queue.list.Back()
	if e != nil {
		queue.list.Remove(e)
		return e.Value
	}
	return nil
}

//* 取队尾(不出栈) */
func (queue *QueueS) PopPeek() interface{} {
	defer queue.lock.RUnlock()
	queue.lock.RLock()
	e := queue.list.Back()
	if e != nil {
		return e.Value
	}
	return nil
}

//* 入队首 */
func (queue *QueueS) UnShift(value interface{}) {
	if value == nil {
		return
	}
	defer queue.lock.Unlock()
	queue.lock.Lock()
	if queue.list.Len() == queue.maxLen {
		queue.list.Remove(queue.list.Back())
	}
	queue.list.PushFront(value)
}

//* 出队首(出队) */
func (queue *QueueS) Shift() interface{} {
	defer queue.lock.Unlock()
	queue.lock.Lock()
	e := queue.list.Front()
	if e != nil {
		queue.list.Remove(e)
		return e.Value
	}
	return nil
}

//* 出队首(不出队) */
func (queue *QueueS) ShiftPeek() interface{} {
	defer queue.lock.RUnlock()
	queue.lock.RLock()
	e := queue.list.Front()
	if e != nil {
		return e.Value
	}
	return nil
}

//* 删除链表 */
func (queue *QueueS) Delete(e *list.Element) interface{} {
	if e == nil {
		return nil
	}
	defer queue.lock.Unlock()
	queue.lock.Lock()
	v := queue.list.Remove(e)
	return v
}

//* 设置链表最大长度 */
func (queue *QueueS) GetMaxLen() int {
	return queue.maxLen
}

//* 设置链表最大长度 */
func (queue *QueueS) SetMaxLen(maxLen int) {
	queue.maxLen = maxLen
}

//* 获取链表长度 */
func (queue *QueueS) Len() int {
	return queue.list.Len()
}

//* 判断链表是否为空 */
func (queue *QueueS) IsEmpty() bool {
	return queue.list.Len() == 0
}

//* 查询对象是否存在[存在则返回Element指针] */
func (queue *QueueS) Contains(element interface{}) *list.Element {
	if element == nil {
		return nil
	}
	if queue.IsEmpty() {
		return nil
	}
	defer queue.lock.RUnlock()
	queue.lock.RLock()
	for e := queue.list.Front(); e != nil; e = e.Next() {
		if reflect.DeepEqual(element, e.Value) {
			return e
		}
	}
	return nil
}

//* 获取链表头 */
func (queue *QueueS) Front() *list.Element {
	defer queue.lock.RUnlock()
	queue.lock.RLock()
	return queue.list.Front()
}

//* 获取链表尾 */
func (queue *QueueS) Back() *list.Element {
	defer queue.lock.RUnlock()
	queue.lock.RLock()
	return queue.list.Back()
}

//* 链表转数组指针 */
func (queue *QueueS) ToArray() []*list.Element {
	defer queue.lock.RUnlock()
	queue.lock.RLock()
	resultQueue := make([]*list.Element, 0, queue.list.Len())
	for e := queue.list.Front(); e != nil; e = e.Next() {
		resultQueue = append(resultQueue, e)
	}
	return resultQueue
}

//* 链表转数组对象 */
func (queue *QueueS) ToArrayV() []interface{} {
	defer queue.lock.RUnlock()
	queue.lock.RLock()
	resultQueue := make([]interface{}, 0, queue.list.Len())
	for e := queue.list.Front(); e != nil; e = e.Next() {
		resultQueue = append(resultQueue, e.Value)
	}
	return resultQueue
}

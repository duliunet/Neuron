package model

import (
	"sort"
	"sync"
)

const ShardSize = 32

//* 线程安全的map容器 */
type SyncMapHub struct {
	Tag      string
	mapShard []*syncMapShard
}

//* 线程安全的map切片 */
type syncMapShard struct {
	item map[string]interface{}
	sync.RWMutex
}

//* map元组 */
type syncMapTuple struct {
	Key   string
	Value interface{}
}

//* 初始化 */
func (hub *SyncMapHub) Init(tags ...string) {
	if len(tags) > 0 {
		hub.Tag = tags[0]
	}
	hub.mapShard = make([]*syncMapShard, ShardSize)
	for i := 0; i < ShardSize; i++ {
		hub.mapShard[i] = &syncMapShard{item: make(map[string]interface{})}
	}
}

//* 返回长度 */
func (hub *SyncMapHub) Len() int {
	if hub.mapShard == nil {
		return 0
	}
	count := 0
	for i := 0; i < ShardSize; i++ {
		shard := hub.mapShard[i]
		shard.RLock()
		count += len(shard.item)
		shard.RUnlock()
	}
	return count
}

//* 判断是否为空 */
func (hub *SyncMapHub) IsEmpty() bool {
	return hub.Len() == 0
}

//* 获取元素 */
func (hub *SyncMapHub) Get(k string) interface{} {
	if hub.mapShard == nil {
		return nil
	}
	shard := hub.GetShard(k)
	shard.RLock()
	v, found := shard.item[k]
	shard.RUnlock()
	if found {
		return v
	}
	return nil
}

//* 获取元素并删除 */
func (hub *SyncMapHub) Pop(k string) interface{} {
	if hub.mapShard == nil {
		return nil
	}
	shard := hub.GetShard(k)
	shard.Lock()
	v, found := shard.item[k]
	delete(shard.item, k)
	shard.Unlock()
	if found {
		return v
	}
	return nil
}

//* 设置元素 */
func (hub *SyncMapHub) Set(k string, v interface{}) {
	if hub.mapShard == nil {
		return
	}
	shard := hub.GetShard(k)
	shard.Lock()
	shard.item[k] = v
	shard.Unlock()
}

//* 通过map设置元素 */
func (hub *SyncMapHub) SetByMap(data map[string]interface{}) {
	if hub.mapShard == nil {
		return
	}
	for k, v := range data {
		shard := hub.GetShard(k)
		shard.Lock()
		shard.item[k] = v
		shard.Unlock()
	}
}

//* 删除元素 */
func (hub *SyncMapHub) Del(k string) {
	if hub.mapShard == nil {
		return
	}
	shard := hub.GetShard(k)
	shard.Lock()
	delete(shard.item, k)
	shard.Unlock()
}

//* 利用Channel化数据深拷贝Map */
func (hub *SyncMapHub) DeepCopyMap() <-chan syncMapTuple {
	if hub.mapShard == nil {
		return nil
	}
	ch := make(chan syncMapTuple, hub.Len())
	go func() {
		wg := sync.WaitGroup{}
		wg.Add(ShardSize)
		for _, shard := range hub.mapShard {
			go func(shard *syncMapShard) {
				shard.RLock()
				for key, val := range shard.item {
					ch <- syncMapTuple{key, val}
				}
				shard.RUnlock()
				wg.Done()
			}(shard)
		}
		wg.Wait()
		close(ch)
	}()
	return ch
}

//* 获取所有Key */
func (hub *SyncMapHub) Key2Slice(needSorts ...bool) []string {
	if hub.mapShard == nil {
		return nil
	}
	tmp := make([]string, 0, hub.Len())
	for item := range hub.DeepCopyMap() {
		tmp = append(tmp, item.Key)
	}
	if len(needSorts) > 0 {
		if needSorts[0] {
			sort.Strings(tmp)
		}
	}
	return tmp
}

//* 获取所有Value */
func (hub *SyncMapHub) Val2Slice() []interface{} {
	if hub.mapShard == nil {
		return nil
	}
	tmp := make([]interface{}, 0, hub.Len())
	for item := range hub.DeepCopyMap() {
		tmp = append(tmp, item.Value)
	}
	return tmp
}

//* 转化成map[string]interface{} */
func (hub *SyncMapHub) Convert2Map() map[string]interface{} {
	if hub.mapShard == nil {
		return nil
	}
	tmp := make(map[string]interface{}, hub.Len())
	for item := range hub.DeepCopyMap() {
		tmp[item.Key] = item.Value
	}
	return tmp
}

//* 迭代器 [return true -> 继续运行] [return false -> 停止运行] */
func (hub *SyncMapHub) Iterator(cb func(n int, k string, v interface{}) bool) {
	if hub.mapShard == nil {
		return
	}
	var n int
	for item := range hub.DeepCopyMap() {
		if cb(n, item.Key, item.Value) {
			n++
			continue
		}
		return
	}
}

//* 原子操作[防止指针引用过程中锁无效] */
func (hub *SyncMapHub) AtomProcess(k string, cb func()) {
	if hub.mapShard == nil {
		return
	}
	shard := hub.GetShard(k)
	shard.Lock()
	cb()
	shard.Unlock()
}

//* 根据Key获取分区编号 */
func (hub *SyncMapHub) GetShard(key string) *syncMapShard {
	if hub.mapShard == nil {
		return nil
	}
	return hub.mapShard[uint(fnv32(key))%uint(ShardSize)]
}

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}

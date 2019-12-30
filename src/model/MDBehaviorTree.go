/**
===========================================================================
 * 行为数据结构
 * Behavioral data structure
===========================================================================
*/
package model

//* ================================ STRUCTS ================================ */

type BehaviorForestS struct {
	// 行为树编号
	UUIDQ *QueueS /* UUID */
	// 行为树容器
	Trees SyncMapHub /* map[UUID]*model.BehaviorTreeS */
	// 错误容器
	ErrorQ *QueueS /* string */
}

//* 行为树 */
type BehaviorTreeS struct {
	Tag  string
	Task *QueueS
}

//* 行为任务 */
type BehaviorTaskS struct {
	Tag string
	// 任务开始时间
	Timestamp string
	Action    *QueueS
}

//* 行为动作 */
type BehaviorActionS struct {
	Tag string
	// 任务
	Command string
	// 参数
	Params string
	// 回调
	Callback string
}

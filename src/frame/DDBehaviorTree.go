/**
===========================================================================
 * 行为树 BehaviorTree -> 包含BehaviorTask的队列
 * 行为任务 BehaviorTask -> 包含BehaviorAction的队列
 * 行为动作 BehaviorAction -> 包含具体行为Function队列Tag
 * {
 *     "Tag": "TreeTag",
 *     "Task": [
 *         {
 *             "Tag": "TaskTag",
 *             "Action": [
 *                 {
 *                     "Tag": "Action_1",
 *                     "Command": "",
 *                     "Params": "",
 *                     "Callback": ""
 *                 },
 *                 {
 *                     "Tag": "Action_2",
 *                     "Command": "",
 *                     "Params": "",
 *                     "Callback": ""
 *                 }
 *             ]
 *         }
 *     ]
 * }
===========================================================================
*/
package frame

import (
	"model"
	"reflect"
)

//* ================================ DEFINE ================================ */

type BehaviorTreeS struct {
	tag   string
	brain *BrainS
}

type treeS struct {
	Tag  string
	Task []interface{}
}

//* ================================ INNER INTERFACE ================================ */

//* ================================ PRIVATE ================================ */

func (mBehaviorTree *BehaviorTreeS) main() {

}

//* ================================ PUBLIC ================================ */

//* 构造本体 */
func (mBehaviorTree *BehaviorTreeS) Ontology(neuron *NeuronS) *BehaviorTreeS {
	mBehaviorTree.tag = "BehaviorTree"
	mBehaviorTree.brain = neuron.Brain
	mBehaviorTree.brain.SafeFunction(mBehaviorTree.main)
	return mBehaviorTree
}

//* 新建行为森林 */
func (mBehaviorTree *BehaviorTreeS) NewForest(tags ...string) model.BehaviorForestS {
	tag := "BehaviorForest"
	if len(tags) > 0 {
		tag = tags[0]
	}
	forest := model.BehaviorForestS{}
	forest.UUIDQ = new(model.QueueS).New()
	forest.Trees.Init(tag)
	forest.ErrorQ = new(model.QueueS).New(mBehaviorTree.brain.Const.BehaviorTree.ErrorQLen)
	return forest
}

//* 新建行为树 */
func (mBehaviorTree *BehaviorTreeS) NewTree(tags ...string) *model.BehaviorTreeS {
	tag := "BehaviorTree"
	if len(tags) > 0 {
		tag = tags[0]
	}
	behaviorTree := new(model.BehaviorTreeS)
	behaviorTree.Tag = tag
	behaviorTree.Task = behaviorTree.Task.New()
	return behaviorTree
}

//* 添加行为任务 */
func (mBehaviorTree *BehaviorTreeS) NewTask(tag string) *model.BehaviorTaskS {
	behaviorTask := new(model.BehaviorTaskS)
	behaviorTask.Tag = tag
	behaviorTask.Timestamp = mBehaviorTree.brain.GetDateTime("now").TimestampMill
	behaviorTask.Action = behaviorTask.Action.New()
	return behaviorTask
}

//* 添加行为动作 */
func (mBehaviorTree *BehaviorTreeS) NewAction(tag string, command string, params ...string) *model.BehaviorActionS {
	behaviorAction := new(model.BehaviorActionS)
	behaviorAction.Tag = tag
	behaviorAction.Command = command
	behaviorAction.Params = ""
	if len(params) > 0 {
		behaviorAction.Params = params[0]
	}
	behaviorAction.Callback = ""
	return behaviorAction
}

//* 查询行为任务 */
func (mBehaviorTree *BehaviorTreeS) FindTask(tree *model.BehaviorTreeS, tag string) *model.BehaviorTaskS {
	treeArr := tree.Task.ToArray()
	for _, v := range treeArr {
		vv := v.Value.(*model.BehaviorTaskS)
		if reflect.TypeOf(vv) == reflect.TypeOf((*model.BehaviorTaskS)(nil)) {
			taskTag := vv.Tag
			if taskTag == tag {
				return vv
			}
		}
	}
	return nil
}

//* 删除行为动作 */
func (mBehaviorTree *BehaviorTreeS) FindAction(task *model.BehaviorTaskS, tag string) *model.BehaviorActionS {
	taskArr := task.Action.ToArray()
	for _, v := range taskArr {
		vv := v.Value.(*model.BehaviorActionS)
		if reflect.TypeOf(vv) == reflect.TypeOf((*model.BehaviorActionS)(nil)) {
			actionTag := vv.Tag
			if actionTag == tag {
				return vv
			}
		}
	}
	return nil
}

//* 获取当前行为动作 */
func (mBehaviorTree *BehaviorTreeS) FetchBranch(hub *model.QueueS, needShift ...bool) (*model.BehaviorTreeS, *model.BehaviorTaskS, *model.BehaviorActionS) {
	if hub.IsEmpty() {
		return nil, nil, nil
	}
	// Tree
	behaviorTreeI := hub.ShiftPeek()
	if mBehaviorTree.brain.CheckIsNull(behaviorTreeI) {
		return nil, nil, nil
	}
	behaviorTree := behaviorTreeI.(*model.BehaviorTreeS)
	// Task
	behaviorTaskI := behaviorTree.Task.ShiftPeek()
	if mBehaviorTree.brain.CheckIsNull(behaviorTaskI) {
		hub.Shift()
		return behaviorTree, nil, nil
	}
	behaviorTask := behaviorTaskI.(*model.BehaviorTaskS)
	// Action
	behaviorActionI := behaviorTask.Action.ShiftPeek()
	if len(needShift) > 0 {
		if needShift[0] {
			behaviorTask.Action.Shift()
		}
	}
	if mBehaviorTree.brain.CheckIsNull(behaviorActionI) {
		behaviorTree.Task.Shift()
		return behaviorTree, behaviorTask, nil
	}
	behaviorAction := behaviorActionI.(*model.BehaviorActionS)
	return behaviorTree, behaviorTask, behaviorAction
}

//* Branch转BehaviorTree */
func (mBehaviorTree *BehaviorTreeS) Branch2Tree(tree *model.BehaviorTreeS, task *model.BehaviorTaskS, action *model.BehaviorActionS) *model.BehaviorTreeS {
	treeTemp := mBehaviorTree.NewTree(tree.Tag)
	taskTemp := mBehaviorTree.NewTask(task.Tag)
	taskTemp.Timestamp = task.Timestamp
	taskTemp.Action.Push(action)
	treeTemp.Task.Push(taskTemp)
	return treeTemp
}

//* 序列化行为树 */
func (mBehaviorTree *BehaviorTreeS) Tree2Json(tree *model.BehaviorTreeS) (int, interface{}) {
	var code int
	var data interface{}
	mBehaviorTree.brain.SafeFunction(func() {
		treeS := treeS{
			tree.Tag,
			tree.Task.ToArrayV(),
		}
		for k, v := range treeS.Task {
			task, found := v.(*model.BehaviorTaskS)
			if !found {
				continue
			}
			treeS.Task[k] = struct {
				Tag    string
				Action []interface{}
			}{
				task.Tag,
				task.Action.ToArrayV(),
			}
		}
		code = 100
		data = mBehaviorTree.brain.JsonEncoder(treeS)
	}, func(err interface{}) {
		if err == nil {
			return
		}
		code = 200
		data = err
	})
	return code, data
}

//* 反序列化行为树 */
func (mBehaviorTree *BehaviorTreeS) Json2Tree(json []byte) (int, interface{}) {
	var code int
	var data interface{}
	mBehaviorTree.brain.SafeFunction(func() {
		tree, found := mBehaviorTree.brain.JsonDecoder(json, new(treeS)).(*treeS)
		if !found || mBehaviorTree.brain.CheckIsNull(tree) {
			code = 202
			data = string(json)
			return
		}
		behaviorTree := mBehaviorTree.NewTree(tree.Tag)
		// push task
		for _, v := range tree.Task {
			task := v.(map[string]interface{})
			behaviorTask := mBehaviorTree.NewTask(task["Tag"].(string))
			// push action
			for _, vv := range task["Action"].([]interface{}) {
				taskAction := vv.(map[string]interface{})
				action := mBehaviorTree.NewAction(taskAction["Tag"].(string), taskAction["Command"].(string), taskAction["Params"].(string))
				callback := taskAction["Callback"].(string)
				if callback != "" {
					action.Callback = callback
				}
				behaviorTask.Action.Push(action)
			}
			behaviorTree.Task.Push(behaviorTask)
		}
		code = 100
		data = behaviorTree
	}, func(err interface{}) {
		if err == nil {
			return
		}
		code = 200
		data = err
	})
	return code, data
}

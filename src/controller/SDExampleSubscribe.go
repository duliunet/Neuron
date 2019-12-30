/**
===========================================================================
 * 双子服务示例
 * Example Service Twins
===========================================================================
*/
package controller

import (
	"bytes"
	"fmt"
	"frame"
	"model"
	"modules/websocket"
	"net/http"
)

//* ================================ DEFINE ================================ */
type ExampleSubscribeS struct {
	Const struct {
		tag  string
		root string
		// The other half of the service
		twin string
	}
	Container struct {
		BehaviorTreeQ *model.QueueS
	}
	Connection  struct{}
	StopChannel struct {
		behaviorLooperSC chan bool
	}

	isStarted bool
	neuron    *frame.NeuronS
	mux       *http.ServeMux
}

//* ================================ PRIVATE ================================ */

//* Register service */
func (mExampleSubscribe *ExampleSubscribeS) main() {
	// Const Initialize
	mExampleSubscribe.Const.twin = "/ExamplePublish"
}

//* ================================ INTERFACE ================================ */

//* Interface Example */
func (mExampleSubscribe *ExampleSubscribeS) exampleInterface() {
	brain := mExampleSubscribe.neuron.Brain
	mExampleSubscribe.mux.HandleFunc(mExampleSubscribe.Const.root+"/Example", func(res http.ResponseWriter, req *http.Request) {
		mExampleSubscribe.neuron.Express.ConstructInterface(res, req, mExampleSubscribe.isStarted, func() {
			//* Get/Post FormValue */
			id := req.FormValue("id")
			if brain.CheckIsNull(id) {
				mExampleSubscribe.neuron.Express.CodeResponse(res, 207, "Lack of Parameter -> id", "exampleInterface")
				return
			}
			//* SQL Query */
			mExampleSubscribe.neuron.Mysql.ExecQuery("SELECT *", func(code int, data interface{}) {
				if code != 100 {
					mExampleSubscribe.neuron.Express.CodeResponse(res, code, data, "exampleInterface")
					return
				}
				//* SQL Analyze */
				dataDB := data.(model.SQLDataS).Data[0].([]interface{})
				openId := dataDB[0].(string)
				mExampleSubscribe.Log("openId", openId)
				//* Response */
				mExampleSubscribe.neuron.Express.CodeResponse(res, 100, data.(model.SQLDataS))
			})
		}, func(err interface{}) {
			brain.MessageHandler(mExampleSubscribe.Const.tag, "exampleInterface[ConstructInterface]", 204, err)
			mExampleSubscribe.neuron.Express.CodeResponse(res, 204)
		})
	})
}

//* ================================ PROCESS ================================ */

//* ================================ SQL PROCESS ================================ */

//* ================================ TOOL ================================ */

//* ================================ LOOPER & RECEIVER ================================ */

//* 任务机 -> 回调 */
func (mExampleSubscribe *ExampleSubscribeS) behaviorProcesser(receiverConn *websocket.Conn, messageId string) {
	brain := mExampleSubscribe.neuron.Brain
	// 运行中则避免重复
	if !brain.CheckIsNull(mExampleSubscribe.StopChannel.behaviorLooperSC) || mExampleSubscribe.Container.BehaviorTreeQ.IsEmpty() {
		return
	}
	mExampleSubscribe.StopChannel.behaviorLooperSC = make(chan bool)
	go brain.SetInterval(func() (int, interface{}) {
		if mExampleSubscribe.Container.BehaviorTreeQ.IsEmpty() {
			return 103, "BehaviorTreeQ.IsEmpty"
		}
		// 检索任务队列
		tree, task, action := mExampleSubscribe.neuron.BehaviorTree.FetchBranch(mExampleSubscribe.Container.BehaviorTreeQ, true)
		if action == nil {
			return 100, nil
		}
		var buf bytes.Buffer
		switch action.Command {
		case "RemoteRequest2BTree":
			treeR := mExampleSubscribe.RemoteRequest2BTree(mExampleSubscribe.neuron.BehaviorTree.Branch2Tree(tree, task, action))
			code, data := mExampleSubscribe.neuron.BehaviorTree.Tree2Json(treeR)
			if code != 100 {
				brain.MessageHandler(mExampleSubscribe.Const.tag, "behaviorProcesser[Tree2Json]", code, data)
				return 100, nil
			}
			buf.Write(data.([]byte))
		}
		// 消息广播
		mExampleSubscribe.neuron.Express.ReceiverEval(receiverConn, messageId, mExampleSubscribe.Const.twin, "BehaviorTreeAnalyze", buf.Bytes())
		return 100, nil
	}, func(code int, data interface{}) {
		if code != 100 {
			brain.MessageHandler(mExampleSubscribe.Const.tag, "behaviorProcesser[SetInterval]", code, data)
		}
	}, brain.Const.Interval.HZ25Interval, mExampleSubscribe.StopChannel.behaviorLooperSC)
}

//* 析构扫描任务机 */
func (mExampleSubscribe *ExampleSubscribeS) behaviorLooperKiller() {
	mExampleSubscribe.neuron.Brain.ClearInterval(mExampleSubscribe.StopChannel.behaviorLooperSC)
}

//* ================================ SERVICE ================================ */

//* 构造服务 */
func (mExampleSubscribe *ExampleSubscribeS) service() {
	// 行为森林初始化
	mExampleSubscribe.Container.BehaviorTreeQ = new(model.QueueS).New()
}

//* 析构服务 */
func (mExampleSubscribe *ExampleSubscribeS) serviceKiller() {}

//* ================================ PUBLIC ================================ */

//* 构造本体 */
func (mExampleSubscribe *ExampleSubscribeS) Ontology(neuron *frame.NeuronS, mux *http.ServeMux, root string) *ExampleSubscribeS {
	mExampleSubscribe.neuron = neuron
	mExampleSubscribe.mux = mux
	mExampleSubscribe.Const.tag = root[1:]
	mExampleSubscribe.Const.root = root

	if neuron.Brain.Const.AutorunConfig.SDExampleSubscribe {
		mExampleSubscribe.neuron.Brain.SafeFunction(mExampleSubscribe.main)
		mExampleSubscribe.StartService()
	} else {
		mExampleSubscribe.StopService()
	}
	return mExampleSubscribe
}

//* 返回开关量 */
func (mExampleSubscribe *ExampleSubscribeS) IsStarted() bool {
	return mExampleSubscribe.isStarted
}

//* 启动服务 */
func (mExampleSubscribe *ExampleSubscribeS) StartService() {
	if mExampleSubscribe.isStarted {
		return
	}
	mExampleSubscribe.isStarted = true
	go mExampleSubscribe.neuron.Brain.SafeFunction(mExampleSubscribe.service)
}

//* 停止服务 */
func (mExampleSubscribe *ExampleSubscribeS) StopService() {
	if !mExampleSubscribe.isStarted {
		return
	}
	mExampleSubscribe.isStarted = false
	go mExampleSubscribe.neuron.Brain.SafeFunction(mExampleSubscribe.serviceKiller)
}

//* 打印信息 */
func (mExampleSubscribe *ExampleSubscribeS) Log(title string, content ...interface{}) {
	if title == mExampleSubscribe.Const.tag {
		mExampleSubscribe.neuron.Brain.LogGenerater(model.LogTrace, mExampleSubscribe.Const.tag, title, fmt.Sprintf("%+v", mExampleSubscribe))
	} else {
		mExampleSubscribe.neuron.Brain.LogGenerater(model.LogInfo, mExampleSubscribe.Const.tag, title, content)
	}
}

//* ================================ RPC INTERFACE ================================ */

//* Receiver -> 添加任务 */
func (mExampleSubscribe *ExampleSubscribeS) BehaviorTreePush(receiverConn *websocket.Conn, messageId string, message64 string) {
	brain := mExampleSubscribe.neuron.Brain
	brain.SafeFunction(func() {
		if !mExampleSubscribe.isStarted {
			panic("Interface Banned")
		}
		if brain.CheckIsNull(message64) {
			panic("Params Null")
		}
		var msgbuf bytes.Buffer
		// 解码消息
		msgbuf.Write(brain.Base64Decoder(message64))
		// 解析消息
		code, treeI := mExampleSubscribe.neuron.BehaviorTree.Json2Tree(msgbuf.Bytes())
		if code != 100 {
			brain.MessageHandler(mExampleSubscribe.Const.tag, "behaviorTreePush[Json2Tree]", 209, msgbuf.String())
			return
		}
		tree, found := treeI.(*model.BehaviorTreeS)
		if !found {
			brain.MessageHandler(mExampleSubscribe.Const.tag, "behaviorTreePush[Found]", 220, msgbuf.String())
			return
		}
		// 消息队列
		mExampleSubscribe.Container.BehaviorTreeQ.Push(tree)
		// 启动任务机
		mExampleSubscribe.behaviorProcesser(receiverConn, messageId)
	}, func(err interface{}) {
		if err == nil {
			return
		}
		// 加密回调错误信息
		msg := brain.GenerateMessage("!", "ER", []interface{}{"BehaviorTreePush", err}, messageId)
		if err := mExampleSubscribe.neuron.Express.ReceiverEval(receiverConn, messageId, mExampleSubscribe.Const.twin, "BehaviorTreeError", msg.Bytes()); err != nil {
			brain.MessageHandler(mExampleSubscribe.Const.tag, "BehaviorTreePush[EvalError]", 214, err)
		}
	})
}

//* Receiver -> 执行远程请求 */
func (mExampleSubscribe *ExampleSubscribeS) RemoteRequest(receiverConn *websocket.Conn, messageId string, message64 string) {
	brain := mExampleSubscribe.neuron.Brain
	brain.SafeFunction(func() {
		if !mExampleSubscribe.isStarted {
			panic("Interface Banned")
		}
		if brain.CheckIsNull(message64) {
			panic("Params Null")
		}
		var msgbuf bytes.Buffer
		// 解码消息
		msgbuf.Write(brain.Base64Decoder(message64))
		// 解析消息
		param := *brain.JsonDecoder(msgbuf.Bytes(), new(model.RequestParamS)).(*model.RequestParamS)
		code, data := brain.RequestSync(param)
		msgReply := model.MessageS{
			Code:    code,
			Message: brain.Const.ErrorCode[code],
			Data:    data,
		}
		// 消息广播
		mExampleSubscribe.neuron.Express.ReceiverEval(receiverConn, messageId, mExampleSubscribe.Const.twin, "RemoteRequestAnalyze", brain.JsonEncoder(msgReply))
	})
}

//* Behavior -> 执行远程请求 */
func (mExampleSubscribe *ExampleSubscribeS) RemoteRequest2BTree(tree *model.BehaviorTreeS) *model.BehaviorTreeS {
	brain := mExampleSubscribe.neuron.Brain
	task := tree.Task.ShiftPeek().(*model.BehaviorTaskS)
	action := task.Action.ShiftPeek().(*model.BehaviorActionS)
	// 解析消息
	param := *brain.JsonDecoder([]byte(action.Params), new(model.RequestParamS)).(*model.RequestParamS)
	code, data := brain.RequestSync(param)
	msgReply := model.MessageS{
		Code:    code,
		Message: brain.Const.ErrorCode[code],
		Data:    data,
	}
	action.Callback = string(brain.JsonEncoder(msgReply))
	return tree
}
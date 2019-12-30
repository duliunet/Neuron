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
	"net/url"
)

//* ================================ DEFINE ================================ */
type ExamplePublishS struct {
	Const struct {
		tag  string
		root string
		// The other half of the service
		twin string
	}
	Container struct {
		BehaviorForest model.BehaviorForestS
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
func (mExamplePublish *ExamplePublishS) main() {
	// Const Initialize
	mExamplePublish.Const.twin = "/ExampleSubscribe"
	// Interface
	mExamplePublish.exampleInterface()
}

//* ================================ INTERFACE ================================ */

//* Interface Example */
func (mExamplePublish *ExamplePublishS) exampleInterface() {
	brain := mExamplePublish.neuron.Brain
	mExamplePublish.mux.HandleFunc(mExamplePublish.Const.root+"/Example", func(res http.ResponseWriter, req *http.Request) {
		mExamplePublish.neuron.Express.ConstructInterface(res, req, mExamplePublish.isStarted, func() {
			//* Get/Post FormValue */
			id := req.FormValue("id")
			if brain.CheckIsNull(id) {
				mExamplePublish.neuron.Express.CodeResponse(res, 207, "Lack of Parameter -> id", "exampleInterface")
				return
			}
			//* SQL Query */
			//mExamplePublish.neuron.Mysql.ExecQuery("SELECT *", func(code int, data interface{}) {
			//	if code != 100 {
			//		mExamplePublish.neuron.Express.CodeResponse(res, code, data, "exampleInterface")
			//		return
			//	}
			//	//* SQL Analyze */
			//	dataDB := data.(model.SQLDataS).Data[0].([]interface{})
			//	Id := dataDB[0].(string)
			//	mExamplePublish.Log("Id", Id)
			//})
			mExamplePublish.requestTaskPush(id)
			//* Response */
			mExamplePublish.neuron.Express.CodeResponse(res, 100, "Success")
		}, func(err interface{}) {
			brain.MessageHandler(mExamplePublish.Const.tag, "exampleInterface[ConstructInterface]", 204, err)
			mExamplePublish.neuron.Express.CodeResponse(res, 204)
		})
	})
}

//* ================================ PROCESS ================================ */

//* 构造扫描任务 */
func (mExamplePublish *ExamplePublishS) requestTaskPush(id string) {
	// 树初始化
	behaviorTree := mExamplePublish.neuron.BehaviorTree.NewTree(mExamplePublish.Const.tag)
	// 任务初始化
	ScanTask := mExamplePublish.neuron.BehaviorTree.NewTask("RequestTask")
	// 动作初始化
	host, path := mExamplePublish.neuron.Express.Url2HostPath("https://github.com/")
	header := mExamplePublish.neuron.Brain.Const.HTTPRequest.DefaultHeader
	header["Cookie"] = []string{}
	header["Content-Type"] = []string{"application/x-www-form-urlencoded"}
	postData := url.Values{}
	postData.Add("Id", id)
	query, _ := url.QueryUnescape(postData.Encode())
	reqParam := model.RequestParamS{
		Host:   host,
		Path:   fmt.Sprintf("%v?%v", path, query),
		Header: header,
	}
	ScanTask.Action.Push(mExamplePublish.neuron.BehaviorTree.NewAction("ActionTag", "RemoteRequest2BTree", string(mExamplePublish.neuron.Brain.JsonEncoder(reqParam))))
	behaviorTree.Task.Push(ScanTask)
	// 封印任务树
	uuid := mExamplePublish.neuron.Brain.UUID()
	mExamplePublish.Container.BehaviorForest.UUIDQ.Push(uuid)
	mExamplePublish.Container.BehaviorForest.Trees.Set(uuid, behaviorTree)
}

//* ================================ SQL PROCESS ================================ */

//* ================================ TOOL ================================ */

//* ================================ LOOPER & RECEIVER ================================ */

//* 任务机 -> 扫描 */
func (mExamplePublish *ExamplePublishS) behaviorLooper() {
	brain := mExamplePublish.neuron.Brain
	// 运行中则避免重复
	if !brain.CheckIsNull(mExamplePublish.StopChannel.behaviorLooperSC) {
		return
	}
	// 获取CommanderHub
	hub := brain.Container.CommanderHub
	// 设立索引缓存
	selectIndex := 0
	// 开始轮循
	mExamplePublish.StopChannel.behaviorLooperSC = make(chan bool)
	go brain.SetInterval(func() (int, interface{}) {
		// 选择性广播
		selectIndex = brain.SelectIterator(hub.Len(), selectIndex)
		// 获取所有客户端Key
		clientKeys := hub.Key2Slice(true)
		for k, v := range clientKeys {
			if selectIndex != k {
				continue
			}
			neuronId := hub.Get(v).(model.SocketClient).Tag
			// 抛出UUID
			uuid, found := mExamplePublish.Container.BehaviorForest.UUIDQ.Shift().(string)
			if !found {
				continue
			}
			// 抛出UUID对应BTree
			tree, found := mExamplePublish.Container.BehaviorForest.Trees.Pop(uuid).(*model.BehaviorTreeS)
			if !found {
				continue
			}
			// BTree转Json
			code, data := mExamplePublish.neuron.BehaviorTree.Tree2Json(tree)
			if code != 100 {
				continue
			}
			mExamplePublish.neuron.Express.CommanderEval(neuronId, mExamplePublish.Const.twin, "BehaviorTreePush", data.([]byte))
		}
		return 100, nil
	}, func(code int, data interface{}) {
		if code != 100 {
			brain.MessageHandler(mExamplePublish.Const.tag, "behaviorProcesser[SetInterval]", code, data)
		}
	}, brain.Const.Interval.HZ8Interval, mExamplePublish.StopChannel.behaviorLooperSC, true)
}

//* 析构扫描任务机 */
func (mExamplePublish *ExamplePublishS) behaviorLooperKiller() {
	mExamplePublish.neuron.Brain.ClearInterval(mExamplePublish.StopChannel.behaviorLooperSC)
}

//* ================================ SERVICE ================================ */

//* 构造服务 */
func (mExamplePublish *ExamplePublishS) service() {
	// 行为森林初始化
	mExamplePublish.Container.BehaviorForest = mExamplePublish.neuron.BehaviorTree.NewForest()
	// 初始化任务机
	mExamplePublish.behaviorLooper()
}

//* 析构服务 */
func (mExamplePublish *ExamplePublishS) serviceKiller() {
	mExamplePublish.behaviorLooperKiller()
}

//* ================================ PUBLIC ================================ */

//* 构造本体 */
func (mExamplePublish *ExamplePublishS) Ontology(neuron *frame.NeuronS, mux *http.ServeMux, root string) *ExamplePublishS {
	mExamplePublish.neuron = neuron
	mExamplePublish.mux = mux
	mExamplePublish.Const.tag = root[1:]
	mExamplePublish.Const.root = root

	if neuron.Brain.Const.AutorunConfig.SDExamplePublish {
		mExamplePublish.neuron.Brain.SafeFunction(mExamplePublish.main)
		mExamplePublish.StartService()
	} else {
		mExamplePublish.StopService()
	}
	return mExamplePublish
}

//* 返回开关量 */
func (mExamplePublish *ExamplePublishS) IsStarted() bool {
	return mExamplePublish.isStarted
}

//* 启动服务 */
func (mExamplePublish *ExamplePublishS) StartService() {
	if mExamplePublish.isStarted {
		return
	}
	mExamplePublish.isStarted = true
	go mExamplePublish.neuron.Brain.SafeFunction(mExamplePublish.service)
}

//* 停止服务 */
func (mExamplePublish *ExamplePublishS) StopService() {
	if !mExamplePublish.isStarted {
		return
	}
	mExamplePublish.isStarted = false
	go mExamplePublish.neuron.Brain.SafeFunction(mExamplePublish.serviceKiller)
}

//* 打印信息 */
func (mExamplePublish *ExamplePublishS) Log(title string, content ...interface{}) {
	if title == mExamplePublish.Const.tag {
		mExamplePublish.neuron.Brain.LogGenerater(model.LogTrace, mExamplePublish.Const.tag, title, fmt.Sprintf("%+v", mExamplePublish))
	} else {
		mExamplePublish.neuron.Brain.LogGenerater(model.LogInfo, mExamplePublish.Const.tag, title, content)
	}
}

//* ================================ RPC INTERFACE ================================ */

//* Commander -> 分析任务结果 */
func (mExamplePublish *ExamplePublishS) BehaviorTreeAnalyze(commanderConn *websocket.Conn, messageId string, message64 string) {
	brain := mExamplePublish.neuron.Brain
	brain.SafeFunction(func() {
		if !mExamplePublish.isStarted {
			panic(fmt.Sprintf("[%s]Interface Banned", messageId))
		}
		if brain.CheckIsNull(message64) {
			panic(fmt.Sprintf("[%s]Params Null", messageId))
		}
		// 解码消息
		var msgbuf bytes.Buffer
		msgbuf.Write(brain.Base64Decoder(message64))
		// 解析消息
		code, data := mExamplePublish.neuron.BehaviorTree.Json2Tree(msgbuf.Bytes())
		if code != 100 {
			brain.MessageHandler(mExamplePublish.Const.tag, "BehaviorTreeAnalyze[Json2Tree]", 204, data)
			return
		}
		// 分析内容
		tree, found := data.(*model.BehaviorTreeS)
		if !found {
			brain.MessageHandler(mExamplePublish.Const.tag, "BehaviorTreeAnalyze[*model.BehaviorTreeS]", 221, data)
			return
		}
		// 不是本模块指令则抛弃
		if tree.Tag != mExamplePublish.Const.tag {
			brain.MessageHandler(mExamplePublish.Const.tag, "BehaviorTreeAnalyze[*model.BehaviorTreeS]", 221, "tree.Tag != mExamplePublish.Const.tag")
			return
		}
		task := tree.Task.ShiftPeek().(*model.BehaviorTaskS)
		if !found {
			brain.MessageHandler(mExamplePublish.Const.tag, "BehaviorTreeAnalyze[*model.BehaviorTaskS]", 221, data)
			return
		}
		action := task.Action.ShiftPeek().(*model.BehaviorActionS)
		if !found {
			brain.MessageHandler(mExamplePublish.Const.tag, "BehaviorTreeAnalyze[*model.BehaviorActionS]", 221, data)
			return
		}
		// 分类讨论
		switch task.Tag {
		case "RequestTask":
			switch action.Command {
			// 解析远程爬虫数据
			case "RemoteRequest2BTree":
				// 解析 -> MessageS
				msg, found := brain.JsonDecoder([]byte(action.Callback), new(model.MessageS)).(*model.MessageS)
				if !found {
					brain.MessageHandler(mExamplePublish.Const.tag, "BehaviorTreeAnalyze[RemoteRequest2BTree(model.MessageS)]", 221, data)
					return
				}
				if msg.Code != 100 {
					brain.MessageHandler(mExamplePublish.Const.tag, "BehaviorTreeAnalyze[RemoteRequest2BTree(model.MessageS)]", msg.Code, msg.Data)
					return
				}
				// 解析 -> ResponseDataS
				res, found := brain.Map2Struct(msg.Data, new(model.ResponseDataS)).(*model.ResponseDataS)
				if !found {
					brain.MessageHandler(mExamplePublish.Const.tag, "BehaviorTreeAnalyze[RemoteRequest2BTree(model.ResponseDataS)]", 221, data)
					return
				}
				// 解析 -> Data
				mExamplePublish.Log("Header", res.Header)
				mExamplePublish.Log("Body", string(res.Body))
				mExamplePublish.Log("URLProxy", res.URLProxy)
			}
		}
	})
}

//* Commander -> 收集错误信息 */
func (mExamplePublish *ExamplePublishS) BehaviorTreeError(commanderConn *websocket.Conn, messageId string, message64 string) {
	brain := mExamplePublish.neuron.Brain
	brain.SafeFunction(func() {
		if !mExamplePublish.isStarted {
			panic(fmt.Sprintf("[%s]Interface Banned", messageId))
		}
		if brain.CheckIsNull(message64) {
			panic(fmt.Sprintf("[%s]Params Null", messageId))
		}
		var msgbuf bytes.Buffer
		// 解码消息
		msgbuf.Write(brain.Base64Decoder(message64))
		// 日志记录
		brain.LogGenerater(model.LogError, mExamplePublish.Const.tag, "BehaviorTreeError[message]", msgbuf.String())
		// 缓存错误
		mExamplePublish.Container.BehaviorForest.ErrorQ.Push(msgbuf.String())
	})
}

//* Commander -> 分析远程请求 */
func (mExamplePublish *ExamplePublishS) RemoteRequestAnalyze(commanderConn *websocket.Conn, messageId string, message64 string) {
	brain := mExamplePublish.neuron.Brain
	brain.SafeFunction(func() {
		if !mExamplePublish.isStarted {
			panic("Interface Banned")
		}
		if brain.CheckIsNull(message64) {
			panic("Params Null")
		}
		// 解析消息
		msgReply := *brain.JsonDecoder([]byte(message64), new(model.MessageS)).(*model.MessageS)
		// 日志记录
		fmt.Printf("msgReply -> %s\n", brain.Base64Decoder(msgReply.Data.(string)))
	})
}

//* Behavior -> 分析远程请求 */
func (mExamplePublish *ExamplePublishS) RemoteRequest2BTreeAnalyze(tree *model.BehaviorTreeS) *model.BehaviorTreeS {
	brain := mExamplePublish.neuron.Brain
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
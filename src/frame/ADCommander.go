/**
===========================================================================
 * 分布式运算队列主节点
 * Distributed computing queue master node
===========================================================================
*/

package frame

import (
	"fmt"
	"model"
	"modules/trigger"
	"modules/websocket"
	"net/http"
	"strings"
)

//* ================================ DEFINE ================================ */
type CommanderS struct {
	Const struct {
		tag  string
		root string
	}
	Container   struct{}
	Connection  struct{}
	StopChannel struct {
		CommanderLooperSC chan bool
	}

	isStarted bool
	neuron    *NeuronS
	mux       *http.ServeMux
}

//* ================================ INNER INTERFACE ================================ */

func (mCommander *CommanderS) WSHub() model.SyncMapHub {
	return mCommander.neuron.Brain.Container.CommanderHub
}

func (mCommander *CommanderS) GMessageHandler(clientI interface{}, msgI interface{}) {
	ws := clientI.(model.SocketClient).Conn.(*websocket.Conn)
	msg := msgI.([]byte)
	// 解密
	msgData := mCommander.neuron.Brain.SystemDecrypt(msg)
	if mCommander.neuron.Brain.CheckIsNull(msgData) {
		mCommander.neuron.Brain.MessageHandler(mCommander.Const.tag, mCommander.neuron.Brain.Container.CommanderHub.Tag+" -> SystemDecrypt", 203, "[Visitor -> "+ws.Request().RemoteAddr+"]")
	} else {
		// 解码
		GMessageArr := mCommander.neuron.Brain.AnalyzeMessage(string(msgData))
		if mCommander.neuron.Brain.CheckIsNull(GMessageArr) {
			mCommander.neuron.Brain.MessageHandler(mCommander.Const.tag, mCommander.neuron.Brain.Container.CommanderHub.Tag+" -> AnalyzeMessage", 203, "[Visitor -> "+ws.Request().RemoteAddr+"]")
		} else {
			// 处理逻辑
			mCommander.gMessageHandler(ws, GMessageArr)
		}
	}
}

func (mCommander *CommanderS) gMessageHandler(ws *websocket.Conn, GMessageArr []*model.GMessageS) {
	// Load Client from Hub
	clientI := mCommander.neuron.Brain.Container.CommanderHub.Get(ws.Request().RemoteAddr)
	client, found := clientI.(model.SocketClient)
	if !found {
		return
	}
	// 成功
	for _, v := range GMessageArr {
		if v != nil {
			if mCommander.neuron.Brain.Const.CommanderLog {
				mCommander.neuron.Brain.MessageHandler(mCommander.Const.tag, fmt.Sprintf("%v -> GMessage[%v]", mCommander.neuron.Brain.Container.CommanderHub.Tag, ws.Request().RemoteAddr), 100, []interface{}{v.ID, v.Head, v.Tag, v.Cmds})
			}
			switch v.Head {
			case "!":
				switch v.Tag {
				case "HEART":
					// 赋予tag信息为Const.NeuronId
					client.Tag = v.ID
					for _, vv := range v.Cmds {
						// 用于其他模块获取心跳信息后更新数据
						mCommander.Log(fmt.Sprintf("HEART -> [%v]", v.ID), mCommander.neuron.Brain.Base64Decoder(vv.(string)))
						mCommander.neuron.Brain.Container.CommanderReply.Push(model.CommanderPiece{NeuronId: client.Tag, GMessage: *v})
					}
				case "REPLY":
					for _, vv := range v.Cmds {
						mCommander.Log(fmt.Sprintf("REPLY -> [%v]", v.ID), mCommander.neuron.Brain.Base64Decoder(vv.(string)))
						mCommander.neuron.Brain.Container.CommanderReply.Push(model.CommanderPiece{NeuronId: client.Tag, GMessage: *v})
					}
				}
			case "?":
				if v.Tag == "EVAL" {
					var args []interface{}
					args = append(args, client.Conn)
					args = append(args, v.ID)
					for k, v := range v.Cmds {
						if k != 0 && k != 1 {
							args = append(args, v)
						}
					}
					trigger.FireBackground("EVAL", v.Cmds[0].(string), v.Cmds[1].(string), args)
				}
			case "~":
			}
		}
	}
	// Save wsClient
	mCommander.neuron.Brain.Container.CommanderHub.Set(ws.Request().RemoteAddr, client)
}

//* ================================ PRIVATE ================================ */

//* 注册服务 */
func (mCommander *CommanderS) main() {
	/* 初始化通信协议 */
	mCommander.commandChannelInit()
	mCommander.commandMessageInterface()
}

//* ================================ INTERFACE ================================ */

//* 指令发送接口 */
func (mCommander *CommanderS) commandMessageInterface() {
	// Interface Init
	mCommander.mux.HandleFunc(mCommander.Const.root+"/Message", func(res http.ResponseWriter, req *http.Request) {
		mCommander.neuron.Express.ConstructInterface(res, req, mCommander.isStarted, func() {
			query := mCommander.neuron.Express.Req2Query(req)
			if mCommander.neuron.Brain.CheckIsNull(query) {
				mCommander.neuron.Express.CodeResponse(res, 207, "Lack of Parameter", "commandMessageInterface")
				return
			}
			neuronId := query["neuronId"][0]
			if mCommander.neuron.Brain.CheckIsNull(neuronId) {
				mCommander.neuron.Express.CodeResponse(res, 207, "Lack of Parameter -> neuronId", "commandMessageInterface")
				return
			}
			message := query["message"][0]
			if mCommander.neuron.Brain.CheckIsNull(message) {
				mCommander.neuron.Express.CodeResponse(res, 207, "Lack of Parameter -> message", "commandMessageInterface")
				return
			}

			gMsg := mCommander.neuron.Brain.AnalyzeMessage(message)
			mCommander.neuron.Brain.Container.CommanderQueue.Push(model.CommanderPiece{NeuronId: neuronId, GMessage: *gMsg[0]})
			mCommander.neuron.Express.CodeResponse(res, 100)
		})
	})
}

//* ================================ PROCESS ================================ */

//* 初始化指令频道 */
func (mCommander *CommanderS) commandChannelInit() {
	// Container Init
	mCommander.neuron.Brain.Container.CommanderHub.Init("CommanderChannel")
	// Queue Init
	mCommander.neuron.Brain.Container.CommanderQueue = new(model.QueueS).New()
	// Reply Init
	mCommander.neuron.Brain.Container.CommanderReply = new(model.QueueS).New(1 << 20)
	// Interface Init
	mCommander.mux.HandleFunc(mCommander.Const.root+"/Channel", func(res http.ResponseWriter, req *http.Request) {
		mCommander.neuron.Express.ConstructInterface(res, req, mCommander.isStarted, func() {
			switch req.Header.Get("Connection") {
			case "Upgrade":
				websocket.Handler(mCommander.neuron.Express.WSHandler).ServeHTTP(res, req, mCommander)
				break
			default:
				mCommander.neuron.Express.ErrorResponse(res, 500)
				break
			}
		})
	})
}

//* 循环任务 */
func (mCommander *CommanderS) commanderLooper() {
	mCommander.StopChannel.CommanderLooperSC = make(chan bool)
	go mCommander.neuron.Brain.SetInterval(func() (int, interface{}) {
		if !mCommander.isStarted {
			return 103, "commanderLooper -> Shutdown"
		}
		// Unshift CommanderQueue
		if !mCommander.neuron.Brain.Container.CommanderQueue.IsEmpty() {
			CommanderPiece := mCommander.neuron.Brain.Container.CommanderQueue.Shift()
			mCommander.sendCommand(CommanderPiece)
		}
		return 100, nil
	}, func(code int, data interface{}) {
		if code != 100 {
			mCommander.neuron.Brain.MessageHandler(mCommander.Const.tag, "commanderLooper -> Error", code, data)
		}
	}, mCommander.neuron.Brain.Const.Interval.CommanderInterval, mCommander.StopChannel.CommanderLooperSC)
}

func (mCommander *CommanderS) commanderLooperKiller() {
	mCommander.neuron.Brain.ClearInterval(mCommander.StopChannel.CommanderLooperSC)
}

//* ================================ TOOL ================================ */

//* 发送指令 */
func (mCommander *CommanderS) sendCommand(pieceI interface{}) {
	if mCommander.neuron.Brain.CheckIsNull(pieceI) {
		return
	}
	piece := pieceI.(model.CommanderPiece)
	mCommander.neuron.Express.WSBroadcast(func(rank int, ip string, neuronId string, conn *websocket.Conn) {
		// tag为空则广播
		if piece.NeuronId == neuronId || strings.TrimSpace(piece.NeuronId) == "" {
			gMsg := mCommander.neuron.Brain.GenerateMessage(piece.GMessage.Head, piece.GMessage.Tag, piece.GMessage.Cmds, piece.GMessage.ID)
			_, err := conn.Write(mCommander.neuron.Brain.SystemEncrypt(gMsg.Bytes()))
			if err != nil {
				// 发送则记录日志
				if mCommander.neuron.Brain.Const.CommanderLog {
					mCommander.Log("Broadcast2Neuron", fmt.Sprintf("[%s] -> %s", ip, gMsg.String()))
				}
			}
		}
	}, mCommander.WSHub())
}

//* ================================ SERVICE ================================ */

//* 构造服务 */
func (mCommander *CommanderS) service() {
	/* 命令发布 */
	mCommander.commanderLooper()
}

//* 析构服务 */
func (mCommander *CommanderS) serviceKiller() {
	// 停止CommanderLooper
	mCommander.commanderLooperKiller()
	if !mCommander.neuron.Brain.Container.CommanderHub.IsEmpty() {
		// 清空WSHub
		mCommander.neuron.Brain.Container.CommanderHub.Iterator(func(n int, k string, v interface{}) bool {
			v.(model.SocketClient).Conn.(*websocket.Conn).Close()
			mCommander.neuron.Brain.Container.CommanderHub.Del(k)
			return true
		})
	}
}

//* ================================ PUBLIC ================================ */

//* 构造本体 */
func (mCommander *CommanderS) Ontology(neuron *NeuronS, mux *http.ServeMux, root string) *CommanderS {
	mCommander.neuron = neuron
	mCommander.mux = mux
	mCommander.Const.tag = root[1:]
	mCommander.Const.root = root

	if neuron.Brain.Const.AutorunConfig.ADCommander {
		mCommander.neuron.Brain.SafeFunction(mCommander.main)
		mCommander.StartService()
	} else {
		mCommander.StopService()
	}
	return mCommander
}

//* 返回开关量 */
func (mCommander *CommanderS) IsStarted() bool {
	return mCommander.isStarted
}

//* 启动服务 */
func (mCommander *CommanderS) StartService() {
	if mCommander.isStarted {
		return
	}
	mCommander.isStarted = true
	go mCommander.neuron.Brain.SafeFunction(mCommander.service)
}

//* 停止服务 */
func (mCommander *CommanderS) StopService() {
	if !mCommander.isStarted {
		return
	}
	mCommander.isStarted = false
	go mCommander.neuron.Brain.SafeFunction(mCommander.serviceKiller)
}

//* 打印信息 */
func (mCommander *CommanderS) Log(title string, content ...interface{}) {
	if title == mCommander.Const.tag {
		mCommander.neuron.Brain.LogGenerater(model.LogTrace, mCommander.Const.tag, title, fmt.Sprintf("%+v", mCommander))
	} else {
		mCommander.neuron.Brain.LogGenerater(model.LogInfo, mCommander.Const.tag, title, content)
	}
}

/**
===========================================================================
 * 分布式运算队列分节点
 * Distributed computing queue sub-node
===========================================================================
*/

package frame

import (
	"bytes"
	"fmt"
	"model"
	"modules/trigger"
	"modules/websocket"
	"net/http"
)

//* ================================ DEFINE ================================ */
type ReceiverS struct {
	Const struct {
		tag  string
		root string
	}
	Container  struct{}
	Connection struct {
		receiverConn *websocket.Conn
	}
	StopChannel struct {
		receiverLooperSC chan bool
	}
	isStarted bool
	neuron    *NeuronS
	mux       *http.ServeMux
}

//* ================================ PRIVATE ================================ */

//* 注册服务 */
func (mReceiver *ReceiverS) main() {
	mReceiver.receiverMessageInterface()
}

//* ================================ INTERFACE ================================ */

//* 指令发送接口 */
func (mReceiver *ReceiverS) receiverMessageInterface() {
	// Interface Init
	mReceiver.mux.HandleFunc(mReceiver.Const.root+"/Message", func(res http.ResponseWriter, req *http.Request) {
		mReceiver.neuron.Express.ConstructInterface(res, req, mReceiver.isStarted, func() {
			query := mReceiver.neuron.Express.Req2Query(req)
			if mReceiver.neuron.Brain.CheckIsNull(query["message"]) {
				mReceiver.neuron.Express.CodeResponse(res, 207, "Lack of Parameter -> message", "receiverMessageInterface")
				return
			}
			message := query["message"][0]
			if !mReceiver.neuron.Brain.CheckIsNull(mReceiver.Connection.receiverConn) {
				mReceiver.neuron.Brain.Retry(3, func() (int, interface{}) {
					if _, err := mReceiver.Connection.receiverConn.Write([]byte(message)); err != nil {
						return 214, err
					}
					return 100, nil
				}, func(code int, data interface{}) {
					if code != 100 {
						mReceiver.neuron.Brain.MessageHandler(mReceiver.Const.tag, "receiverMessageInterface[Retry]", code, data)
						return
					}
				})
				mReceiver.neuron.Express.CodeResponse(res, 100)
			} else {
				mReceiver.neuron.Express.CodeResponse(res, 214, "Lack of Connection", "receiverMessageInterface")
			}
		})
	})
}

//* ================================ PROCESS ================================ */

//* 初始化指令频道 */
func (mReceiver *ReceiverS) receiverInit() {
	mTrigger := trigger.New()
	mTrigger.On("Open", func(code int, data interface{}) {
		mReceiver.neuron.Brain.MessageHandler(mReceiver.Const.tag, "receiverInit -> Open", 100, "Connected")
		mReceiver.Connection.receiverConn = data.(*websocket.Conn)
		// Heart Beat Run
		mReceiver.StopChannel.receiverLooperSC = make(chan bool)
		go mReceiver.neuron.Brain.SetInterval(func() (int, interface{}) {
			if !mReceiver.neuron.Brain.CheckIsNull(mReceiver.Connection.receiverConn) {
				var buf bytes.Buffer
				buf.WriteString(mReceiver.neuron.Brain.Const.NeuronId)
				buf.WriteString("#!HEART#**")
				if _, err := mReceiver.Connection.receiverConn.Write(mReceiver.neuron.Brain.SystemEncrypt(buf.Bytes())); err != nil {
					return 214, err
				}
			}
			return 100, nil
		}, func(code int, data interface{}) {
			if code != 100 {
				mReceiver.neuron.Brain.MessageHandler(mReceiver.Const.tag, "receiverInit -> Open", code, data)
			}
		}, mReceiver.neuron.Brain.Const.WSParam.Interval, mReceiver.StopChannel.receiverLooperSC)
	})

	mTrigger.On("Close", func(code int, data interface{}) {
		mReceiver.neuron.Brain.MessageHandler(mReceiver.Const.tag, "receiverInit -> Close", code, data)
		mReceiver.Connection.receiverConn = nil
		mReceiver.receiverKiller(true)
	})

	mTrigger.On("Error", func(code int, data interface{}) {
		mReceiver.neuron.Brain.MessageHandler(mReceiver.Const.tag, "receiverInit -> Error", code, data)
	})

	mTrigger.On("Message", func(code int, data interface{}) {
		msg := data.([]byte)
		if mReceiver.neuron.Brain.Const.RunEnv == 0 {
			mReceiver.neuron.Brain.MessageHandler(mReceiver.Const.tag, "receiverInit -> Message", 100, fmt.Sprintf("%X", msg))
		}
		// 解密
		msgData := mReceiver.neuron.Brain.SystemDecrypt(msg)
		if mReceiver.neuron.Brain.CheckIsNull(msgData) {
			mReceiver.neuron.Brain.MessageHandler(mReceiver.Const.tag, "receiverInit -> SystemDecrypt", 203, "[decodeData -> Error]")
		} else {
			// 解码
			GMessageArr := mReceiver.neuron.Brain.AnalyzeMessage(string(msgData))
			if mReceiver.neuron.Brain.CheckIsNull(GMessageArr) {
				mReceiver.neuron.Brain.MessageHandler(mReceiver.Const.tag, "receiverInit -> AnalyzeMessage", 203, "[decodeData -> Error]")
			} else {
				// 处理命令
				mReceiver.gMessageHandler(GMessageArr)
			}
		}
	})
	go mReceiver.neuron.Express.WSClient(mReceiver.neuron.Brain.Const.CommanderHost, mTrigger, mReceiver.neuron.Brain.Const.WSParam.Interval)
}

func (mReceiver *ReceiverS) receiverKiller(restart ...bool) {
	if !mReceiver.neuron.Brain.CheckIsNull(mReceiver.Connection.receiverConn) {
		mReceiver.Connection.receiverConn.Close()
	}
	if len(restart) > 0 {
		if restart[0] {
			go mReceiver.neuron.Brain.After(func() {
				// 重新开启通信线程
				mReceiver.receiverInit()
			})
		}
	}
}

//* 执行指令 */
func (mReceiver *ReceiverS) gMessageHandler(GMessageArr []*model.GMessageS) {
	for _, v := range GMessageArr {
		if v != nil {
			mReceiver.neuron.Brain.MessageHandler(mReceiver.Const.tag, "receiverInit -> GMessage", 100, []interface{}{v.ID, v.Head, v.Tag, v.Cmds})
			switch v.Head {
			case "!":
				break
			case "?":
				if v.Tag == "EVAL" {
					var args []interface{}
					args = append(args, mReceiver.Connection.receiverConn)
					args = append(args, v.ID)
					for k, v := range v.Cmds {
						if k != 0 && k != 1 {
							args = append(args, v)
						}
					}
					trigger.FireBackground("EVAL", v.Cmds[0].(string), v.Cmds[1].(string), args)
				}
				break
			case "~":
				break
			}
		}
	}
}

//* ================================ TOOL ================================ */

//* ================================ SERVICE ================================ */

//* 构造服务 */
func (mReceiver *ReceiverS) service() {
	// Receiver Websocket Init
	mReceiver.receiverInit()
}

//* 析构服务 */
func (mReceiver *ReceiverS) serviceKiller() {
	// Receiver Websocket Killer
	mReceiver.receiverKiller()
}

//* ================================ PUBLIC ================================ */

//* 构造本体 */
func (mReceiver *ReceiverS) Ontology(neuron *NeuronS, mux *http.ServeMux, root string) *ReceiverS {
	mReceiver.neuron = neuron
	mReceiver.mux = mux
	mReceiver.Const.tag = root[1:]
	mReceiver.Const.root = root

	if mReceiver.neuron.Brain.Const.AutorunConfig.ADReceiver {
		mReceiver.neuron.Brain.SafeFunction(mReceiver.main)
		mReceiver.StartService()
	} else {
		mReceiver.StopService()
	}
	return mReceiver
}

//* 返回开关量 */
func (mReceiver *ReceiverS) IsStarted() bool {
	return mReceiver.isStarted
}

//* 启动服务 */
func (mReceiver *ReceiverS) StartService() {
	if mReceiver.isStarted {
		return
	}
	mReceiver.isStarted = true
	go mReceiver.neuron.Brain.SafeFunction(mReceiver.service)
}

//* 停止服务 */
func (mReceiver *ReceiverS) StopService() {
	if !mReceiver.isStarted {
		return
	}
	mReceiver.isStarted = false
	go mReceiver.neuron.Brain.SafeFunction(mReceiver.serviceKiller)
}

//* 打印信息 */
func (mReceiver *ReceiverS) Log(title string, content ...interface{}) {
	if title == mReceiver.Const.tag {
		mReceiver.neuron.Brain.LogGenerater(model.LogTrace, mReceiver.Const.tag, title, fmt.Sprintf("%+v", mReceiver))
	} else {
		mReceiver.neuron.Brain.LogGenerater(model.LogInfo, mReceiver.Const.tag, title, content)
	}
}

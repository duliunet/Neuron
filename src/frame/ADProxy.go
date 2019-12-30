/**
===========================================================================
 * 代理服务器
 * Proxy Server
===========================================================================
*/

package frame

import (
	"fmt"
	"model"
	"modules/serial"
	"net/http"
)

//* ================================ DEFINE ================================ */
type ProxyS struct {
	Const struct {
		tag  string
		root string
	}
	Container struct {
		proxyConfig map[string]interface{}
	}
	Connection  struct{}
	StopChannel struct {
		tcp2tcpSCA  []chan bool
		udp2udpSCA  []chan bool
		udp2tcpSCA  []chan bool
		tcp2udpSCA  []chan bool
		uart2udpSCA []chan bool
	}
	isStarted bool
	neuron    *NeuronS
	mux       *http.ServeMux
}

//* ================================ PRIVATE ================================ */

//* 注册服务 */
func (mProxy *ProxyS) main() {
	/* Var */
	mProxy.StopChannel.tcp2tcpSCA = make([]chan bool, 0, 10)
	mProxy.StopChannel.udp2udpSCA = make([]chan bool, 0, 10)
	mProxy.StopChannel.udp2tcpSCA = make([]chan bool, 0, 10)
	mProxy.StopChannel.tcp2udpSCA = make([]chan bool, 0, 10)
	mProxy.StopChannel.uart2udpSCA = make([]chan bool, 0, 10)
	/* Func */
	mProxy.readConfig()
}

//* ================================ INTERFACE ================================ */

//* ================================ PROCESS ================================ */

//* 读取端口转发配置文件 */
func (mProxy *ProxyS) readConfig() {
	proxyHub := mProxy.neuron.Brain.Const.Proxy.ProxyHub
	if mProxy.neuron.Brain.CheckIsNull(proxyHub) {
		mProxy.Log("readConfig", "[proxyHub] -> Null")
		return
	}
	mProxy.Container.proxyConfig = proxyHub
}

//* 基于TCP协议的端口转发 */
func (mProxy *ProxyS) runTCP2TCP() {
	if !mProxy.neuron.Brain.CheckIsNull(mProxy.Container.proxyConfig["TCP"]) {
		for k, v := range mProxy.Container.proxyConfig["TCP"].(map[string]interface{}) {
			stopC := make(chan bool)
			go mProxy.neuron.Express.TCPForward(k, v.(string), stopC)
			mProxy.StopChannel.tcp2tcpSCA = append(mProxy.StopChannel.tcp2tcpSCA, stopC)
		}
	}
}

func (mProxy *ProxyS) killTCP2TCP() {
	if !mProxy.neuron.Brain.CheckIsNull(mProxy.StopChannel.tcp2tcpSCA) {
		for _, v := range mProxy.StopChannel.tcp2tcpSCA {
			mProxy.neuron.Brain.ClearInterval(v)
		}
	}
}

//* 基于UDP协议的端口转发 */
func (mProxy *ProxyS) runUDP2UDP() {
	if !mProxy.neuron.Brain.CheckIsNull(mProxy.Container.proxyConfig["UDP"]) {
		for k, v := range mProxy.Container.proxyConfig["UDP"].(map[string]interface{}) {
			stopC := make(chan bool)
			go mProxy.neuron.Express.UDPForward(k, v.(string), stopC)
			mProxy.StopChannel.udp2udpSCA = append(mProxy.StopChannel.udp2udpSCA, stopC)
		}
	}
}

func (mProxy *ProxyS) killUDP2UDP() {
	if !mProxy.neuron.Brain.CheckIsNull(mProxy.StopChannel.udp2udpSCA) {
		for _, v := range mProxy.StopChannel.udp2udpSCA {
			mProxy.neuron.Brain.ClearInterval(v)
		}
	}
}

//* UDP转TCP协议 */
func (mProxy *ProxyS) runUDP2TCP() {
	if !mProxy.neuron.Brain.CheckIsNull(mProxy.Container.proxyConfig["UDP2TCP"]) {
		for k, v := range mProxy.Container.proxyConfig["UDP2TCP"].(map[string]interface{}) {
			stopC := make(chan bool)
			go mProxy.neuron.Express.UDP2TCPForward(k, v.(string), stopC)
			mProxy.StopChannel.udp2tcpSCA = append(mProxy.StopChannel.udp2tcpSCA, stopC)
		}
	}
}

func (mProxy *ProxyS) killUDP2TCP() {
	if !mProxy.neuron.Brain.CheckIsNull(mProxy.StopChannel.udp2tcpSCA) {
		for _, v := range mProxy.StopChannel.udp2tcpSCA {
			mProxy.neuron.Brain.ClearInterval(v)
		}
	}
}

//* TCP转UDP协议 */
func (mProxy *ProxyS) runTCP2UDP() {
	if !mProxy.neuron.Brain.CheckIsNull(mProxy.Container.proxyConfig["TCP2UDP"]) {
		for k, v := range mProxy.Container.proxyConfig["TCP2UDP"].(map[string]interface{}) {
			stopC := make(chan bool)
			go mProxy.neuron.Express.TCP2UDPForward(k, v.(string), stopC)
			mProxy.StopChannel.tcp2udpSCA = append(mProxy.StopChannel.tcp2udpSCA, stopC)
		}
	}
}

func (mProxy *ProxyS) killTCP2UDP() {
	if !mProxy.neuron.Brain.CheckIsNull(mProxy.StopChannel.tcp2udpSCA) {
		for _, v := range mProxy.StopChannel.tcp2udpSCA {
			mProxy.neuron.Brain.ClearInterval(v)
		}
	}
}

//* UART转UDP协议 */
func (mProxy *ProxyS) runUART2UDP() {
	if !mProxy.neuron.Brain.CheckIsNull(mProxy.Container.proxyConfig["UART2UDP"]) {
		for k, v := range mProxy.Container.proxyConfig["UART2UDP"].(map[string]interface{}) {
			stopC := make(chan bool)
			option := serial.OpenOptions{
				PortName:        v.(map[string]interface{})["PortName"].(string),
				BaudRate:        uint(v.(map[string]interface{})["BaudRate"].(float64)),
				DataBits:        uint(v.(map[string]interface{})["DataBits"].(float64)),
				StopBits:        uint(v.(map[string]interface{})["StopBits"].(float64)),
				MinimumReadSize: uint(v.(map[string]interface{})["MinimumReadSize"].(float64)),
			}
			go mProxy.neuron.Express.UART2UDPForward(k, option, stopC)
			mProxy.StopChannel.uart2udpSCA = append(mProxy.StopChannel.uart2udpSCA, stopC)
		}
	}
}

func (mProxy *ProxyS) killUART2UDP() {
	if !mProxy.neuron.Brain.CheckIsNull(mProxy.StopChannel.uart2udpSCA) {
		for _, v := range mProxy.StopChannel.uart2udpSCA {
			mProxy.neuron.Brain.ClearInterval(v)
		}
	}
}

//* ================================ TOOL ================================ */

//* ================================ SERVICE ================================ */

//* 构造服务 */
func (mProxy *ProxyS) service() {
	mProxy.runTCP2TCP()
	mProxy.runUDP2UDP()
	mProxy.runUDP2TCP()
	mProxy.runTCP2UDP()
	mProxy.runUART2UDP()
}

//* 析构服务 */
func (mProxy *ProxyS) serviceKiller() {
	mProxy.killTCP2TCP()
	mProxy.killUDP2UDP()
	mProxy.killUDP2TCP()
	mProxy.killTCP2UDP()
	mProxy.killUART2UDP()
}

//* ================================ PUBLIC ================================ */

//* 构造本体 */
func (mProxy *ProxyS) Ontology(neuron *NeuronS, mux *http.ServeMux, root string) *ProxyS {
	mProxy.neuron = neuron
	mProxy.mux = mux
	mProxy.Const.tag = root[1:]
	mProxy.Const.root = root

	if neuron.Brain.Const.AutorunConfig.ADProxy {
		mProxy.neuron.Brain.SafeFunction(mProxy.main)
		mProxy.StartService()
	} else {
		mProxy.StopService()
	}
	return mProxy
}

//* 返回开关量 */
func (mProxy *ProxyS) IsStarted() bool {
	return mProxy.isStarted
}

//* 启动服务 */
func (mProxy *ProxyS) StartService() {
	if mProxy.isStarted {
		return
	}
	mProxy.isStarted = true
	go mProxy.neuron.Brain.SafeFunction(mProxy.service)
}

//* 停止服务 */
func (mProxy *ProxyS) StopService() {
	if !mProxy.isStarted {
		return
	}
	mProxy.isStarted = false
	go mProxy.neuron.Brain.SafeFunction(mProxy.serviceKiller)
}

//* 打印信息 */
func (mProxy *ProxyS) Log(title string, content ...interface{}) {
	if title == mProxy.Const.tag {
		mProxy.neuron.Brain.LogGenerater(model.LogTrace, mProxy.Const.tag, title, fmt.Sprintf("%+v", mProxy))
	} else {
		mProxy.neuron.Brain.LogGenerater(model.LogInfo, mProxy.Const.tag, title, content)
	}
}

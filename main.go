package main

import (
	"controller"
	"database/sql"
	"fmt"
	"frame"
	"model"
	"modules/logs/logger"
	"modules/trigger"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

//* ================================ DEFINE ================================ */

const tag = "Neuron"

//* Application */
var application struct {
	looperStopC chan bool
	// 神经元指针
	neuron *frame.NeuronS
	// 系统信号量
	sysExitSignalC chan os.Signal
	// 系统重启信号量
	sysRebootSignalC chan os.Signal
	// 服务系统
	serverHub *model.QueueS
}

//* ================================ EVENT ================================ */

//* 循环事件 */
func looperEvent() {
	brain := application.neuron.Brain
	// 开发环境显示服务器数量
	brain.LogGenerater(model.LogInfo, tag, "LooperEvent", fmt.Sprintf("Server Alive Count -> [%d]", application.serverHub.Len()))
	// 监视服务
	application.looperStopC = make(chan bool)
	brain.SetInterval(func() (int, interface{}) {
		for e := application.serverHub.Front(); e != nil; e = e.Next() {
			server, found := e.Value.(*model.ServerS)
			if !found {
				continue
			}
			select {
			case data := <-server.Alive:
				if !data {
					brain.LogGenerater(model.LogError, tag, "LooperEvent", fmt.Sprintf("Server Dead -> [%s]", server.Tag))
					application.serverHub.Delete(e)
					// 显示服务器数量
					brain.LogGenerater(model.LogInfo, tag, "LooperEvent", fmt.Sprintf("Server Alive Count -> [%d]", application.serverHub.Len()))
				}
			default:
				continue
			}
		}
		return 100, nil
	}, func(code int, data interface{}) {
		if code >= 200 {
			brain.LogGenerater(model.LogError, tag, "LooperEvent", fmt.Sprintf("Looper Error -> %s", data))
		}
	}, brain.Const.Interval.HZ1Interval, application.looperStopC)
}

//* 退出事件 */
func exitEvent(exitSignal ...string) {
	// 服务栈销毁
	for e := application.serverHub.Front(); e != nil; e = e.Next() {
		server, found := e.Value.(*model.ServerS)
		if !found {
			continue
		}
		for _, v := range server.Services {
			application.neuron.Brain.Eval(v, "StopService")
		}
	}
	// 停止looperEvent
	application.neuron.Brain.ClearInterval(application.looperStopC)
	// redis销毁
	if application.neuron.Redis != nil {
		application.neuron.Redis.Pool.Close()
	}
	// mysql销毁
	if application.neuron.Mysql != nil {
		application.neuron.Mysql.Pool.Iterator(func(n int, k string, v interface{}) bool {
			db, found := v.(*sql.DB)
			if !found {
				return true
			}
			db.Close()
			return true
		})
	}
	// logs保存
	logger.Flush()
	if len(exitSignal) == 0 {
		application.neuron.Brain.LogGenerater(model.LogInfo, tag, "", tag+" Stopped Gracefully..")
	} else {
		application.neuron.Brain.LogGenerater(model.LogInfo, tag, "", tag+" Stopped by Signal -> "+fmt.Sprintf("%s", exitSignal)+"..")
	}
	os.Exit(0)
}

//* 监听系统退出信号 */
func sysExitSignalEvent() {
	go func() {
		exitSignal := <-application.sysExitSignalC
		exitEvent(exitSignal.String())
	}()
	// remove SIGPIPE(管道broken) SIGFPE(浮点运算错误) SIGTRAP(断点)
	signal.Notify(application.sysExitSignalC, syscall.SIGINT, syscall.SIGABRT, syscall.SIGALRM, syscall.SIGBUS,
		syscall.SIGHUP, syscall.SIGILL, syscall.SIGKILL, syscall.SIGQUIT,
		syscall.SIGSEGV, syscall.SIGTERM)
}

func init() {
	// 初始化神经元
	application.neuron = new(frame.NeuronS).Ontology()
	// 初始化应用
	application.sysExitSignalC = make(chan os.Signal)
	// 初始化服务容器
	application.serverHub = new(model.QueueS).New(512)
	// 本地化脑
	brain := application.neuron.Brain
	// 欢迎信息
	brain.LogGenerater(model.LogInfo, tag, "", tag+" Starting..")
	// 监听系统信号
	sysExitSignalEvent()
}

//* ================================ SERVICE ================================ */

//* Server Process -> HTTP */
func serverProcess() *model.ServerS {
	// server init
	server := new(model.ServerS)
	server.Tag = "Express"
	server.Alive = make(chan bool)
	// server run
	go application.neuron.Brain.SafeFunction(func() {
		/* Map of all Services */
		server.Services = make(map[string]interface{})
		/* Construct Interface */
		mux := http.NewServeMux()
		/* Static Interface */
		mux.HandleFunc("/", application.neuron.Express.StaticHandler)
		/* Dev & Test Interface */
		if application.neuron.Brain.Const.RunEnv == 0 {
			/* Dev Code */
			mux.HandleFunc("/dev", func(res http.ResponseWriter, req *http.Request) {
				go devCode(req)
				res.Write([]byte("Run devCode.."))
			})
		}

		if application.neuron.Brain.Const.RunEnv == 1 {
			/* Dev Code */
			mux.HandleFunc("/test", func(res http.ResponseWriter, req *http.Request) {
				go testCode(req)
				res.Write([]byte("Run TEST Code.."))
			})
		}

		application.neuron.Brain.LogGenerater(model.LogInfo, tag, server.Tag, "Preparing..")

		/* AD -> System */
		systemRoot := "/System"
		system := new(frame.SystemS).Ontology(application.neuron, mux, systemRoot)
		server.Services[systemRoot] = system
		mux.HandleFunc(systemRoot, func(res http.ResponseWriter, req *http.Request) {
			application.neuron.Express.ConstructService(system, systemRoot, res, req)
		})

		/* AD -> Commander */
		commanderRoot := "/Commander"
		commander := new(frame.CommanderS).Ontology(application.neuron, mux, commanderRoot)
		server.Services[commanderRoot] = commander
		mux.HandleFunc(commanderRoot, func(res http.ResponseWriter, req *http.Request) {
			application.neuron.Express.ConstructService(commander, commanderRoot, res, req)
		})

		/* AD -> Receiver */
		receiverRoot := "/Receiver"
		receiver := new(frame.ReceiverS).Ontology(application.neuron, mux, receiverRoot)
		server.Services[receiverRoot] = receiver
		mux.HandleFunc(receiverRoot, func(res http.ResponseWriter, req *http.Request) {
			application.neuron.Express.ConstructService(receiver, receiverRoot, res, req)
		})

		/* AD -> Proxy */
		proxyRoot := "/Proxy"
		proxy := new(frame.ProxyS).Ontology(application.neuron, mux, proxyRoot)
		server.Services[proxyRoot] = proxy
		mux.HandleFunc(proxyRoot, func(res http.ResponseWriter, req *http.Request) {
			application.neuron.Express.ConstructService(proxy, proxyRoot, res, req)
		})

		/* AD -> ExamplePublish */
		examplePublishRoot := "/ExamplePublish"
		examplePublish := new(controller.ExamplePublishS).Ontology(application.neuron, mux, examplePublishRoot)
		server.Services[examplePublishRoot] = examplePublish
		mux.HandleFunc(examplePublishRoot, func(res http.ResponseWriter, req *http.Request) {
			application.neuron.Express.ConstructService(examplePublish, examplePublishRoot, res, req)
		})

		/* AD -> ExampleSubscribe */
		exampleSubscribeRoot := "/ExampleSubscribe"
		exampleSubscribe := new(controller.ExampleSubscribeS).Ontology(application.neuron, mux, exampleSubscribeRoot)
		server.Services[exampleSubscribeRoot] = exampleSubscribe
		mux.HandleFunc(exampleSubscribeRoot, func(res http.ResponseWriter, req *http.Request) {
			application.neuron.Express.ConstructService(exampleSubscribe, exampleSubscribeRoot, res, req)
		})

		/* Construct Application Trigger */
		trigger.On("EVAL", func(service string, function string, args []interface{}) {
			argArr := make([]interface{}, 0, len(args)-2)
			for _, v := range args[2:] {
				argArr = append(argArr, v)
			}
			if application.neuron.Brain.Const.RunEnv < 2 {
				application.neuron.Brain.LogGenerater(model.LogWarn, tag, "Eval -> "+service, fmt.Sprintf("%s(%s)", function, argArr))
			}
			application.neuron.Brain.Eval(server.Services[service], function, args...)
		})

		/* Construct Terminal Trigger */
		for k := range server.Services {
			trigger.On(k, func(args []string) {
				if len(args) < 2 {
					return
				}
				service := args[0]
				function := args[1]
				if application.neuron.Brain.Const.RunEnv < 2 {
					application.neuron.Brain.LogGenerater(model.LogTrace, tag, "Terminal -> "+service[1:], fmt.Sprintf("%s(%s)", function, args[2:]))
				}
				argArr := make([]interface{}, 0, len(args)-2)
				for _, vv := range args[2:] {
					argArr = append(argArr, vv)
				}
				application.neuron.Brain.Eval(server.Services[service], function, argArr...)
			})
		}

		application.neuron.Brain.LogGenerater(model.LogInfo, tag, server.Tag, "Prepared..")

		go protocolHTTP(server, mux)
		if application.neuron.Brain.Const.HTTPS.Open {
			go protocolTLS(server, mux)
		}
	})
	return server
}

func protocolHTTP(server *model.ServerS, mux *http.ServeMux) {
	// HTTP Listen Port
	listenPort := strconv.Itoa(application.neuron.Brain.Const.HTTPServer.Port)
	listenAddr := application.neuron.Brain.Const.HTTPServer.Host + ":" + listenPort
	application.neuron.Brain.LogGenerater(model.LogInfo, tag, server.Tag, "Listening port -> "+listenPort)
	err := http.ListenAndServe(listenAddr, mux)
	if err != nil {
		application.neuron.Brain.MessageHandler(tag, "Protocal -> HTTP", 204, err)
		protocolTLS(server, mux)
		return
	}
	server.Alive <- false
}

func protocolTLS(server *model.ServerS, mux *http.ServeMux) {
	// Listen Port
	listenPort := strconv.Itoa(application.neuron.Brain.Const.HTTPS.TLSPort)
	listenAddr := application.neuron.Brain.Const.HTTPServer.Host + ":" + listenPort
	application.neuron.Brain.LogGenerater(model.LogInfo, tag, server.Tag+"[TLS]", "Listening port -> "+listenPort)
	// Get Crt & Key
	crtPath := application.neuron.Brain.PathAbs(application.neuron.Brain.Const.HTTPS.TLSCertPath + ".crt")
	keyPath := application.neuron.Brain.PathAbs(application.neuron.Brain.Const.HTTPS.TLSCertPath + ".key")
	err := http.ListenAndServeTLS(listenAddr, crtPath, keyPath, mux)
	if err != nil {
		application.neuron.Brain.MessageHandler(tag, "Protocal -> TLS", 204, err)
	}
	server.Alive <- false
}

//* ================================ MAIN ================================ */

func main() {
	// push main process -> Neuron
	application.serverHub.Push(serverProcess())
	time.Sleep(time.Millisecond)

	// pprof server
	if application.neuron.Brain.Const.RunEnv < 2 {
		go http.ListenAndServe(fmt.Sprintf("%s:%d", application.neuron.Brain.Const.HTTPServer.Host, application.neuron.Brain.Const.HTTPServer.Port+1), nil)
	}

	if application.neuron.Brain.Const.RunEnv == 0 {
		go devCode(nil)
	}

	if application.neuron.Brain.Const.RunEnv == 1 {
		go testCode(nil)
	}

	looperEvent()
}

//* ================================ TEST ================================ */

// 测试代码
func testCode(request interface{}) {
	application.neuron.Brain.LogGenerater(model.LogInfo, tag, "TESTCode", "Runing..")

	application.neuron.Brain.LogGenerater(model.LogInfo, tag, "TESTCode", "Finished..")
}

//* ================================ DEVELOP ================================ */

// 开发代码
func devCode(request interface{}) {
	brain := application.neuron.Brain
	brain.LogGenerater(model.LogInfo, tag, "DevCode", "Runing..")

	brain.LogGenerater(model.LogInfo, tag, "DevCode", "Finished..")
}
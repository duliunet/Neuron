package model

//* ================================ DEFINE ================================ */
type autorunS struct {
	ADProxy     bool
	ADCommander bool
	ADReceiver  bool

	SDExamplePublish bool
	SDExampleSubscribe bool
}

type behaviorTreeS struct {
	ErrorQLen int
}

type databaseS struct {
	Open     bool
	Log      bool
	Host     string
	User     string
	Password string
	Database string
}

type redisS struct {
	Open     bool
	Host     string
	Port     int
	Password string
	Db       string
	Channel  string
}

type proxyS struct {
	ProxyHub  map[string]interface{}
}

type fileS struct {
	Chmod    int
	TempPath string
}

type requestS struct {
	DefaultHeader map[string][]string
}

type serverS struct {
	Host       string
	Port       int
	StaticPath string
	UploadPath string
	XPoweredBy string
	ACAO       bool
}

type tlsServerS struct {
	Open        bool
	TLSPort     int
	TLSCertPath string
}

type wsParamS struct {
	Interval   int
	BufferSize int
}

type tcpParamS struct {
	Interval   int
	BufferSize int
}

type udpParamS struct {
	MaxLen     int
	Interval   int
	BufferSize int
}

type uartParamS struct {
	Interval   int
	BufferSize int
}

type intervalS struct {
	HZ25Interval      int
	HZ8Interval       int
	HZ4Interval       int
	HZ2Interval       int
	HZ1Interval       int
	CommanderInterval int
	LooperInterval    int
	SystemInterval    int
	RetryInterval     int
	TwoHourInterval   int
}

//* ================================ PUBLIC ================================ */

type Const struct {
	/* Application Running Environment
		0 -> dev
		1 -> test
		2 -> produce
	*/
	RunEnv        int
	Version       string
	NeuronId      string
	SystemSplit   string
	CommanderHost string
	CommanderLog  bool
	BehaviorTree  behaviorTreeS
	AutorunConfig autorunS
	ErrorCode     map[int]string
	Database      databaseS
	Redis         redisS
	File          fileS
	Proxy         proxyS
	HTTPRequest   requestS
	HTTPServer    serverS
	HTTPS         tlsServerS
	WSParam       wsParamS
	TCPParam      tcpParamS
	UDPParam      udpParamS
	UartParam     uartParamS
	Interval      intervalS
}

//* 构造本体 */
func (*Const) Ontology() Const {
	return Const{
		0,
		"1.4.7",
		"Neuron",
		"___",
		"ws://127.0.0.1:8800/Commander/Channel",
		false,
		behaviorTreeS{
			512,
		},
		/* 自启动配置 */
		autorunS{
			true,
			true,
			true,

			true,
			true,
		},
		/* 错误代码 */
		map[int]string{
			100: "Success",
			101: "System Running",
			102: "System ShutDown",
			103: "Process ShutDown",
			104: "Process Timeout",
			105: "Return Stack",

			200: "Failed",
			201: "Interface Banned",
			202: "JSON Error",
			203: "Command Error",
			204: "System Error",
			205: "File Error",
			206: "Buffer Error",
			207: "Request Error",
			208: "Auth Error",
			209: "Encode/Decode Error",
			210: "TCP Conn Error",
			211: "UDP Conn Error",
			212: "Url Error",
			213: "Path Error",
			214: "Websocket Error",
			215: "Transform Error",
			216: "IOReader Error",
			217: "Platform Error",
			218: "Exec Error",
			219: "Convert Error",
			220: "Null Error",
			221: "DataType Error",
			222: "UART Error",

			300: "Database Disconnected",
			301: "Query Error",
			302: "TransAction Error",
			303: "RollBack Error",
			304: "Commit Error",
			305: "Data Analyze Error",

			400: "Redis Disconnected",
			401: "Redis Error",
		},
		/*数据库配置*/
		databaseS{
			false,
			false,
			"",
			"",
			"",
			"",
		},
		redisS{
			false,
			"",
			6379,
			"",
			"0",
			"",
		},
		fileS{
			0766,
			"/static/temp/",
		},
		proxyS{
			map[string]interface{}{
				"TCP":      map[string]interface{}{},
				"UDP":      map[string]interface{}{},
				"TCP2UDP":  map[string]interface{}{},
				"UDP2TCP":  map[string]interface{}{},
				"UART2UDP": map[string]interface{}{},
			},
		},
		requestS{
			map[string][]string{
				"User-Agent": {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.186 Safari/537.36"},
			},
		},
		serverS{
			"0.0.0.0",
			8800,
			"/static",
			"/static/upload",
			"Neuron",
			/* 跨域标识 */
			false,
		},
		tlsServerS{
			false,
			8443,
			"/tls/tls",
		},
		wsParamS{
			120000,
			2 << 20,
		},
		tcpParamS{
			120000,
			2 << 20,
		},
		udpParamS{
			1,
			120000,
			2 << 20,
		},
		uartParamS{
			120000,
			2 << 20,
		},
		intervalS{
			40,
			125,
			250,
			500,
			1000,
			100,
			3000,
			60000,
			5000,
			7200000,
		},
	}
}

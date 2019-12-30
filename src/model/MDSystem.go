package model

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

//* Log Type */
const (
	LogInfo int = iota
	LogDebug
	LogTrace
	LogWarn
	LogError
	LogCritical
)

//* 神经元服务 */
type ServerS struct {
	Tag   string
	Alive chan bool
	Services map[string]interface{}
}

//* 时间信息 */
type TimeS struct {
	YMD           string
	Week          time.Weekday
	Time          string
	Timestamp     string
	TimestampMill string
	TimestampNano string
	Datetime      time.Time
}

//* 内部消息 */
type MessageS struct {
	Code    int
	Message string
	Data    interface{}
}

//* 外部消息 */
type GMessageS struct {
	ID   string
	Head string
	Tag  string
	Cmds []interface{}
}

//* TCP通信中心为SyncMap形式 */

//* UDP通信中心为链表形式 */
type ConnQHub struct {
	//* 此Tag为QHub的名称 */
	Tag string
	//* [SocketClient] */
	ConnQ *QueueS
}

//* Socket客户端结构 */
type SocketClient struct {
	//* 此Tag为配置文件中NeuronId */
	Tag  string
	Conn interface{} /* ws -> [*websocket.Conn] | tcp -> [net.Conn] | */
}

//* Request参数 */
/*
var postData = {
	'msg': 'Hello World!'
};
var params = {
	postData : postData,
	hostname: 'www.google.com',
	port: 80,
	path: '/upload',
	headers: {
		'Content-Type': 'application/x-www-form-urlencoded',
		'Content-Length': Buffer.byteLength(postData)
	}
}
*/
type RequestParamS struct {
	PostData interface{}
	Host     string
	Path     string
	Header   map[string][]string
}

//* Response对象 */
type ResponseDataS struct {
	URLProxy *url.URL
	Header   http.Header
	Body     []byte
}

//* 数据库数据对象 */
type SQLDataS struct {
	Column []string
	// Rows.([]interface)[Column]
	Data []interface{}
}

//* UDP通信消息数据结构 */
type UDPPacket struct {
	Addr *net.UDPAddr
	Msg  []byte
}

//* CommanderQueue内容数据结构 */
type CommanderPiece struct {
	NeuronId string
	GMessage GMessageS
}
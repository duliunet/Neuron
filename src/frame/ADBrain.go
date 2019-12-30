/**
===========================================================================
 * 大脑 -> 公共方法集合
 * Brain -> public method collection
===========================================================================
*/
package frame

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	randc "crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"model"
	"modules/blowfish"
	"modules/decimal"
	"modules/logs/logger"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

//* ================================ DEFINE ================================ */

type BrainS struct {
	tag       string
	Const     model.Const
	Container struct {
		CommanderHub   model.SyncMapHub /* map[IP]SocketClient */
		CommanderQueue *model.QueueS
		CommanderReply *model.QueueS
	}
}

//* ================================ System Function ================================ */

//* 获取系统秘钥 */
func (brain *BrainS) SystemKey() []byte {
	var buf bytes.Buffer
	buf.WriteByte('E')
	buf.WriteByte('x')
	buf.WriteByte('a')
	buf.WriteByte('m')
	buf.WriteByte('p')
	buf.WriteByte('l')
	buf.WriteByte('e')
	buf.WriteByte('K')
	buf.WriteByte('e')
	buf.WriteByte('y')
	for _, v := range strings.Split(brain.Const.Version, ".") {
		buf.WriteString(v)
	}
	return buf.Bytes()
}

//* 获取混淆秘钥 */
func (brain *BrainS) SystemSalt() []byte {
	var buf bytes.Buffer
	buf.WriteByte('.')
	buf.WriteByte('S')
	buf.WriteByte('a')
	buf.WriteByte('l')
	buf.WriteByte('t')
	buf.WriteByte('.')
	return buf.Bytes()
}

//* 构造本体 */
func (brain *BrainS) Ontology() *BrainS {
	brain.tag = "Brain"
	brain.Const = brain.Const.Ontology()
	return brain
}

//* 生成UUID */
func (brain *BrainS) UUID(split ...string) string {
	splitStr := ""
	if !brain.CheckIsNull(split) {
		splitStr = split[0]
	}
	b := make([]byte, 16)
	_, _ = io.ReadFull(randc.Reader, b)
	b[6] = (b[6] & 0x0F) | 0x40
	b[8] = (b[8] &^ 0x40) | 0x80
	return fmt.Sprintf("%x%s%x%s%x%s%x%s%x", b[:4], splitStr, b[4:6], splitStr, b[6:8], splitStr, b[8:10], splitStr, b[10:])
}

//* 系统默认加密方法 */
func (brain *BrainS) SystemEncrypt(decryptData []byte) []byte {
	// Define
	var buf bytes.Buffer
	// Produce
	brain.SafeFunction(func() {
		// 随机位置
		n := brain.RandomInt(len(decryptData))
		buf.Write(decryptData[:n])
		buf.Write(brain.SystemSalt())
		buf.Write(decryptData[n:])
	})
	// Return
	return brain.HuffmanEncoder(brain.BfEncode(buf.Bytes()))
}

//* 系统默认解密方法 */
func (brain *BrainS) SystemDecrypt(encryptData []byte) []byte {
	// Define
	salt := brain.SystemSalt()
	var result []byte
	// Produce
	brain.SafeFunction(func() {
		encryptData = brain.HuffmanDecoder(encryptData)
		decryptData := brain.BfDecode(encryptData)
		// 如果不存在authKey
		if !bytes.Contains(decryptData, salt) {
			return
		}
		// 取出authKey
		decryptData = bytes.Replace(decryptData, salt, []byte{}, -1)
		result = decryptData
	})
	// Return
	return result
}

//* 根据NeuronSplit分割字符串 */
func (brain *BrainS) SystemSplit(str string) []string {
	return strings.Split(str, brain.Const.SystemSplit)
}

//* 根据操作系统执行系统命令判断 */
func (brain *BrainS) systemSelect(osCommand map[string]string) string {
	// 判断是否是合适的系统
	command, ok := osCommand[runtime.GOOS]
	if !ok {
		commandAll, okAll := osCommand["exec"]
		if !okAll {
			brain.LogGenerater(model.LogError, brain.tag, "systemSelect", fmt.Sprintf("Platform Not Support -> %s\n", runtime.GOOS))
			return ""
		}
		command = commandAll
	}
	return command
}

//* 执行系统指令 */
/* example:
var osCommand = map[string]string{
    "windows": "start",
    "darwin":  "open",
    "linux":   "xdg-open",
}
*/
func (brain *BrainS) SystemExec(callback func(cmd *exec.Cmd) (int, interface{}), osCommand map[string]string, params ...string) (int, interface{}) {
	var codeR int
	var dataR interface{}
	brain.SafeFunction(func() {
		// 执行命令
		command := brain.systemSelect(osCommand)
		cmd := exec.Command(command, params...)
		// 配置运行目录
		if !brain.CheckIsNull(osCommand["dir"]) {
			cmd.Dir = osCommand["dir"]
		}
		brain.LogGenerater(model.LogWarn, brain.tag, "SystemExec", fmt.Sprintf("[Exec] -> %s %v", command, params))
		codeR, dataR = callback(cmd)
	}, func(err interface{}) {
		if err == nil {
			return
		}
		codeR = 204
		dataR = err
	})
	return codeR, dataR
}

//* TryCatch实现 */
func (brain *BrainS) SafeFunction(next func(), callback ...func(err interface{})) {
	if brain.Const.RunEnv > 0 {
		defer func() {
			err := recover()
			if err != nil {
				// 捕获堆栈信息
				if brain.Const.RunEnv < 2 {
					var buf [102400]byte
					n := runtime.Stack(buf[:], false)
					err = fmt.Sprintf("[%v]\r\n%v", err, string(buf[:n]))
				}
				brain.LogGenerater(model.LogCritical, brain.tag, "SafeFunction", fmt.Sprintf("{%v -> %v}", brain.GetFuncName(next), err))
			}
			for _, v := range callback {
				v(err)
			}
		}()
	} else {
		defer func() {
			for _, v := range callback {
				v(nil)
			}
		}()
	}
	next()
}

//* 方法重试 */
/*
times -> 尝试次数
next -> 尝试方法
callback -> 尝试方法回调
Return -> 最终返回值[正确区间为100-199]
*/
func (brain *BrainS) Retry(times int, next func() (int, interface{}), callback func(code int, data interface{}), millisecond ...int) (code int, data interface{}) {
	var codeR int
	var dataR interface{}
	brain.SafeFunction(func() {
		millisecondInt := 1000
		if len(millisecond) > 0 {
			millisecondInt = millisecond[0]
		}
		for i := 0; i < times; i++ {
			code, data := next()
			callback(code, data)
			if code >= 100 && code < 200 {
				codeR = 100
				dataR = map[string]interface{}{"Try Times": i + 1}
				break
			} else {
				codeR = 200
				dataR = map[string]interface{}{"Try Times": i + 1}
			}
			time.Sleep(time.Duration(millisecondInt) * time.Millisecond)
		}
	}, func(err interface{}) {
		if err == nil {
			return
		}
		codeR = 204
		dataR = err
	})
	return codeR, dataR
}

//* 方法等待 */
func (brain *BrainS) After(callback func(), i ...int) *time.Timer {
	interval := brain.Const.Interval.RetryInterval
	if len(i) > 0 {
		interval = i[0]
	}
	return time.AfterFunc(time.Duration(interval)*time.Millisecond, func() {
		brain.SafeFunction(func() {
			callback()
		})
	})
}

//* 永久循环 */
/*
stopC -> true则停止循环
needTimeDevides -> true则根据返回值code == 100时的data为int数量将循环时间等分切片,用于分布式客户端命令执行
*/

func (brain *BrainS) SetInterval(next func() (int, interface{}), callback func(code int, data interface{}), interval int, stopC chan bool, needTimeDevides ...bool) {
	nextName := brain.GetFuncName(next)
	if brain.CheckIsNull(stopC) {
		brain.LogGenerater(model.LogError, brain.tag, "SetInterval", fmt.Sprintf("%s -> Lack of Stop Channel", nextName))
		return
	}
	if brain.Const.RunEnv < 2 {
		brain.LogGenerater(model.LogDebug, brain.tag, "SetInterval", nextName)
	}
	var wg sync.WaitGroup
	endC := make(chan map[int]interface{})
	msgC := make(chan map[int]interface{})
	defer func() {
		wg.Wait()
		close(stopC)
		close(endC)
		close(msgC)
	}()
	// Timer Init
	duration := time.Duration(interval) * time.Millisecond
	mTimer := time.NewTimer(duration)
	defer mTimer.Stop()
	// Timer Runnable
	for {
		select {
		// Exit Handler
		case data := <-stopC:
			if data {
				callback(103, nextName)
				return
			}
		default:
			wg.Add(1)
			go brain.SafeFunction(func() {
				code, data := next()
				if code == 100 {
					msgC <- map[int]interface{}{code: data}
				} else {
					endC <- map[int]interface{}{code: data}
				}
			}, func(err interface{}) {
				wg.Done()
				if err == nil {
					return
				}
				endC <- map[int]interface{}{204: err}
			})
		}
		select {
		case data := <-endC:
			// Error Handler
			for k, v := range data {
				callback(k, v)
			}
			return
		case <-mTimer.C:
			// Message Handler
			msg := <-msgC
			brain.SafeFunction(func() {
				for k, v := range msg {
					callback(k, v)
				}
			})
			// Reset duration
			mduraion := duration
			if len(needTimeDevides) > 0 {
				if needTimeDevides[0] {
					var codeR int
					var dataR interface{}
					for k, v := range msg {
						codeR = k
						dataR = v
					}
					if codeR == 100 {
						// 如果需要时间切片
						count, found := dataR.(int)
						if found {
							if count == 0 {
								count = 1
							}
							mduraion = time.Duration(interval/count) * time.Millisecond
						}
					}
				}
			}
			mTimer.Reset(mduraion)
		}
	}
}

//* 结束永久循环 */
func (brain *BrainS) ClearInterval(stopC chan bool) {
	defer func() {
		if err := recover(); err != nil {
			brain.MessageHandler(brain.tag, "ClearInterval", 204, err)
		}
	}()
	timeout := make(chan bool)
	defer close(timeout)
	go brain.After(func() {
		if !brain.CheckIsNull(timeout) {
			timeout <- true
		}
	}, brain.Const.Interval.HZ1Interval)
	for {
		select {
		case <-timeout:
			return
		case stopC <- true:
			return
		}
	}
}

//* 超时循环 */
/*
104 -> 超时
*/
func (brain *BrainS) SetTimeoutInterval(next func() (int, interface{}), callback func(code int, data interface{}), interval int, timeout int) {
	nextName := brain.GetFuncName(next)
	var wg sync.WaitGroup
	endC := make(chan map[int]interface{})
	msgC := make(chan map[int]interface{})
	defer func() {
		wg.Wait()
		close(endC)
		close(msgC)
	}()
	// Timer Init
	duration := time.Duration(interval) * time.Millisecond
	times := timeout / interval
	timesRun := 0
	if times < 1 {
		brain.LogGenerater(model.LogError, brain.tag, fmt.Sprintf("SetTimeoutInterval[%s] -> %dms", nextName, timeout), "Warning -> Times lower than once")
		return
	}
	mTimer := time.NewTimer(duration)
	defer mTimer.Stop()
	// Timer Runnable
	for {
		wg.Add(1)
		go brain.SafeFunction(func() {
			code, data := next()
			if code == 100 {
				msgC <- map[int]interface{}{code: timesRun}
			} else {
				endC <- map[int]interface{}{code: data}
			}
		}, func(err interface{}) {
			wg.Done()
			if err == nil {
				return
			}
			endC <- map[int]interface{}{204: err}
		})
		select {
		case data := <-endC:
			for k, v := range data {
				callback(k, v)
			}
			return
		case <-mTimer.C:
			data := <-msgC
			// Message Handler
			for k, v := range data {
				callback(k, v)
			}
			if timesRun++; timesRun >= times {
				callback(104, "Timeout Occured")
				return
			}
			mTimer.Reset(duration)
		}
	}
}

//* 信息处理 */
func (brain *BrainS) MessageHandler(tag string, function string, code int, data interface{}) model.MessageS {
	// Code & Data Output
	logtype := model.LogInfo
	if code > 100 && code < 200 {
		logtype = model.LogWarn
	} else if code >= 200 {
		logtype = model.LogError
	}
	message := brain.Const.ErrorCode[code]
	if brain.CheckIsNull(message) {
		message = brain.Const.ErrorCode[200]
	}
	msgs := model.MessageS{
		Code:    code,
		Message: message,
		Data:    data,
	}
	brain.LogGenerater(logtype, tag, function, msgs)
	return msgs
}

//* 结构化日志记录 */
func (brain *BrainS) LogGenerater(logtype int, model string, function string, content interface{}) {
	if brain.CheckIsNull(content) {
		content = ""
	}
	timenow := "[" + time.Now().Format("2006-01-02 15:04:05") + "] "
	if function != "" {
		function = "_" + function
	}

	switch logtype {
	case 0:
		logger.Info(timenow + "[Info] " + brain.Const.NeuronId + "[" + brain.Const.Version + "]" + " - [" + model + function + "] => " + fmt.Sprintf("%+v", content))
	case 1:
		logger.Debug(timenow + "[Debug] " + brain.Const.NeuronId + "[" + brain.Const.Version + "]" + " - [" + model + function + "] => " + fmt.Sprintf("%+v", content))
	case 2:
		logger.Trace(timenow + "[Trace] " + brain.Const.NeuronId + "[" + brain.Const.Version + "]" + " - [" + model + function + "] => " + fmt.Sprintf("%+v", content))
	case 3:
		logger.Warn(timenow + "[Warn] " + brain.Const.NeuronId + "[" + brain.Const.Version + "]" + " - [" + model + function + "] => " + fmt.Sprintf("%+v", content))
	case 4:
		logger.Error(timenow + "[Error] " + brain.Const.NeuronId + "[" + brain.Const.Version + "]" + " - [" + model + function + "] => " + fmt.Sprintf("%+v", content))
	case 5:
		logger.Critical(timenow + "[Critical] " + brain.Const.NeuronId + "[" + brain.Const.Version + "]" + " - [" + model + function + "] => " + fmt.Sprintf("%+v", content))
	}
}

//* 反射执行内部代码 */
func (brain *BrainS) Eval(service interface{}, function string, args ...interface{}) {
	brain.SafeFunction(func() {
		inputs := make([]reflect.Value, len(args))
		for i := range args {
			inputs[i] = reflect.ValueOf(args[i])
		}
		if reflect.ValueOf(service).MethodByName(function).IsValid() {
			reflect.ValueOf(service).MethodByName(function).Call(inputs)
		} else {
			brain.MessageHandler(brain.tag, "EVAL", 200, "Reflect Error -> "+reflect.ValueOf(service).String()+"."+function+"("+fmt.Sprintf("%v)", args))
		}
	})
}

//* 生成指令 */
/*
* param:
*   <ID>#<head><tag>#<cmds>**
*	123#?SYSINFO#token#**
* return:
*   [string]
 */
func (brain *BrainS) GenerateMessage(head string, tag string, cmds []interface{}, id ...string) *bytes.Buffer {
	ids := ""
	if len(id) != 0 {
		ids = id[0]
	}
	ids = strings.Replace(ids, "#", "", -1)
	head = strings.Replace(head, "#", "", -1)
	tag = strings.Replace(tag, "#", "", -1)
	GMessage := new(bytes.Buffer)
	if len(id) != 0 {
		ids += "#"
	}
	GMessage.WriteString(ids)
	GMessage.WriteString(head)
	GMessage.WriteString(tag)
	if len(cmds) > 0 {
		GMessage.WriteString("#")
	}
	for k, v := range cmds {
		GMessage.Write(brain.Interface2Bytes(v))
		if k != len(cmds)-1 {
			GMessage.WriteString("#")
		}
	}
	GMessage.WriteString("**")
	return GMessage
}

/**
* 解析指令
* param:
*   msg => 指令体[string]
* return:
*   object => [
*      id => 指令编号
*      head => 指挥头,判断是指令(?)/消息(!)/对象(~)[string]
*      tag => 指挥者,判断来自哪个模块[string]
*      cmd_array => 指令体,获取指令集[array]
*   ]
 */
func (brain *BrainS) AnalyzeMessage(msgs string, splits ...string) []*model.GMessageS {
	var result []*model.GMessageS
	brain.SafeFunction(func() {
		split := "**"
		if len(splits) > 0 {
			split = splits[0]
		}
		msgArr := strings.Split(msgs, split)
		msgArr = msgArr[:len(msgArr)-1]
		msgObj := make([]*model.GMessageS, len(msgArr))
		for k := range msgArr {
			msgArr[k] += split
			msgObj[k] = brain.analyzeMessage(msgArr[k], split)
		}
		result = msgObj
	}, func(err interface{}) {
		if err == nil {
			return
		}
		result = nil
	})
	return result
}

func (brain *BrainS) analyzeMessage(msg string, split string) *model.GMessageS {
	if brain.CheckIsNull(msg) {
		return nil
	}
	var msgId, msgHead, msgTag string
	var msgCmds []interface{}
	// 分隔符
	separator := "#"
	msgStr := msg
	// 尾部不能没有**
	if strings.Index(msgStr, split) == -1 {
		return nil
	}
	msgStr = strings.Replace(msgStr, "\r", "", -1)
	msgStr = strings.Replace(msgStr, "\n", "", -1)
	// 判断是否存在ID(条件为第一个#的位置前于head)
	msgHeadIndex := strings.Index(msgStr, "?")
	if msgHeadIndex == -1 {
		msgHeadIndex = strings.Index(msgStr, "!")
		if msgHeadIndex == -1 {
			msgHeadIndex = strings.Index(msgStr, "~")
		}
	}
	// 不能没有头部
	if msgHeadIndex == -1 {
		return nil
	}
	firstSplitIndex := strings.Index(msgStr, separator)
	// 如果不存在separator
	if firstSplitIndex != -1 {
		if firstSplitIndex < msgHeadIndex {
			// 存在ID
			msgId = msgStr[0:firstSplitIndex]
		} else {
			// 不存在ID
			firstSplitIndex = -1
		}
	}
	// 重构没有ID的GMessage
	msgStr = msgStr[firstSplitIndex+1:]
	msgHead = msgStr[0:1]
	secondSplitIndex := strings.Index(msgStr, separator)
	// 如果不存在secondSplitIndex则只有Head和Tag
	if secondSplitIndex < 0 {
		msgTag = msgStr[1 : len(msgStr)-2]
	} else {
		msgTag = msgStr[1:strings.Index(msgStr, separator)]
		msgCmdStr := msgStr[strings.Index(msgStr, separator)+1 : len(msgStr)-2]
		msgCmdArr := strings.Split(msgCmdStr, separator)
		msgCmds = make([]interface{}, len(msgCmdArr))
		for k, v := range msgCmdArr {
			msgCmds[k] = v
		}
	}

	GMessage := new(model.GMessageS)
	GMessage.ID = msgId
	GMessage.Head = msgHead
	GMessage.Tag = msgTag
	GMessage.Cmds = msgCmds
	return GMessage
}

//* 构造绝对路径 */
func (brain *BrainS) PathAbs(dirPath string) string {
	dirPath = strings.Replace(dirPath, "../", "", -1)
	if dirPath[:1] != "/" {
		dirPath = "/" + dirPath
	}
	return path.Dir(os.Args[0]) + dirPath
}

//* 相对路径转绝对路径 */
func (brain *BrainS) FilePath2AbsPath(dirPath string) string {
	retPath, err := filepath.Abs(dirPath)
	if err != nil {
		brain.MessageHandler(brain.tag, "FilePath2AbsPath", 200, err)
		retPath = dirPath
	}
	return retPath
}

//* 获取绝对路径中的文件路径 */
func (brain *BrainS) PathAbs2PathBaseExt(dirPath string) (string, string, string) {
	dirPath = path.Clean(dirPath)
	base := path.Base(dirPath)
	ext := path.Ext(dirPath)
	return path.Dir(dirPath), base[:(len(base) - len(ext))], ext
}

//* 查询文件夹 */
func (brain *BrainS) PathExists(dirPath string) bool {
	_, err := os.Stat(dirPath)
	if err != nil {
		return false
	} else {
		return true
	}
}

//* 列出文件夹 */
func (brain *BrainS) PathList(dirPath string) (int, interface{}) {
	info, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return 205, err
	} else {
		return 100, info
	}
}

//* 创建文件夹 */
func (brain *BrainS) PathCreate(dirPath string) (int, interface{}) {
	if brain.PathExists(dirPath) {
		return 100, nil
	} else {
		err := os.MkdirAll(dirPath, os.FileMode(brain.Const.File.Chmod))
		if err != nil {
			return 205, err
		} else {
			return 100, nil
		}
	}
}

//* 文件读 */
func (brain *BrainS) FileReader(filePath string) (int, interface{}) {
	var codeR int
	var dataR interface{}
	brain.SafeFunction(func() {
		dirPath := path.Dir(filePath)
		if brain.PathExists(dirPath) {
			fileBuffer, err := ioutil.ReadFile(filePath)
			if err != nil {
				codeR = 205
				dataR = err
			} else {
				codeR = 100
				dataR = fileBuffer
			}
		} else {
			codeR = 205
			dataR = errors.New("[FileReader] file path Error")
		}
	})
	return codeR, dataR
}

//* 文件写 */
func (brain *BrainS) FileWriter(filePath string, data []byte) (int, interface{}) {
	var codeR int
	var dataR interface{}
	brain.SafeFunction(func() {
		dirPath := path.Dir(filePath)
		if code, err := brain.PathCreate(dirPath); code == 100 {
			err := ioutil.WriteFile(filePath, data, os.FileMode(brain.Const.File.Chmod))
			if err != nil {
				codeR = 205
				dataR = err
			} else {
				codeR = 100
				dataR = nil
			}
		} else {
			codeR = code
			dataR = err
		}
	})
	return codeR, dataR
}

//* 文件追加 */
func (brain *BrainS) FileAppend(filePath string, data []byte) (int, interface{}) {
	var codeR int
	var dataR interface{}
	brain.SafeFunction(func() {
		dirPath := path.Dir(filePath)
		if code, err := brain.PathCreate(dirPath); code == 100 {
			f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.FileMode(brain.Const.File.Chmod))
			n, err := f.Write(data)
			if err == nil && n < len(data) {
				err = io.ErrShortWrite
			}
			if err1 := f.Close(); err == nil {
				err = err1
			}
			if err != nil {
				codeR = 205
				dataR = err
			} else {
				codeR = 100
				dataR = nil
			}
		} else {
			codeR = code
			dataR = err
		}
	})
	return codeR, dataR
}

//* 文件重命名 */
func (brain *BrainS) FileRename(filePath string, filename string) (int, interface{}) {
	var codeR int
	var dataR interface{}
	brain.SafeFunction(func() {
		Path, _, _ := brain.PathAbs2PathBaseExt(filePath)
		newPath := fmt.Sprintf("%s/%s", Path, filename)
		if err := os.Rename(filePath, newPath); err != nil {
			codeR = 200
			dataR = newPath
		} else {
			codeR = 100
			dataR = newPath
		}
	})
	return codeR, dataR
}

//* 文件重命名 */
func (brain *BrainS) FileMove(filePath string, newPath string) (int, interface{}) {
	var codeR int
	var dataR interface{}
	brain.SafeFunction(func() {
		if err := os.Rename(filePath, newPath); err != nil {
			codeR = 200
			dataR = newPath
		} else {
			codeR = 100
			dataR = newPath
		}
	})
	return codeR, dataR
}

//* 文件删除 */
func (brain *BrainS) FileRemove(filePath string) (int, interface{}) {
	var codeR int
	var dataR interface{}
	brain.SafeFunction(func() {
		if err := os.Remove(filePath); err != nil {
			codeR = 200
			dataR = err
		} else {
			codeR = 100
			dataR = nil
		}
	})
	return codeR, dataR
}

//* 文件夹删除 */
func (brain *BrainS) FileRemovAll(DirPath string) (int, interface{}) {
	var codeR int
	var dataR interface{}
	brain.SafeFunction(func() {
		if err := os.RemoveAll(DirPath); err != nil {
			codeR = 200
			dataR = err
		} else {
			codeR = 100
			dataR = nil
		}
	})
	return codeR, dataR
}

//* 持久化结构体存储 -> Bytes */
func (brain *BrainS) Bytes2File(FileName string, b []byte) (int, interface{}) {
	DirPath := brain.PathAbs(fmt.Sprintf("/data/%s.dat", FileName))
	return brain.FileWriter(DirPath, b)
}

//* 持久化结构体读取 -> Bytes */
func (brain *BrainS) File2Bytes(FileName string) (int, interface{}) {
	DirPath := brain.PathAbs(fmt.Sprintf("/data/%s.dat", FileName))
	code, data := brain.FileReader(DirPath)
	if code != 100 {
		return code, data
	}
	return 100, data.([]byte)
}

//* 持久化结构体存储 -> 直接输入结构体 */
func (brain *BrainS) Struct2File(FileName string, s interface{}) (int, interface{}) {
	DirPath := brain.PathAbs(fmt.Sprintf("/data/%s.dat", FileName))
	var buf bytes.Buffer
	buf.Write(brain.SystemEncrypt(brain.JsonEncoder(s)))
	return brain.FileWriter(DirPath, buf.Bytes())
}

//* 持久化结构体读取 -> 输出结构体JSONString[需套用结构解析] */
func (brain *BrainS) File2Struct(FileName string) (int, interface{}) {
	DirPath := brain.PathAbs(fmt.Sprintf("/data/%s.dat", FileName))
	var buf bytes.Buffer
	code, data := brain.FileReader(DirPath)
	if code != 100 {
		return code, data
	}
	buf.Write(brain.SystemDecrypt(data.([]byte)))
	return 100, buf.Bytes()
}

//* HTTP Request */
/* 构造Post方法 -> [
	host, path := express.Url2HostPath(u)
	postData := url.Values{}
	postData.Add("key", "value")
	header := brain.Const.HTTPRequest.DefaultHeader
	header["Content-Type"] = []string{"application/x-www-form-urlencoded"}
	brain.Request(model.RequestParamS{PostData: postData.Encode(), Host: host, Path: path, Header: header}, func(code int, data interface{}) {})
]
*/
func (brain *BrainS) Request(params model.RequestParamS, callback func(code int, data interface{}), proxyUrls ...*url.URL) {
	codeC := make(chan int)
	dataC := make(chan interface{})
	defer close(codeC)
	defer close(dataC)
	go brain.SafeFunction(func() {
		// 默认PATH
		if brain.CheckIsNull(params.Path) {
			params.Path = "/"
		}
		// 判断request方法
		method := "GET"
		// 构建io.Reader
		var body io.Reader
		if !brain.CheckIsNull(params.PostData) {
			method = "POST"

			postData, found := params.PostData.(string)
			if !found {
				codeC <- 221
				dataC <- "postData Type Error"
				return
			}

			body = strings.NewReader(postData)
			// 判断类型增加头
			if brain.CheckIsNull(params.Header["Content-Type"]) {
				if brain.JsonChecker([]byte(postData)) {
					params.Header["Content-Type"] = []string{"application/json"}
				}
			}
		}
		// 补协议头
		protocal := ""
		if !strings.Contains(params.Host, "http://") && !strings.Contains(params.Host, "https://") {
			protocal = "http://"
		}
		// Query转uri
		uri, err := url.Parse(params.Path)
		if err != nil {
			codeC <- 207
			dataC <- err
			return
		}
		getParams := url.PathEscape(uri.RawQuery)
		if getParams != "" {
			getParams = "?" + getParams
		}
		u := protocal + params.Host + uri.Path + getParams
		// Query Request
		req, err := http.NewRequest(method, u, body)
		if err != nil {
			codeC <- 207
			dataC <- err
			return
		}
		req.Header = params.Header
		client := &http.Client{}
		if len(proxyUrls) > 0 {
			proxyUrl := proxyUrls[0]
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			}
			client = &http.Client{
				Transport: transport,
			}
		}
		res, err := client.Do(req)
		if err != nil {
			codeC <- 207
			dataC <- err
			return
		}
		// 构建io.Reader
		var reader io.ReadCloser
		switch res.Header.Get("Content-Encoding") {
		case "gzip":
			reader, err = gzip.NewReader(res.Body)
		default:
			reader = res.Body
		}
		resBody, err := ioutil.ReadAll(reader)
		res.Body.Close()
		reader.Close()
		if err != nil {
			codeC <- 207
			dataC <- err
			return
		}
		codeC <- 100
		dataC <- model.ResponseDataS{URLProxy: res.Request.URL, Header: res.Header, Body: resBody}
	})
	// 回调错误抛出
	brain.SafeFunction(func() {
		callback(<-codeC, <-dataC)
	}, func(err interface{}) {
		if err == nil {
			return
		}
		brain.MessageHandler(brain.tag, "Request -> ["+params.Host+params.Path+"]", 204, err)
	})
}

//* 同步执行Request */
func (brain *BrainS) RequestSync(param model.RequestParamS, proxyUrls ...*url.URL) (int, interface{}) {
	var codeR int
	var dataR interface{}
	brain.Request(param, func(code int, data interface{}) {
		if code == 100 {
			codeR = 100
			dataR = data /*model.ResponseDataS*/
		} else {
			codeR = code
			dataR = data
		}
	}, proxyUrls...)
	return codeR, dataR
}

//* RequestSync -> 取Cookies */
func (brain *BrainS) RequestSync4Cookies(param model.RequestParamS, proxyUrls ...*url.URL) (int, interface{}) {
	var codeR int
	var dataR interface{}
	brain.Request(param, func(code int, data interface{}) {
		if code == 100 {
			codeR = 100
			res, found := data.(model.ResponseDataS)
			if !found {
				codeR = 221
				dataR = "ResponseDataS is Null"
			}
			dataR = res.Header["Set-Cookie"]
		} else {
			codeR = code
			dataR = data
		}
	}, proxyUrls...)
	return codeR, dataR
}

//* MultipartFile Request */
func (brain *BrainS) RequestMultipartFile(params model.RequestParamS, filePathHub map[string]string, callback func(code int, data interface{})) {
	defer runtime.GC()
	codeC := make(chan int)
	dataC := make(chan interface{})
	defer close(codeC)
	defer close(dataC)
	go brain.SafeFunction(func() {
		// DEFINE
		encodeFlag := false
		if !brain.CheckIsNull(params.Header["Content-Encoding"]) {
			if params.Header["Content-Encoding"][0] == "gzip" {
				encodeFlag = true
			}
		}
		var bodyBuf bytes.Buffer
		bodyWriter := multipart.NewWriter(&bodyBuf)
		// Set Boundary
		err := bodyWriter.SetBoundary(fmt.Sprintf("__%s__", strings.Replace(brain.Const.HTTPServer.XPoweredBy, " ", "_", -1)))
		if err != nil {
			brain.MessageHandler(brain.tag, "RequestMultipartFile  -> SetBoundary", 204, err)
		}
		// Set FileBuffer
		for k, v := range filePathHub {
			_, base, ext := brain.PathAbs2PathBaseExt(v)
			filename := base + ext
			fileWriter, err := bodyWriter.CreateFormFile(k, filename)
			if err != nil {
				brain.MessageHandler(brain.tag, "RequestMultipartFile  -> CreateFormFile["+filename+"]", 205, err)
				continue
			}
			// 获取文件数据
			code, data := brain.FileReader(v)
			if code != 100 {
				brain.MessageHandler(brain.tag, "RequestMultipartFile -> FileReader["+filename+"]", code, data)
				continue
			}
			// 补充gzip压缩
			if encodeFlag {
				gzipData, _ := brain.GzipEncoder(data.([]byte))
				fileWriter.Write(gzipData)
			} else {
				fileWriter.Write(data.([]byte))
			}
		}
		bodyWriter.Close()
		params.Header["Content-Type"] = []string{bodyWriter.FormDataContentType()}
		// 补协议头
		protocal := ""
		if !strings.Contains(params.Host, "http://") && !strings.Contains(params.Host, "https://") {
			protocal = "http://"
		}
		// Query转uri
		uri, err := url.Parse(params.Path)
		if err != nil {
			codeC <- 207
			dataC <- err
			return
		}
		getParams := url.PathEscape(uri.RawQuery)
		if getParams != "" {
			getParams = "?" + getParams
		}
		u := protocal + params.Host + uri.Path + getParams
		req, err := http.NewRequest("POST", u, &bodyBuf)
		if err != nil {
			codeC <- 207
			dataC <- err
			return
		}
		req.Header = params.Header
		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			codeC <- 207
			dataC <- err
			return
		}
		// 构建io.Reader
		var reader io.ReadCloser
		switch res.Header.Get("Content-Encoding") {
		case "gzip":
			reader, err = gzip.NewReader(res.Body)
		default:
			reader = res.Body
		}
		resBody, err := ioutil.ReadAll(reader)
		res.Body.Close()
		reader.Close()
		if err != nil {
			codeC <- 207
			dataC <- err
			return
		}
		codeC <- 100
		dataC <- model.ResponseDataS{URLProxy: res.Request.URL, Header: res.Header, Body: resBody}
	})
	// 回调错误抛出
	brain.SafeFunction(func() {
		callback(<-codeC, <-dataC)
	}, func(err interface{}) {
		if err == nil {
			return
		}
		brain.MessageHandler(brain.tag, "RequestMultipart -> ["+params.Host+params.Path+"]", 204, err)
	})
}

//* 获取一个Terminal指令 */
func (brain *BrainS) TerminalInput(notices ...string) (int, interface{}) {
	os.Stdin.Sync()
	// Notice
	for _, v := range notices {
		fmt.Println(v)
	}
	// Define
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	line := scanner.Text()
	return 100, line
}

func (brain *BrainS) YV12ToRGB24(yv12 []byte, width int, height int) (int, interface{}) {
	// 参数判断
	if width == 0 || height == 0 || len(yv12) == 0 {
		return 200, "width/height/yv12 Error"
	}
	// 长度构造
	nYLen := width * height
	halfWidth := width >> 1
	if nYLen < 1 || halfWidth < 1 {
		return 200, "nYLen/halfWidth Error"
	}
	// 数据构造
	rgb24 := make([]byte, width*height*3)
	rgb := make([]int, 3)
	var i, j, m, n int
	m = -width
	n = -halfWidth
	// 数据计算
	for y := 0; y < height; y++ {
		m += width
		if y%2 != 0 {
			n += halfWidth
		}
		for x := 0; x < width; x++ {
			i = m + x
			j = n + (x >> 1)
			// r
			rgb[2] = int(float64(yv12[i]&0xFF) + 1.370705*(float64(yv12[nYLen+j]&0xFF)-float64(128)))
			// g
			rgb[1] = int(float64(yv12[i]&0xFF) + 0.698001*(float64(yv12[nYLen+(nYLen>>2)+j]&0xFF)-float64(128)) - 0.703125*(float64(yv12[nYLen+j]&0xFF)-float64(128)))
			// b
			rgb[0] = int(float64(yv12[i]&0xFF) + 1.732446*(float64(yv12[nYLen+(nYLen>>2)+j]&0xFF)-float64(128)))
			//j = nYLen - iWidth - m + x;
			//i = (j<<1) + j;    //图像是上下颠倒的
			j = m + x
			i = (j << 1) + j
			for j = 0; j < 3; j++ {
				if rgb[j] >= 0 && rgb[j] <= 255 {
					rgb24[i+j] = byte(rgb[j])
				} else {
					if rgb[j] < 0 {
						rgb24[i+j] = 0
					} else {
						rgb24[i+j] = 255
					}
				}
			}
		}
	}
	return 100, rgb24
}

//* ================================ Utils Function ================================ */

//* 判断是否为空 */
func (brain *BrainS) CheckIsNull(i interface{}) bool {
	if brain.Const.RunEnv != 0 {
		defer func() {
			if err := recover(); err != nil {
				brain.MessageHandler(brain.tag, "CheckIsNull", 204, err)
			}
		}()
	}
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Struct, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Float32, reflect.Float64:
		return false
	case reflect.Slice, reflect.Array:
		if v.IsNil() {
			return true
		} else {
			if v.Len() == 0 {
				return true
			} else {
				return false
			}
		}
	case reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface:
		if v.IsNil() {
			return true
		} else {
			return false
		}
	case reflect.Chan:
		if v.IsNil() {
			return true
		}
		switch i.(type) {
		case chan bool:
			select {
			case _, ok := <-i.(chan bool):
				return !ok
			default:
			}
		case chan int:
			select {
			case _, ok := <-i.(chan int):
				return !ok
			default:
			}
		case chan interface{}:
			select {
			case _, ok := <-i.(chan interface{}):
				return !ok
			default:
			}
		}
	}
	return v.Interface() == reflect.Zero(v.Type()).Interface()
}

//* 判断是否是对应数据类型 */
func (brain *BrainS) CheckIsType(i interface{}, typ string) bool {
	if brain.Const.RunEnv != 0 {
		defer func() {
			if err := recover(); err != nil {
				brain.MessageHandler(brain.tag, "CheckIsType", 204, err)
			}
		}()
	}
	if i == nil {
		return false
	}
	v := reflect.ValueOf(i)
	if !v.IsValid() {
		return false
	}
	if v.Type().String() != typ {
		return false
	}
	return true
}

//* 通用数据类型转[]byte */
func (brain *BrainS) Interface2Bytes(i interface{}) []byte {
	if i == nil {
		return nil
	}
	reflectI := reflect.ValueOf(i)
	if !reflectI.IsValid() {
		return nil
	}
	switch reflectI.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return i.([]byte)
	default:
		return []byte(fmt.Sprintf("%v", i))
	}
}

//* MD5加密 */
func (brain *BrainS) MD5Encode(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}

//* Sha1加密 */
func (brain *BrainS) Sha1Encode(data []byte) string {
	return fmt.Sprintf("%x", sha1.Sum(data))
}

//* Sha256加密 */
func (brain *BrainS) HmacSha256Encode(data []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return fmt.Sprintf("%x", mac.Sum(nil))
}

//* BlowFish加密 */
func (brain *BrainS) BfEncode(data []byte) []byte {
	var result []byte
	brain.SafeFunction(func() {
		bfs := struct {
			key []byte
			iv  []byte
		}{
			key: brain.SystemKey(),
			iv:  make([]byte, 8),
		}
		c, err := blowfish.NewCipher(brain.SystemKey())
		if err != nil {
			brain.MessageHandler(brain.tag, "BfEncode", 209, err.Error())
			return
		}
		s := cipher.NewCFBEncrypter(c, bfs.iv)
		result = make([]byte, len(data))
		s.XORKeyStream(result, data)
	}, func(err interface{}) {
		if err == nil {
			return
		}
		brain.MessageHandler(brain.tag, "BfDecode", 204, err)
	})
	return result
}

//* BlowFish解密 */
func (brain *BrainS) BfDecode(data []byte) []byte {
	var result []byte
	brain.SafeFunction(func() {
		bfs := struct {
			key []byte
			iv  []byte
		}{
			key: brain.SystemKey(),
			iv:  make([]byte, 8),
		}
		c, err := blowfish.NewCipher(brain.SystemKey())
		if err != nil {
			brain.MessageHandler(brain.tag, "BfDecode", 209, err)
			return
		}
		s := cipher.NewCFBDecrypter(c, bfs.iv)
		result = make([]byte, len(data))
		s.XORKeyStream(result, data)
	})
	return result
}

//* Huffman Encoder */
func (brain *BrainS) HuffmanEncoder(in []byte, levels ...int) []byte {
	var buf bytes.Buffer
	level := -2
	if len(levels) > 0 {
		level = levels[0]
	}
	w, err := flate.NewWriter(&buf, level)
	if err != nil {
		brain.MessageHandler(brain.tag, "HuffmanEncoder", 219, err)
		return nil
	}
	w.Write(in)
	w.Close()
	return buf.Bytes()
}

//* Huffman Decoder */
func (brain *BrainS) HuffmanDecoder(in []byte) []byte {
	var out bytes.Buffer
	r := flate.NewReader(bytes.NewReader(in))
	io.Copy(&out, r)
	return out.Bytes()
}

//* Gzip Encoder */
func (brain *BrainS) GzipEncoder(in []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	_, err := writer.Write(in)
	if err != nil {
		return nil, err
	}
	writer.Close()
	return buffer.Bytes(), nil
}

//* Gzip Decoder */
func (brain *BrainS) GzipDecoder(in []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	reader.Close()
	return ioutil.ReadAll(reader)
}

//* Zlib Encoder */
func (brain *BrainS) ZlibEncoder(in []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, err := w.Write(in)
	if err != nil {
		return nil, err
	}
	w.Close()
	return buf.Bytes(), nil
}

//* Zlib Decoder */
func (brain *BrainS) ZlibDecoder(in []byte) ([]byte, error) {
	var out bytes.Buffer
	r, err := zlib.NewReader(bytes.NewReader(in))
	if err != nil {
		return nil, err
	}
	io.Copy(&out, r)
	return out.Bytes(), nil
}

//* 文字转Unicode */
func (brain *BrainS) String2Unicode(unicode string) string {
	var result string
	brain.SafeFunction(func() {
		textQuoted := strconv.QuoteToASCII(unicode)
		result = textQuoted[1 : len(textQuoted)-1]
	})
	return result
}

//* Unicode转文字 */
func (brain *BrainS) Unicode2String(str string) string {
	sUnicode := strings.Split(str, "\\u")
	var result string
	brain.SafeFunction(func() {
		var context bytes.Buffer
		for _, v := range sUnicode {
			if len(v) < 1 {
				continue
			}
			slice, err := strconv.ParseInt(v, 16, 32)
			if err != nil {
				brain.MessageHandler(brain.tag, "String2Unicode", 204, err)
			}
			context.WriteString(fmt.Sprintf("%c", slice))
		}
		result = context.String()
	})
	return result
}

//* 判断是否在两个百分数之间 */
func (brain *BrainS) BetweenPercent(price float64, average float64, percent float64) bool {
	priceD := decimal.NewFromFloat(price)
	averageD := decimal.NewFromFloat(average)
	percentD := decimal.NewFromFloat(percent)
	deltaD := priceD.Sub(averageD).Abs()
	if deltaD.Div(averageD).Cmp(percentD.Div(decimal.New(100, 32))) < 0 {
		return true
	} else {
		return false
	}
}

//* 合并对象 */
func (brain *BrainS) ObjectMerge(s ...[]interface{}) (slice []interface{}) {
	switch len(s) {
	case 0:
	case 1:
		slice = s[0]
	default:
		s1 := s[0]
		s2 := brain.ObjectMerge(s[1:]...)
		slice = make([]interface{}, len(s1)+len(s2))
		copy(slice, s1)
		copy(slice[len(s1):], s2)
	}
	return
}

//* 质朴长存法补0 */
func (brain *BrainS) NumPadZero(num int, n int) int {
	numLen := len(strconv.Itoa(num))
	numStr := strconv.Itoa(num)
	for numLen < n {
		numStr = "0" + numStr
		numLen++
	}
	ret, _ := strconv.Atoi(numStr)
	return ret
}

//* 结构体转Map */
func (brain *BrainS) Struct2Map(structS interface{}) map[string]interface{} {
	t := reflect.TypeOf(structS)
	v := reflect.ValueOf(structS)
	var data = make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		data[t.Field(i).Name] = v.Field(i).Interface()
	}
	return data
}

//* Map转结构体 */
func (brain *BrainS) Map2Struct(m interface{}, structS interface{}) interface{} {
	b := brain.JsonEncoder(m)
	if b == nil {
		brain.MessageHandler(brain.tag, "Map2Struct[JsonEncoder]", 202, m)
		return nil
	}
	return brain.JsonDecoder(b, structS)
}

//* 获取Map中的所有key */
func (brain *BrainS) Map2KeyArray(data map[string]interface{}) []string {
	returnVal := make([]string, 0, len(data))
	for k := range data {
		returnVal = append(returnVal, k)
	}
	return returnVal
}

//* Base64 */
func (brain *BrainS) Base64Encoder(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
func (brain *BrainS) Base64Decoder(data string) []byte {
	resultByte, _ := base64.StdEncoding.DecodeString(data)
	return resultByte
}

//* Json -> CheckIsRight */
func (brain *BrainS) JsonChecker(data []byte) bool {
	err := json.Unmarshal(data, new(map[string]interface{}))
	if err == nil {
		return true
	}
	return false
}

//* Json -> String2Object */
func (brain *BrainS) JsonDecoder(data []byte, structS ...interface{}) interface{} {
	// Try Decode Struct
	if len(structS) > 0 {
		err := json.Unmarshal(data, structS[0])
		if err == nil {
			return structS[0]
		} else {
			brain.MessageHandler(brain.tag, fmt.Sprintf("JsonDecoder[Struct] -> %s", structS[0]), 202, err)
		}
	}
	// Try Decode MapObject
	var dataObjMap map[string]interface{}
	err := json.Unmarshal(data, &dataObjMap)
	if err == nil {
		return dataObjMap
	} else {
		brain.MessageHandler(brain.tag, fmt.Sprintf("JsonDecoder[Map] -> %s", data), 202, err)
	}
	// Try Decode ArrayObject
	var dataObjArray []interface{}
	err = json.Unmarshal(data, &dataObjArray)
	if err == nil {
		return dataObjArray
	} else {
		brain.MessageHandler(brain.tag, fmt.Sprintf("JsonDecoder[Array] -> %s", data), 202, err)
	}
	return nil
}

//* Json -> Object2String */
func (brain *BrainS) JsonEncoder(data interface{}, indent ...bool) []byte {
	switcher := false
	if !brain.CheckIsNull(indent) {
		switcher = indent[0]
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(data)
	if err != nil {
		brain.MessageHandler(brain.tag, "JsonEncoder", 202, err)
		return nil
	}
	if switcher {
		var dstBuf bytes.Buffer
		err := json.Indent(&dstBuf, buf.Bytes(), "", "    ")
		if err != nil {
			brain.MessageHandler(brain.tag, "JsonEncoder", 202, err)
			return nil
		}
		return dstBuf.Bytes()
	}
	return buf.Bytes()
}

//* XML -> String2Object */
func (brain *BrainS) XMLDecoder(data []byte, dataType interface{}) interface{} {
	err := xml.Unmarshal(data, dataType)
	if err == nil {
		return dataType
	}
	brain.MessageHandler(brain.tag, "XMLDecoder", 202, err)
	return nil
}

//* XML -> Object2String */
func (brain *BrainS) XMLEncoder(data interface{}, indent ...bool) []byte {
	switcher := false
	if !brain.CheckIsNull(indent) {
		switcher = indent[0]
	}
	if switcher {
		xdata, err := xml.MarshalIndent(data, "", "    ")
		if err != nil {
			brain.MessageHandler(brain.tag, "XMLEncoder", 202, err)
			return nil
		}
		return xdata
	} else {
		xdata, err := xml.Marshal(data)
		if err != nil {
			brain.MessageHandler(brain.tag, "XMLEncoder", 202, err)
			return nil
		}
		return xdata
	}
}

//* HEX转化 */
func (brain *BrainS) HEXEncoder(b []byte) string {
	return strings.ToUpper(hex.EncodeToString(b))
}

func (brain *BrainS) HEXDecoder(s string) []byte {
	b, _ := hex.DecodeString(s)
	return b
}

//* 获取格式化时间 */
/**
* param:
*   datetime => similar to '2012-12-12 00:00:00'[string]
*   dateformat => just like '/' & '-'
*   timeformat => just like ':'
* return:
*   object => [
*     YMD  => yyyy-MM-dd[string]
*     week => weekday[string]
*     time => HH:mm:ss[string]
*     timestamp => timestamp[timestamp]
*     datetime => datetime[datetime]
*   ]
 */
func (brain *BrainS) GetDateTime(datetime string, changeDuration ...time.Duration) *model.TimeS {
	timeS := new(model.TimeS)
	dateSplit := "-"
	timeSplit := ":"

	timeResult := time.Now()
	// 时区纠正
	loc, _ := time.LoadLocation("Local")
	if datetime != "now" {
		// 去TZ
		if strings.Contains(datetime, "T") {
			datetime = strings.ReplaceAll(datetime, "T", " ")
			datetime = strings.ReplaceAll(datetime, ".000Z", "")
		}
		if strings.Contains(datetime, timeSplit) {
			timeParse, err := time.ParseInLocation("2006"+dateSplit+"01"+dateSplit+"02 15"+timeSplit+"04"+timeSplit+"05", datetime, loc)
			if err != nil {
				return nil
			}
			timeResult = timeParse
		} else {
			timestamp, _ := strconv.Atoi(datetime)
			timeResult = time.Unix(int64(timestamp), 0)
		}
	}

	if len(changeDuration) > 0 {
		timeResult = timeResult.Add(changeDuration[0])
	}
	timeS.YMD = timeResult.Format("2006" + dateSplit + "01" + dateSplit + "02")
	timeS.Week = timeResult.Weekday()
	timeS.Time = timeResult.Format("15" + timeSplit + "04" + timeSplit + "05")
	timeS.Timestamp = fmt.Sprintf("%d", timeResult.Unix())
	timeS.TimestampMill = fmt.Sprintf("%d", timeResult.UnixNano()/int64(time.Millisecond))
	timeS.TimestampNano = fmt.Sprintf("%d", timeResult.UnixNano())
	timeS.Datetime = timeResult

	return timeS
}

//* 判断现在是否到达特定时间 */
/**
* param:
*   timestamp_check => 时间戳[timestamp]
*   interval => 间隔时间[int]
* return:
*   boolean => [
*     true => arrived
*     flase => not arrived
*   ]
 */
func (brain *BrainS) CheckTimeArrived(datetime string, interval int) bool {
	timestamp1, _ := strconv.Atoi(brain.GetDateTime("now").TimestampMill)
	timestamp2, _ := strconv.Atoi(brain.GetDateTime(datetime).TimestampMill)
	timeDiff := timestamp1 - timestamp2
	if interval == 0 {
		if 0 < timeDiff {
			return true
		} else {
			return false
		}
	} else {
		if 0 < timeDiff && timeDiff <= interval {
			return true
		} else {
			return false
		}
	}
}

//* 时间戳随机生成正数 */
func (brain *BrainS) RandomInt(max int, seeds ...int64) int {
	seed := time.Now().UnixNano()
	if len(seeds) > 0 {
		seed += seeds[0]
	}
	r := rand.New(rand.NewSource(seed))
	return r.Intn(max + 1)
}

//* 时间戳随机生成字符串 */
func (brain *BrainS) RandomChar(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)
	src := rand.NewSource(time.Now().UnixNano())
	b := make([]byte, n)
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}

//* Emoji表情反序列化 */
func (brain *BrainS) EmojiUnicodeDecode(s string) string {
	//emoji表情的数据表达式
	re := regexp.MustCompile("\\[[\\\\u0-9a-zA-Z]+\\]")
	//提取emoji数据表达式
	reg := regexp.MustCompile("\\[\\\\u|]")
	src := re.FindAllString(s, -1)
	for i := 0; i < len(src); i++ {
		e := reg.ReplaceAllString(src[i], "")
		p, err := strconv.ParseInt(e, 16, 32)
		if err == nil {
			s = strings.Replace(s, src[i], string(rune(p)), -1)
		}
	}
	return s
}

//* Emoji表情序列化 */
func (brain *BrainS) EmojiUnicodeEncode(s string) string {
	ret := ""
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		if len(string(rs[i])) == 4 {
			u := `[\u` + strconv.FormatInt(int64(rs[i]), 16) + `]`
			ret += u

		} else {
			ret += string(rs[i])
		}
	}
	return ret
}

//* Emoji表情清除 */
func (brain *BrainS) EmojiUnicodeClear(s string) string {
	ret := ""
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		if len(string(rs[i])) == 4 {
			u := ""
			ret += u
		} else {
			ret += string(rs[i])
		}
	}
	return ret
}

//* 获取局域网内ip地址 -> ipRoute为IP查询段 */
func (brain *BrainS) GetLanIp(ipRoute ...string) (int, interface{}) {
	// 单网卡模式
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return 204, fmt.Sprintf("[Local Network Error] -> %v", err)
	}
	for _, v := range addrs {
		if ip, ok := v.(*net.IPNet); ok && !ip.IP.IsLoopback() && !ip.IP.IsMulticast() {
			if ip.IP.To4() != nil {
				ipAddr := fmt.Sprintf("%s", ip.IP)
				if len(ipRoute) > 0 {
					if ipAddr[:len(ipRoute[0])] == ipRoute[0] {
						return 100, ipAddr
					}
				} else {
					return 100, ipAddr
				}
				continue
			}
		}
	}
	return 200, nil
}

//* 获取数组内容下标 */
func (brain *BrainS) ArrayGetId(array []interface{}, data interface{}) int {
	for k, v := range array {
		if v == data {
			return k
		}
	}
	return -1
}

//* 获取对应表格行下标内容 */
func (brain *BrainS) TableGetData(column []string, data interface{}, directArray []interface{}) interface{} {
	if len(column) != len(directArray) {
		return nil
	}
	columnI := make([]interface{}, len(column))
	for i := 0; i < len(column); i++ {
		columnI[i] = column[i]
	}
	id := brain.ArrayGetId(columnI, data)
	if id == -1 {
		return nil
	}
	return directArray[id]
}

//* 返回bool类型的int对应 */
func (brain *BrainS) Bool2Int(b bool) int {
	if b {
		return 1
	} else {
		return 0
	}
}

//* 获取CRC校验 */
func (brain *BrainS) CRC(b []byte) byte {
	var checksum byte
	for _, v := range b {
		checksum += v
	}
	return ^checksum + 1
}

//* 生成新的Key对 */
func (brain *BrainS) GenerateKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey) {
	privkey, err := rsa.GenerateKey(randc.Reader, bits)
	if err != nil {
		brain.LogGenerater(model.LogError, brain.tag, "GenerateKeyPair", err)
	}
	return privkey, &privkey.PublicKey
}

//* PrivateKey流转化 */
func (brain *BrainS) PrivateKeyToBytes(priv *rsa.PrivateKey) []byte {
	privBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		},
	)
	return privBytes
}
func (brain *BrainS) BytesToPrivateKey(priv []byte) *rsa.PrivateKey {
	block, _ := pem.Decode(priv)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	var err error
	if enc {
		brain.LogGenerater(model.LogInfo, brain.tag, "BytesToPrivateKey[enc]", "Is encrypted pem block")
		b, err = x509.DecryptPEMBlock(block, nil)
		if err != nil {
			brain.LogGenerater(model.LogError, brain.tag, "BytesToPrivateKey[DecryptPEMBlock]", err)
		}
	}
	key, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		brain.LogGenerater(model.LogError, brain.tag, "BytesToPrivateKey[ParsePKCS1PrivateKey]", err)
	}
	return key
}

//* PublicKey流转化 */
func (brain *BrainS) PublicKeyToBytes(pub *rsa.PublicKey) []byte {
	pubASN1, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		brain.LogGenerater(model.LogError, brain.tag, "PublicKeyToBytes", err)
	}
	pubBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubASN1,
	})
	return pubBytes
}
func (brain *BrainS) BytesToPublicKey(pub []byte) *rsa.PublicKey {
	block, _ := pem.Decode(pub)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	var err error
	if enc {
		brain.LogGenerater(model.LogInfo, brain.tag, "BytesToPublicKey[enc]", "Is encrypted pem block")
		b, err = x509.DecryptPEMBlock(block, nil)
		if err != nil {
			brain.LogGenerater(model.LogError, brain.tag, "BytesToPublicKey[DecryptPEMBlock]", err)
		}
	}
	ifc, err := x509.ParsePKIXPublicKey(b)
	if err != nil {
		brain.LogGenerater(model.LogError, brain.tag, "BytesToPublicKey[ParsePKIXPublicKey]", err)
	}
	key, ok := ifc.(*rsa.PublicKey)
	if !ok {
		brain.LogGenerater(model.LogError, brain.tag, "BytesToPublicKey[ifc]", err)
	}
	return key
}

//* 通过PrivateKey加密 */
func (brain *BrainS) EncryptWithPublicKey(msg []byte, pub *rsa.PublicKey) []byte {
	hash := sha512.New()
	ciphertext, err := rsa.EncryptOAEP(hash, randc.Reader, pub, msg, nil)
	if err != nil {
		brain.LogGenerater(model.LogError, brain.tag, "EncryptWithPublicKey[EncryptOAEP]", err)
	}
	return ciphertext
}

//* 通过PrivateKey解密 */
func (brain *BrainS) DecryptWithPrivateKey(ciphertext []byte, priv *rsa.PrivateKey) []byte {
	hash := sha512.New()
	plaintext, err := rsa.DecryptOAEP(hash, randc.Reader, priv, ciphertext, nil)
	if err != nil {
		brain.LogGenerater(model.LogError, brain.tag, "DecryptWithPrivateKey[DecryptOAEP]", err)
	}
	return plaintext
}

//* 反射获取方法名 */
func (brain *BrainS) GetFuncName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

//* 展示对象内容 */
func (brain *BrainS) ShowObject(obj interface{}) {
	fmt.Printf("%s\n", brain.JsonEncoder(obj, true))
}

//* 获取黄金分割点 */
func (brain *BrainS) GoldPoint(price float64) (float64, float64) {
	priceD := decimal.NewFromFloat(price)
	upper, _ := priceD.Mul(decimal.NewFromFloat(1.0618)).Float64()
	lower, _ := priceD.Mul(decimal.NewFromFloat(0.9382)).Float64()
	return upper, lower
}

//* 选择迭代器 */
func (brain *BrainS) SelectIterator(hubLen int, lastIndex int) int {
	lastIndex++
	if lastIndex >= hubLen {
		lastIndex = 0
	}
	return lastIndex
}

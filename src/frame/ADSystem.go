/**
===========================================================================
 * 系统服务
 * System Service
===========================================================================
*/

package frame

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"model"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"
)

//* ================================ DEFINE ================================ */
type SystemS struct {
	Const struct {
		tag  string
		root string
	}
	Container   struct{}
	Connection  struct{}
	StopChannel struct {
		clearLogLooperSC  chan bool
	}
	isStarted bool
	neuron    *NeuronS
	mux       *http.ServeMux
}

//* ================================ PRIVATE ================================ */

//* 注册服务 */
func (mSystem *SystemS) main() {
	// Interface
	mSystem.configInterface()
	mSystem.uploadInterface()
}

//* ================================ INTERFACE ================================ */

//* 远程配置接口 */
func (mSystem *SystemS) configInterface() {
	mSystem.mux.HandleFunc(mSystem.Const.root+"/Config", func(res http.ResponseWriter, req *http.Request) {
		mSystem.neuron.Express.ConstructInterface(res, req, mSystem.isStarted, func() {
			query := mSystem.neuron.Express.Req2Query(req)
			for k := range query {
				switch k {
				case "ReadFile":
					// 读取配置文件
					code, data := mSystem.neuron.Brain.FileReader(mSystem.neuron.Brain.PathAbs("/config.json"))
					switch code {
					case 100:
						mSystem.neuron.Express.CodeResponse(res, code, mSystem.neuron.Brain.JsonDecoder(data.([]byte)))
					default:
						mSystem.neuron.Express.CodeResponse(res, code, data)
					}
				case "ReadConst":
					// 读取全部配置
					mSystem.neuron.Express.CodeResponse(res, 100, mSystem.neuron.Brain.Const)
				case "WriteFile":
					resBody, err := ioutil.ReadAll(req.Body)
					if err != nil {
						mSystem.neuron.Express.CodeResponse(res, 207, err)
						return
					}
					if mSystem.neuron.Brain.JsonChecker(resBody) {
						mSystem.neuron.Brain.FileWriter(mSystem.neuron.Brain.PathAbs("/config.json"), resBody)
					}
					mSystem.ConfigInit()
					mSystem.neuron.Express.CodeResponse(res, 100, mSystem.neuron.Brain.Const)
				default:
					mSystem.neuron.Express.CodeResponse(res, 207, "Param Error", "configInterface")
				}
				return
			}
		}, func(err interface{}) {
			mSystem.neuron.Express.CodeResponse(res, 204, err, "configInterface[ConstructInterface]")
		})
	})
}

//* 远程上传接口 */
func (mSystem *SystemS) uploadInterface() {
	mSystem.mux.HandleFunc(mSystem.Const.root+"/Upload", func(res http.ResponseWriter, req *http.Request) {
		mSystem.neuron.Express.ConstructInterface(res, req, mSystem.isStarted, func() {
			query := mSystem.neuron.Express.Req2Query(req)
			// 特殊上传模式
			if !mSystem.neuron.Brain.CheckIsNull(query) {
				for k := range query {
					switch k {
					case "AUTORUN":
						// 删除临时文件
						mSystem.neuron.Brain.FileRemovAll(mSystem.neuron.Brain.PathAbs(fmt.Sprintf("%v/avatar", mSystem.neuron.Brain.Const.HTTPServer.UploadPath)))
						code, data := mSystem.systemUpdate(res, req)
						mSystem.neuron.Brain.FileRemovAll(mSystem.neuron.Brain.PathAbs(fmt.Sprintf("%v/avatar", mSystem.neuron.Brain.Const.HTTPServer.UploadPath)))
						mSystem.neuron.Express.CodeResponse(res, code, data)
						return
					}
				}
			}
			// 固定上传模式
			code, data := mSystem.uploadFile(res, req)
			switch code {
			case 100:
				mSystem.neuron.Express.CodeResponse(res, code, fmt.Sprintf("Uploaded Files -> %v", data))
			default:
				mSystem.neuron.Express.CodeResponse(res, code, data)
			}
		}, func(err interface{}) {
			mSystem.neuron.Express.CodeResponse(res, 204, err, "uploadInterface[ConstructInterface]")
		})
	})
}

//* ================================ PROCESS ================================ */

//* 远程上传接口 */
func (mSystem *SystemS) uploadFile(res http.ResponseWriter, req *http.Request) (int, interface{}) {
	if mSystem.neuron.Brain.Const.RunEnv < 2 {
		mSystem.neuron.Brain.LogGenerater(model.LogTrace, mSystem.Const.tag, "uploadFile", fmt.Sprintf("Request -> %+v", req.Header))
	}
	// 获取Multipart信息
	if err := req.ParseMultipartForm(64 << 20); err != nil {
		mSystem.neuron.Express.CodeResponse(res, 216, err)
	}
	fileNameArr := make([]string, 0, 10)
	if mSystem.neuron.Brain.CheckIsNull(req.MultipartForm.File) {
		return 220, fmt.Sprintf("uploadFile[MultipartForm.File] -> Null")
	}
	for k := range req.MultipartForm.File {
		uploadPath := mSystem.neuron.Brain.SystemSplit(k)[0]
		file, header, err := req.FormFile(k)
		if err != nil {
			return 216, fmt.Sprintf("uploadFile[FormFile] -> %v", err)
		}
		if mSystem.neuron.Brain.Const.RunEnv < 2 {
			mSystem.neuron.Brain.LogGenerater(model.LogTrace, mSystem.Const.tag, "uploadFile", fmt.Sprintf("Header -> %+v", header.Header))
		}
		// 若果没有文件信息则排除
		if mSystem.neuron.Brain.CheckIsNull(header.Filename) {
			continue
		}
		fileBuffer, err := ioutil.ReadAll(file)
		if err != nil {
			return 216, fmt.Sprintf("uploadFile[ReadAll] -> %v", err)
		}
		// 判断是否压缩
		fileDecode := fileBuffer
		encodeFlag := req.Header.Get("Content-Encoding")
		if encodeFlag == "gzip" {
			fileDecode, err = mSystem.neuron.Brain.GzipDecoder(fileBuffer)
			if err != nil {
				mSystem.neuron.Brain.MessageHandler(mSystem.Const.tag, "GzipDecode", 219, err)
			}
		}
		// 写入文件
		fileNameArr = append(fileNameArr, fmt.Sprintf("/%s/%s", uploadPath, header.Filename))
		code, data := mSystem.neuron.Brain.FileWriter(fmt.Sprintf("%s/%s/%s", mSystem.neuron.Brain.PathAbs(mSystem.neuron.Brain.Const.HTTPServer.UploadPath), uploadPath, header.Filename), fileDecode)
		if code != 100 {
			mSystem.neuron.Brain.MessageHandler(mSystem.Const.tag, "uploadFile[FileWriter]", code, data)
		}
		file.Close()
	}
	return 100, fileNameArr
}

//* 远程更新接口 */
func (mSystem *SystemS) systemUpdate(res http.ResponseWriter, req *http.Request) (int, interface{}) {
	// 获取上传文件
	code, data := mSystem.uploadFile(res, req)
	if code != 100 {
		return code, data
	}
	filename := data.([]string)[0]
	filePasswd := req.Header.Get("passwd")
	path, base, ext := mSystem.neuron.Brain.PathAbs2PathBaseExt(filename)
	pathAbs := mSystem.neuron.Brain.PathAbs(mSystem.neuron.Brain.Const.HTTPServer.UploadPath + path)
	if ext != ".zip" {
		return 221, fmt.Sprintf("FileExt -> %v%v", base, ext)
	}
	// 解压文件
	code, data = mSystem.neuron.Brain.SystemExec(func(cmd *exec.Cmd) (int, interface{}) {
		data, err := cmd.CombinedOutput()
		if err != nil {
			return 218, string(data)
		}
		return 100, string(data)
	}, map[string]string{
		"dir":  pathAbs,
		"exec": "unzip",
	}, "-uqP", string(mSystem.neuron.Brain.SystemDecrypt([]byte(filePasswd))), fmt.Sprintf("./%v", base))
	if code != 100 {
		return code, fmt.Sprintf("Unzip Failed -> %v", data)
	}
	// 执行脚本
	code, data = mSystem.neuron.Brain.SystemExec(func(cmd *exec.Cmd) (int, interface{}) {
		data, err := cmd.CombinedOutput()
		if err != nil {
			return 218, string(data)
		}
		return 100, string(data)
	}, map[string]string{
		"dir":  pathAbs,
		"exec": "bash",
	}, "-c", "./autorun.sh")
	if code != 100 {
		return code, fmt.Sprintf("Autorun Failed -> %v", data)
	}
	return code, data
}

//* ================================ SQL PROCESS ================================ */

//* ================================ TOOL ================================ */

//* LOG下划线日期文件名转日期格式 */
func (mSystem *SystemS) splitFilename2Time(filename string) *model.TimeS {
	filenameSlice := strings.Split(filename, "_")
	var buf bytes.Buffer
	if len(filenameSlice) != 6 {
		return nil
	}
	buf.WriteString(filenameSlice[0])
	buf.WriteString("-")
	buf.WriteString(filenameSlice[1])
	buf.WriteString("-")
	buf.WriteString(filenameSlice[2])
	buf.WriteString(" ")
	buf.WriteString(filenameSlice[3])
	buf.WriteString(":")
	buf.WriteString(filenameSlice[4])
	buf.WriteString(":")
	buf.WriteString(filenameSlice[5])
	return mSystem.neuron.Brain.GetDateTime(buf.String())
}

//* ================================ LOOPER & RECEIVER ================================ */

//* 按月清理LOG记录 */
func (mSystem *SystemS) clearLogLooper() {
	// 运行中则避免重复
	if !mSystem.neuron.Brain.CheckIsNull(mSystem.StopChannel.clearLogLooperSC) {
		return
	}
	// 开启循环 -> 2小时
	mSystem.StopChannel.clearLogLooperSC = make(chan bool)
	go mSystem.neuron.Brain.SetInterval(func() (int, interface{}) {
		// 获取一个月前的时间
		dt := mSystem.neuron.Brain.GetDateTime("now", time.Duration(-30*24*time.Hour))
		// 获取文件列表
		code, data := mSystem.neuron.Brain.PathList(mSystem.neuron.Brain.PathAbs("/log"))
		if code != 100 {
			mSystem.neuron.Brain.MessageHandler(mSystem.Const.tag, "clearLogLooper", code, data)
			return 100, nil
		}
		// 循环获取所有LOG文件夹
		for _, v := range data.([]os.FileInfo) {
			if !v.IsDir() {
				continue
			}
			// 获取文件列表
			code, data := mSystem.neuron.Brain.PathList(mSystem.neuron.Brain.PathAbs("/log/" + v.Name()))
			if code != 100 {
				mSystem.neuron.Brain.MessageHandler(mSystem.Const.tag, "clearLogLooper", code, data)
				continue
			}
			// 循环获取所有LOG文件
			for _, vv := range data.([]os.FileInfo) {
				if vv.IsDir() {
					continue
				}
				_, filename, _ := mSystem.neuron.Brain.PathAbs2PathBaseExt(vv.Name())
				// 格式化带日期时间的文件名
				filenameDT := mSystem.splitFilename2Time(filename)
				if mSystem.neuron.Brain.CheckIsNull(filenameDT) {
					continue
				}
				fileNameTS, err := strconv.Atoi(filenameDT.Timestamp)
				if err != nil {
					continue
				}
				dtTS, err := strconv.Atoi(dt.Timestamp)
				if err != nil {
					continue
				}
				if fileNameTS == 0 || dtTS == 0 {
					continue
				}
				// 文件名时间戳小于目标时间戳的LOG文件全部删除
				if fileNameTS < dtTS {
					mSystem.neuron.Brain.FileRemove(mSystem.neuron.Brain.PathAbs(fmt.Sprintf("/log/%v/%v", v.Name(), vv.Name())))
				}
			}
		}
		return 100, nil
	}, func(code int, data interface{}) {
		if code != 100 {
			mSystem.neuron.Brain.MessageHandler(mSystem.Const.tag, "clearLogLooper[SetInterval]", code, data)
		}
	}, mSystem.neuron.Brain.Const.Interval.TwoHourInterval, mSystem.StopChannel.clearLogLooperSC)
}

//* 析构按月清理LOG记录 */
func (mSystem *SystemS) clearLogLooperKiller() {
	mSystem.neuron.Brain.ClearInterval(mSystem.StopChannel.clearLogLooperSC)
}

//* ================================ SERVICE ================================ */

//* 构造服务 */
func (mSystem *SystemS) service() {
	mSystem.clearLogLooper()
}

//* 析构服务 */
func (mSystem *SystemS) serviceKiller() {
	mSystem.clearLogLooperKiller()
}

//* ================================ PUBLIC ================================ */

//* 构造本体 */
func (mSystem *SystemS) Ontology(neuron *NeuronS, mux *http.ServeMux, root string) *SystemS {
	mSystem.neuron = neuron
	mSystem.mux = mux
	mSystem.Const.tag = root[1:]
	mSystem.Const.root = root
	mSystem.neuron.Brain.SafeFunction(mSystem.main)
	mSystem.StartService()
	return mSystem
}

//* 返回开关量 */
func (mSystem *SystemS) IsStarted() bool {
	return mSystem.isStarted
}

//* 启动服务 */
func (mSystem *SystemS) StartService() {
	if mSystem.isStarted {
		return
	}
	mSystem.isStarted = true
	go mSystem.neuron.Brain.SafeFunction(mSystem.service)
}

//* 停止服务 */
func (mSystem *SystemS) StopService() {
	// 系统服务不可停止
	return
	if !mSystem.isStarted {
		return
	}
	mSystem.isStarted = false
	go mSystem.neuron.Brain.SafeFunction(mSystem.serviceKiller)
}

//* 打印信息 */
func (mSystem *SystemS) Log(title string, content ...interface{}) {
	if title == mSystem.Const.tag {
		mSystem.neuron.Brain.LogGenerater(model.LogTrace, mSystem.Const.tag, title, fmt.Sprintf("%+v", mSystem))
	} else if title == mSystem.neuron.Brain.tag {
		mSystem.neuron.Brain.LogGenerater(model.LogTrace, mSystem.Const.tag, title, fmt.Sprintf("%+v", mSystem.neuron.Brain))
	} else {
		mSystem.neuron.Brain.LogGenerater(model.LogInfo, mSystem.Const.tag, title, content)
	}
}

//* ================================ RPC INTERFACE ================================ */

//* ================================ 开发环境方法 ================================ */

//* 显示数据库默认Token */
func (mSystem *SystemS) DBToken() {
	brain := mSystem.neuron.Brain
	if brain.Const.RunEnv != 0 {
		return
	}
	if !mSystem.isStarted {
		return
	}
	mSystem.Log("DBToken", mSystem.neuron.Mysql.DefaultDBToken)
}

//* 使用系统方法加密 */
func (mSystem *SystemS) Encrypt(sha1Key, str string) {
	brain := mSystem.neuron.Brain
	if brain.Const.RunEnv != 0 {
		return
	}
	if !mSystem.isStarted {
		return
	}
	sha1SystemKey := mSystem.neuron.Brain.Sha1Encode(mSystem.neuron.Brain.SystemKey())
	if sha1SystemKey != sha1Key {
		return
	}
	result := mSystem.neuron.Brain.SystemEncrypt([]byte(str))
	mSystem.Log("Encrypt", string(mSystem.neuron.Brain.Base64Encoder(result)))
}

//* 使用系统方法解密 */
func (mSystem *SystemS) Decrypt(sha1Key, b64 string) {
	brain := mSystem.neuron.Brain
	if brain.Const.RunEnv != 0 {
		return
	}
	if !mSystem.isStarted {
		return
	}
	sha1SystemKey := mSystem.neuron.Brain.Sha1Encode(mSystem.neuron.Brain.SystemKey())
	if sha1SystemKey != sha1Key {
		return
	}
	result := mSystem.neuron.Brain.Base64Decoder(b64)
	mSystem.Log("Decrypt", string(mSystem.neuron.Brain.SystemDecrypt(result)))
}

//* ================================ 公共环境方法 ================================ */

//* 显示所有系统方法 */
func (mSystem *SystemS) Funclist() {
	if !mSystem.isStarted {
		return
	}
	t := reflect.TypeOf(mSystem)
	v := reflect.ValueOf(mSystem)
	for i := 0; i < v.NumMethod(); i++ {
		f := t.Method(i)
		if f.Name == "Ontology" || f.Name == "IsStarted" || f.Name == "StartService" || f.Name == "StopService" || f.Name == "FuncList" {
			continue
		}
		mSystem.Log("FuncList", fmt.Sprintf("%v (%v)", f.Name, f.Func.String()))
	}
}

//* 重新读取配置文件 */
func (mSystem *SystemS) ConfigInit() {
	if !mSystem.isStarted {
		return
	}
	mSystem.neuron.initConfig()
}

//* Sha1加密 */
func (mSystem *SystemS) Sha1Encode(s string) {
	if !mSystem.isStarted {
		return
	}
	mSystem.Log("Sha1Encode", mSystem.neuron.Brain.Sha1Encode([]byte(s)))
}
package frame

import (
	"bytes"
	"fmt"
	"model"
	"os/exec"
	"syscall"
)

//* 启动一个操作系统服务 */
func (brain *BrainS) SystemService(stopC chan bool, osCommand map[string]string, params ...string) (int, interface{}) {
	defer close(stopC)
	var codeR int
	var dataR interface{}
	// Runnable
	brain.SafeFunction(func() {
		codeR, dataR = brain.SystemExec(func(cmd *exec.Cmd) (int, interface{}) {
			// 配置运行参数
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			var buf bytes.Buffer
			cmd.Stdout = &buf
			cmd.Stderr = &buf
			go func() {
				for {
					if data := <-stopC; data {
						cmd.Process.Kill()
						brain.LogGenerater(model.LogWarn, brain.tag, "SystemService", fmt.Sprintf("[Cancel] -> %s %v", brain.systemSelect(osCommand), params))
						return
					} else {
						return
					}
				}
			}()
			err := cmd.Run()
			if err != nil {
				return 218, err
			}
			return 100, buf.Bytes()
		}, osCommand, params...)
	}, func(err interface{}) {
		if err == nil {
			return
		}
		codeR = 204
		dataR = fmt.Sprintf("[Exec]System Error -> %s\n", err)
	})
	return codeR, dataR
}

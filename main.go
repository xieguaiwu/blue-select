// main.go
package main

import (
	"errors"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "connect":
		name := ""
		if len(os.Args) > 2 {
			name = os.Args[2]
		}
		err = runConnect(name)
	case "status":
		err = runStatus()
	case "watch":
		err = runWatch()
	case "install":
		err = runInstall()
	case "-h", "--help", "help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "blue-select:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stdout, `blue-select — 蓝牙耳机一键连接与音频切换

用法:
  blue-select connect [设备名]   连接设备并切换音频输出（无参用默认设备）
  blue-select status             显示连接、电量、profile、输出状态
  blue-select watch              监听新蓝牙 sink，自动设默认并迁移播放流
  blue-select install            安装并启用 watch 的 systemd user 服务
`)
}

// 以下为桩实现，Task 4-7 逐个替换。

func runConnect(name string) error { return errors.New("connect: not implemented") }
func runStatus() error             { return errors.New("status: not implemented") }
func runWatch() error              { return errors.New("watch: not implemented") }
func runInstall() error            { return errors.New("install: not implemented") }

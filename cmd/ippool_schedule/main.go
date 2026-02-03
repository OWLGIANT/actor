package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"actor/schedule"
	"actor/third/log"
)

func main() {
	// 初始化日志
	log.Init("ippool_schedule.log", "info",
		log.SetStdout(true),
		log.SetCaller(true),
		log.SetMaxBackups(3),
	)

	fmt.Println("启动 IP池定时任务调度器...")
	log.Info("启动 IP池定时任务调度器")

	// 初始化定时任务调度器
	// 会立即执行一次 IP池生成，然后每2小时自动执行一次
	schedule.InitSchedule()

	fmt.Println("IP池定时任务调度器已启动，每2小时自动更新一次")
	fmt.Println("按 Ctrl+C 退出")

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 保持程序运行
	<-sigChan

	fmt.Println("\n收到退出信号，正在关闭...")
	log.Info("收到退出信号，正在关闭")
	time.Sleep(time.Second)
	fmt.Println("程序已退出")
}

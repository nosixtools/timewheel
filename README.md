# timewheel

Golang实现的时间轮

[![Go Report Card](https://goreportcard.com/badge/github.com/nosixtools/timewheel)](https://goreportcard.com/report/github.com/nosixtools/timewheel)

![时间轮](https://raw.githubusercontent.com/nosixtools/timewheel/master/timewheel.jpg)


# 安装

```shell
go get -u github.com/nosixtools/timewheel
```

# 使用

```
package main

import (
	"fmt"
	"github.com/nosixtools/timewheel"
	"time"
)

func main() {

        //初始化时间轮盘
        //参数：interval 时间间隔
        //参数：slotNum  轮盘大小
	tw := timewheel.New(time.Second, 160)

	tw.Start()
	
        //添加定时任务
        //参数：interval 时间间隔
        //参数：times 执行次数 -1 表示周期任务 >0 执行指定次数
        //参数：key 任务唯一标识符 用户更新任务和删除任务
        //参数：taskData 回调函数参数
        //参数：job 回调函数
	tw.AddTask(time.Second*2, 5, "did", timewheel.TaskData{"name": "nosixtools"}, func(params timewheel.TaskData) {
		fmt.Println(time.Now().Unix(), params["name"])
	})
	
	//删除定时任务
	tw.RemoveTask("did")
	
	//轮盘停止
	//tw.Stop()

	select {}
}


```
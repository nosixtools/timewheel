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
g
import (
	"fmt"
	"github.com/nosixtools/timewheel"
	"time"
)

func main() {
	tw := timewheel.New(time.Second, 160)

	tw.Start()

	tw.AddTask(time.Second*2, 5, "did", timewheel.TaskData{"name": "nosixtools"}, func(params timewheel.TaskData) {
		fmt.Println(time.Now().Unix(), params["name"])
	})

	select {}
}


```
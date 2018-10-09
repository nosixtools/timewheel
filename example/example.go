package main

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

package timewheel_multi

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Job callback function
type Job func(TaskData)

// TaskData callback params
type TaskData map[interface{}]interface{}

type TimeWheel struct {
	secondWheel    *wheel
	minuteWheel    *wheel
	hourWheel      *wheel
	ticker         *time.Ticker
	addTaskChannel chan *task
	stopChannel    chan bool
	taskRecord     map[interface{}]*task
	taskRecordLock sync.RWMutex
}

type wheel struct {
	id         int
	interval   time.Duration
	slots      []*list.List
	currentPos int
	slotNum    int
}

// task struct
type task struct {
	interval     time.Duration
	remainSecond int
	times        int //-1:no limit >=1:run times
	wheel        int
	key          interface{}
	job          Job
	taskData     TaskData
}

// New create a new time wheel
func New() *TimeWheel {

	tw := &TimeWheel{
		secondWheel: &wheel{id: 0,
			interval:   time.Second,
			currentPos: 0,
			slotNum:    60,
			slots:      make([]*list.List, 60)},

		minuteWheel: &wheel{id: 0,
			interval:   time.Minute,
			currentPos: 0,
			slotNum:    60,
			slots:      make([]*list.List, 60)},

		hourWheel: &wheel{id: 0,
			interval:   time.Hour,
			currentPos: 0,
			slotNum:    24,
			slots:      make([]*list.List, 24)},
		addTaskChannel: make(chan *task),
		stopChannel:    make(chan bool),
		taskRecord:     make(map[interface{}]*task),
	}
	tw.init()
	return tw
}

// time wheel initialize
func (tw *TimeWheel) init() {
	for i := 0; i < tw.secondWheel.slotNum; i++ {
		tw.secondWheel.slots[i] = list.New()
	}

	for i := 0; i < tw.minuteWheel.slotNum; i++ {
		tw.minuteWheel.slots[i] = list.New()
	}

	for i := 0; i < tw.hourWheel.slotNum; i++ {
		tw.hourWheel.slots[i] = list.New()
	}
}

func (tw *TimeWheel) Start() {
	tw.ticker = time.NewTicker(time.Second)
	go tw.start()
}

func (tw *TimeWheel) AddTask(interval time.Duration, times int, key interface{}, data TaskData, job Job) error {
	if interval <= 0 || key == nil || job == nil || times < -1 || times == 0 {
		return errors.New("illegal task params")
	}
	tw.taskRecordLock.RLock()
	defer tw.taskRecordLock.RUnlock()
	if tw.taskRecord[key] != nil {
		return errors.New("duplicate task key")
	}
	tw.addTaskChannel <- &task{interval: interval, times: times, key: key, taskData: data, job: job}
	return nil
}

func (tw *TimeWheel) start() {
	for {
		select {
		case <-tw.ticker.C:
			tw.secondWheelTickHandler()
		case task := <-tw.addTaskChannel:
			tw.addTask(task)
		case <-tw.stopChannel:
			tw.ticker.Stop()
			return
		}
	}
}

func (tw *TimeWheel) secondWheelTickHandler() {
	l := tw.secondWheel.slots[tw.secondWheel.currentPos]
	tw.scanAddRunTask(l)
	if tw.secondWheel.currentPos == tw.secondWheel.slotNum-1 {
		tw.secondWheel.currentPos = 0
		tw.minuteWheelTickHandler()
	} else {
		tw.secondWheel.currentPos++
	}
}

func (tw *TimeWheel) scanAddRunTask(l *list.List) {
	if l == nil {
		return
	}
	for item := l.Front(); item != nil; {
		task := item.Value.(*task)

		if task.times == 0 {
			next := item.Next()
			l.Remove(item)
			item = next
		}

		go task.job(task.taskData)
		next := item.Next()
		l.Remove(item)
		item = next

		if task.times > 0 || task.times == - 1 {
			if task.times > 0 {
				task.times--
			}
			tw.addTask(task)
		} else {
			tw.taskRecordLock.Lock()
			delete(tw.taskRecord, task.key)
			tw.taskRecordLock.Unlock()
		}

	}
}

func (tw *TimeWheel) minuteWheelTickHandler() {
	l := tw.minuteWheel.slots[tw.minuteWheel.currentPos]

	for item := l.Front(); item != nil; {
		task := item.Value.(*task)

		// clean delete task
		if task.times == 0 {
			next := item.Next()
			l.Remove(item)
			item = next
			tw.taskRecordLock.Lock()
			delete(tw.taskRecord, task.key)
			tw.taskRecordLock.Unlock()
			continue
		}
		next := item.Next()
		l.Remove(item)
		item = next
		tw.addTaskToWheel(task)
	}

	if tw.minuteWheel.currentPos == tw.minuteWheel.slotNum-1 {
		tw.minuteWheel.currentPos = 0
		tw.hourWheelTickHandler()
	} else {
		tw.minuteWheel.currentPos++
	}
}

func (tw *TimeWheel) hourWheelTickHandler() {
	l := tw.hourWheel.slots[tw.hourWheel.currentPos]
	fmt.Println("minute")
	for item := l.Front(); item != nil; {
		task := item.Value.(*task)
		// clean delete task
		if task.times == 0 {
			next := item.Next()
			l.Remove(item)
			item = next
			tw.taskRecordLock.Lock()
			delete(tw.taskRecord, task.key)
			tw.taskRecordLock.Unlock()
			continue
		}
		next := item.Next()
		l.Remove(item)
		item = next
		tw.addTaskToWheel(task)
	}

	if tw.hourWheel.currentPos == tw.hourWheel.slotNum-1 {
		tw.hourWheel.currentPos = 0
	} else {
		tw.hourWheel.currentPos ++
	}

}

func (tw *TimeWheel) addTaskToWheel(task *task) {
	wheelId, pos := tw.getPositionAndWheel(task)
	fmt.Println(wheelId, pos)
	switch wheelId {
	case 0:
		tw.secondWheel.slots[pos].PushBack(task)
	case 1:
		tw.minuteWheel.slots[pos].PushBack(task)
	case 2:
		tw.hourWheel.slots[pos].PushBack(task)
	}
}

func (tw *TimeWheel) addTask(task *task) {
	if task.remainSecond == 0 {
		task.remainSecond = int(task.interval.Seconds())
	}
	tw.addTaskToWheel(task)
	tw.taskRecordLock.Lock()
	defer tw.taskRecordLock.Unlock()
	tw.taskRecord[task.key] = task
}

func (tw *TimeWheel) RemoveTask(key interface{}) error {
	if key == nil {
		return nil
	}

	tw.taskRecordLock.RLock()
	defer tw.taskRecordLock.RUnlock()

	task := tw.taskRecord[key]
	if task == nil {
		return errors.New("task not exists, please check you task key")
	} else {
		task.times = 0
		delete(tw.taskRecord, key)
	}
	return nil
}

func (tw *TimeWheel) getPositionAndWheel(task *task) (int, int) {
	delaySeconds := task.remainSecond
	if delaySeconds >= int(time.Hour.Seconds()) {
		offset := delaySeconds / 3600 - 1
		second := delaySeconds % 3600
		task.remainSecond = second
		return 2, (tw.hourWheel.currentPos + offset) % tw.hourWheel.slotNum
	} else if delaySeconds >= int(time.Minute.Seconds()) {
		offset := delaySeconds / 60 - 1
		second := delaySeconds % 60
		task.remainSecond = second
		return 1, (tw.minuteWheel.currentPos + offset) % tw.minuteWheel.slotNum
	} else {
		offset := delaySeconds
		task.remainSecond = 0
		return 0, (tw.secondWheel.currentPos + offset) % tw.secondWheel.slotNum
	}
}

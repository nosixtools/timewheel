package timewheel

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

// time wheel struct
type TimeWheel struct {
	interval          time.Duration
	ticker            *time.Ticker
	slots             []*list.List
	timer             map[interface{}]int
	timerLock         sync.Mutex
	currentPos        int
	slotNum           int
	addTaskChannel    chan *task
	removeTaskChannel chan interface{}
	stopChannel       chan bool
	taskRecord        map[interface{}]*task
	recordLock        sync.Mutex
}

// Job callback function
type Job func(TaskData)

// TaskData callback params
type TaskData map[interface{}]interface{}

// task struct
type task struct {
	interval time.Duration
	times    int //-1:no limit >=1:run times
	circle   int
	key      interface{}
	job      Job
	taskData TaskData
}

// New create a empty time wheel
func New(interval time.Duration, slotNum int) *TimeWheel {
	if interval <= 0 || slotNum <= 0 {
		return nil
	}
	tw := &TimeWheel{
		interval:          interval,
		slots:             make([]*list.List, slotNum),
		timer:             make(map[interface{}]int),
		currentPos:        0,
		slotNum:           slotNum,
		addTaskChannel:    make(chan *task),
		removeTaskChannel: make(chan interface{}),
		stopChannel:       make(chan bool),
		taskRecord:        make(map[interface{}]*task),
	}

	tw.init()

	return tw
}

// Start start the time wheel
func (tw *TimeWheel) Start() {
	tw.ticker = time.NewTicker(tw.interval)
	go tw.start()
}

// Stop stop the time wheel
func (tw *TimeWheel) Stop() {
	tw.stopChannel <- true
}

func (tw *TimeWheel) start() {
	for {
		select {
		case <-tw.ticker.C:
			tw.tickHandler()
		case task := <-tw.addTaskChannel:
			tw.addTask(task)
		case key := <-tw.removeTaskChannel:
			tw.removeTask(key)
		case <-tw.stopChannel:
			tw.ticker.Stop()
			return
		}
	}
}

// AddTask add new task to the time wheel
func (tw *TimeWheel) AddTask(interval time.Duration, times int, key interface{}, data TaskData, job Job) error {
	if interval <= 0 || key == nil || job == nil || times < -1 || times == 0 {
		return errors.New("illegal task params")
	}
	if tw.taskRecord[key] != nil {
		return errors.New("duplicate task key")
	}
	tw.addTaskChannel <- &task{interval: interval, times: times, key: key, taskData: data, job: job}
	return nil
}

// RemoveTask remove the task from time wheel
func (tw *TimeWheel) RemoveTask(key interface{}) error {
	if key == nil {
		return nil
	}
	if tw.taskRecord[key] == nil {
		return errors.New("task not exists, please check you task key")
	}
	tw.removeTaskChannel <- key
	return nil
}

// UpdateTask update task times and data
func (tw *TimeWheel) UpdateTask(key interface{}, interval time.Duration, taskData TaskData) error {
	if key == nil {
		return errors.New("illegal key, please try again")
	}

	task := tw.taskRecord[key]
	if task == nil {
		return errors.New("task not exists, please check you task key")
	}

	task.taskData = taskData
	task.interval = interval
	return nil
}

// time wheel initialize
func (tw *TimeWheel) init() {
	for i := 0; i < tw.slotNum; i++ {
		tw.slots[i] = list.New()
	}
}

//
func (tw *TimeWheel) tickHandler() {
	l := tw.slots[tw.currentPos]
	tw.scanAddRunTask(l)
	if tw.currentPos == tw.slotNum-1 {
		tw.currentPos = 0
	} else {
		tw.currentPos++
	}
}

// add task
func (tw *TimeWheel) addTask(task *task) {

	pos, circle := tw.getPositionAndCircle(task.interval)
	task.circle = circle

	tw.slots[pos].PushBack(task)

	tw.timerLock.Lock()
	defer tw.timerLock.Unlock()
	tw.timer[task.key] = pos

	//record the task
	tw.recordLock.Lock()
	defer tw.recordLock.Unlock()
	tw.taskRecord[task.key] = task

}

// remove task
func (tw *TimeWheel) removeTask(key interface{}) {

	position, ok := tw.timer[key]
	if !ok {
		return
	}
	l := tw.slots[position]
	for e := l.Front(); e != nil; {
		task := e.Value.(*task)
		if task.key == key {
			tw.timerLock.Unlock()
			delete(tw.timer, task.key)
			tw.timerLock.Unlock()
			l.Remove(e)
		}
		e = e.Next()
	}

	//remove from task record
	tw.recordLock.Lock()
	delete(tw.taskRecord, key)
	tw.recordLock.Unlock()
}

// scan task list and run the task
func (tw *TimeWheel) scanAddRunTask(l *list.List) {

	if l == nil {
		return
	}

	for item := l.Front(); item != nil; {
		task := item.Value.(*task)

		if task.times == 0 {
			next := item.Next()
			l.Remove(item)
			tw.timerLock.Unlock()
			delete(tw.timer, task.key)
			tw.timerLock.Unlock()
			item = next
			continue
		}

		if task.circle > 0 {
			task.circle--
			item = item.Next()
			continue
		}

		go task.job(task.taskData)
		next := item.Next()
		l.Remove(item)
		item = next
		if task.times > 0 || task.times == -1 {
			if task.times > 0 {
				task.times--
			}
			tw.addTask(task)
		} else {
			tw.timerLock.Unlock()
			delete(tw.timer, task.key)
			tw.timerLock.Unlock()
		}
	}
}

// get the task position
func (tw *TimeWheel) getPositionAndCircle(d time.Duration) (pos int, circle int) {
	delaySeconds := int(d.Seconds())
	intervalSeconds := int(tw.interval.Seconds())
	circle = int(delaySeconds / intervalSeconds / tw.slotNum)
	pos = int(tw.currentPos+delaySeconds/intervalSeconds) % tw.slotNum
	return
}

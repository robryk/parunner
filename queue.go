package main

type MessageQueue struct {
	outputCh chan interface{}
	inputCh  chan interface{}
	selector chan *MessageQueue
}

func (mq *MessageQueue) Put(m interface{}) {
	mq.inputCh <- m
}

func (mq *MessageQueue) Get() interface{} {
	return <-mq.outputCh
}

func (mq *MessageQueue) Shutdown() {
	close(mq.inputCh)
}

func (mq *MessageQueue) loop() {
	buffer := make([]interface{}, 0, 10)
	for {
		var selector chan *MessageQueue
		var outputChan chan interface{}
		var outputValue interface{}
		if len(buffer) > 0 {
			selector = mq.selector
			outputChan = mq.outputCh
			outputValue = buffer[0]
		}
		select {
		case value, ok := <-mq.inputCh:
			if !ok {
				return
			}
			buffer = append(buffer, value)
		case outputChan <- outputValue:
			buffer = buffer[1:]
		case selector <- mq:
		}
	}
}

func NewMessageQueue(selector chan *MessageQueue) *MessageQueue {
	mq := &MessageQueue{
		outputCh: make(chan interface{}),
		inputCh:  make(chan interface{}, 1),
		selector: selector,
	}
	go mq.loop()
	return mq
}

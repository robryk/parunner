package main

import (
	"time"
)

type fakeInstance struct {
	requestChan  chan *request
	responseChan chan *response
	fakeTime     time.Duration
}

func newFakeInstance() *fakeInstance {
	return &fakeInstance{
		requestChan:  make(chan *request, 1),
		responseChan: make(chan *response, 1),
	}
}

func (fi *fakeInstance) Send(destination int, contents []byte) {
	fi.fakeTime++
	fi.requestChan <- &request{
		requestType: requestSend,
		time:        fi.fakeTime,
		destination: destination,
		message:     contents,
	}
}

func (fi *fakeInstance) RecvFrom(source int) *Message {
	fi.fakeTime++
	fi.requestChan <- &request{
		requestType: requestRecv,
		time:        fi.fakeTime,
		source:      source,
	}
	resp := <-fi.responseChan
	if resp.message.SendTime > fi.fakeTime {
		fi.fakeTime = resp.message.SendTime
	}
	return resp.message
}

func (fi *fakeInstance) Recv() *Message {
	fi.fakeTime++
	fi.requestChan <- &request{
		requestType: requestRecvAny,
		time:        fi.fakeTime,
	}
	resp := <-fi.responseChan
	if resp.message.SendTime > fi.fakeTime {
		fi.fakeTime = resp.message.SendTime
	}
	return resp.message
}

func (fi *fakeInstance) Close() {
	close(fi.requestChan)
}

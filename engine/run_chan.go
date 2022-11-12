package engine

import (
	"sync"
)

type responseChan struct {
	response chan responseData
	isClose  bool
	lock     sync.RWMutex
}

func (r *responseChan) Close() {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.isClose {
		return
	}
	r.isClose = true
	close(r.response)
}

func (r *responseChan) Set(data responseData) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.isClose {
		return
	}
	r.response <- data
}

func (r *responseChan) Get() (responseData, bool) {

	if r.isClose {
		return responseData{}, false
	}
	data, is := <-r.response
	return data, is
}

func (r *responseChan) IsClose() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.isClose
}

type errorChan struct {
	err     chan error
	isClose bool
	lock    sync.RWMutex
}

func (r *errorChan) Close() {

	r.lock.Lock()
	defer r.lock.Unlock()
	if r.isClose {
		return
	}
	r.isClose = true
	close(r.err)
}

func (r *errorChan) Set(data error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.isClose {
		return
	}
	r.err <- data
}

func (r *errorChan) Get() (error, bool) {
	if r.isClose {
		return nil, false
	}
	err, is := <-r.err
	return err, is
}

func (r *errorChan) IsClose() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.isClose
}

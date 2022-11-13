package engine

import (
	"sync"
)

type responseChan struct {
	response chan map[string]any
	isClose  bool
	lock     sync.RWMutex
}

// SetAndClose 只接收一次返回信息信息
func (r *responseChan) SetAndClose(data map[string]any) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.isClose {
		return
	}
	r.response <- data
	r.isClose = true
	close(r.response)
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

func (r *responseChan) IsClose() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.isClose
}

func (r *responseChan) Get() (map[string]any, bool) {
	if r.isClose {
		return nil, false
	}
	data, is := <-r.response
	return data, is
}

type errorChan struct {
	err     chan error
	isClose bool
	lock    sync.RWMutex
}

// SetAndClose 只接收一次错误中断信息
func (r *errorChan) SetAndClose(data error, wg *sync.WaitGroup) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.isClose {
		wg.Done()
		return
	}
	r.isClose = true
	r.err <- data
	close(r.err)
}

// Close 关闭通道
func (r *errorChan) Close() {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.isClose {
		return
	}
	r.isClose = true
	close(r.err)
}

func (r *errorChan) Get() (error, bool) {
	if r.isClose {
		return nil, false
	}
	err, is := <-r.err
	return err, is
}

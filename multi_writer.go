/*
-------------------------------------------------
   Author :       zlyuancn
   date：         2020/10/3
   Description :
-------------------------------------------------
*/

package zrunner

import (
	"io"
	"sync"
)

type MultiWriter struct {
	ws   []io.Writer
	sync bool
	mx   *sync.Mutex
}

func NewMultiWriter(w ...io.Writer) *MultiWriter {
	return &MultiWriter{
		ws: append([]io.Writer{}, w...),
	}
}
func (m *MultiWriter) OnSync() *MultiWriter {
	m.sync = true
	return m
}
func (m *MultiWriter) Write(p []byte) (n int, err error) {
	if m.sync {
		m.mx.Lock()
		defer m.mx.Unlock()
	}

	for _, w := range m.ws {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
		if n != len(p) {
			return n, io.ErrShortWrite
		}
	}
	return len(p), err
}

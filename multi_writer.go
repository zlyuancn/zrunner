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

	"golang.org/x/sync/errgroup"
)

type MultiWriteCloser struct {
	ws   []io.WriteCloser
	sync bool
	mx   *sync.Mutex
}

func NewMultiWriteCloser(w ...io.WriteCloser) *MultiWriteCloser {
	return &MultiWriteCloser{
		ws: append([]io.WriteCloser{}, w...),
	}
}
func (m *MultiWriteCloser) OnSync() *MultiWriteCloser {
	m.sync = true
	return m
}
func (m *MultiWriteCloser) Write(p []byte) (n int, err error) {
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
func (m *MultiWriteCloser) Close() error {
	g := errgroup.Group{}
	for _, w := range m.ws {
		g.Go(w.Close)
	}
	return g.Wait()
}

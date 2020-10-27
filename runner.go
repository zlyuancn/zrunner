/*
-------------------------------------------------
   Author :       zlyuancn
   date：         2020/10/3
   Description :
-------------------------------------------------
*/

package zrunner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// 运行状态
type RunState int32

const (
	// 已停止
	StoppedState RunState = iota
	// 已启动
	StartedState
)

type RunnerConfig struct {
	Dir     string
	Command string
	Args    []string
	Env     []string

	StdIn io.Reader

	Stdout               io.Writer
	StdoutFile           string // 日志文件路径
	StdoutFileMaxSize    int    // 每个日志文件保存的最大尺寸 单位：M
	StdoutFileMaxBackups int    // 文件最多保存多少天
	StdoutFileMaxAge     int    // 日志文件最多保存多少个备份

	Stderr               io.Writer
	RedirectStderr       bool // 重定向err输出到stdout
	StderrFile           string
	StderrFileMaxSize    int
	StderrFileMaxBackups int
	StderrFileMaxAge     int

	User string // 以哪个用户启动
}

type Runner struct {
	RunnerConfig

	stdoutFileWriter io.WriteCloser
	stderrFileWriter io.WriteCloser

	runState RunState // 运行状态 0=关闭, 1=运行

	cmd  *exec.Cmd
	done chan error

	mx sync.Mutex
}

func NewExec(conf *RunnerConfig) *Runner {
	r := &Runner{
		RunnerConfig: *conf,
	}
	if r.Dir == "" {
		r.Dir, _ = os.Getwd()
	}

	return r
}

func (r *Runner) makeLogFile() error {
	var ws []io.Writer
	if r.Stdout != nil {
		ws = append(ws, r.Stdout)
	}
	if r.StdoutFile != "" {
		w, err := r.makeWriter(r.cmd.Dir, r.StdoutFile, r.StdoutFileMaxSize, r.StdoutFileMaxAge, r.StdoutFileMaxBackups)
		if err != nil {
			return err
		}
		r.stdoutFileWriter = w
		ws = append(ws, w)
	}
	if len(ws) > 0 {
		r.cmd.Stdout = NewMultiWriter(ws...)
	}

	if r.RedirectStderr {
		r.cmd.Stderr = r.cmd.Stdout
		return nil
	}

	ws = nil
	if r.Stderr != nil {
		ws = append(ws, r.Stderr)
	}
	if r.StderrFile != "" {
		w, err := r.makeWriter(r.cmd.Dir, r.StderrFile, r.StderrFileMaxSize, r.StderrFileMaxAge, r.StderrFileMaxBackups)
		if err != nil {
			return err
		}
		r.stderrFileWriter = w
		ws = append(ws, w)
	}
	if len(ws) > 0 {
		r.cmd.Stderr = NewMultiWriter(ws...)
	}
	return nil
}
func (r *Runner) makeWriter(dir, filename string, maxSize, maxAge, maxBackups int) (io.WriteCloser, error) {
	if !path.IsAbs(filename) {
		filename = path.Join(dir, filename)
	}

	err := os.MkdirAll(path.Dir(filename), 666)
	if err != nil {
		return nil, fmt.Errorf("无法创建日志目录: <%s>: %s\n", path.Dir(filename), err)
	}

	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSize,
		MaxAge:     maxAge,
		MaxBackups: maxBackups,
		LocalTime:  true,
	}, nil
}

func (r *Runner) Run() error {
	if err := r.Start(); err != nil {
		return err
	}
	return r.Wait()
}

func (r *Runner) Start() error {
	r.mx.Lock()
	defer r.mx.Unlock()

	if r.runState != StoppedState {
		return nil
	}

	r.cmd = exec.Command(r.Command, r.Args...)
	r.cmd.Dir = r.Dir
	r.cmd.Env = append([]string{}, r.Env...)
	r.cmd.Stdin = r.StdIn

	if r.User != "" {
		if err := r.makeExecUser(); err != nil {
			return err
		}
	}

	if err := r.makeLogFile(); err != nil {
		r.closeLogWriter()
		return err
	}

	err := r.cmd.Start()
	if err != nil {
		r.closeLogWriter()
		return err
	}

	r.done = make(chan error, 1)
	r.runState = StartedState
	go r.wait()
	return nil
}

func (r *Runner) wait() {
	err := r.cmd.Wait()

	r.mx.Lock()
	defer r.mx.Unlock()

	r.closeLogWriter()
	r.cmd = nil

	r.runState = StoppedState
	r.done <- err
}

func (r *Runner) closeLogWriter() {
	if r.stdoutFileWriter != nil {
		_ = r.stdoutFileWriter.Close()
		r.stdoutFileWriter = nil
	}
	if r.stderrFileWriter != nil {
		_ = r.stderrFileWriter.Close()
		r.stdoutFileWriter = nil
	}
}

func (r *Runner) Wait() error {
	r.mx.Lock()
	if r.runState == StoppedState {
		r.mx.Unlock()
		return nil
	}
	done := r.done
	r.mx.Unlock()

	return <-done
}

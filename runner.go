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
	"os/user"
	"path"
	"strconv"
	"syscall"

	"gopkg.in/natefinch/lumberjack.v2"
)

type RunnerConfig struct {
	Dir     string
	Command string
	Args    []string
	Env     []string

	Stdout               io.WriteCloser
	StdoutFile           string // 日志文件路径
	StdoutFileMaxSize    int    // 每个日志文件保存的最大尺寸 单位：M
	StdoutFileMaxBackups int    // 文件最多保存多少天
	StdoutFileMaxAge     int    // 日志文件最多保存多少个备份

	Stderr               io.WriteCloser
	RedirectStderr       bool // 重定向err输出到stdout
	StderrFile           string
	StderrFileMaxSize    int
	StderrFileMaxBackups int
	StderrFileMaxAge     int

	User string // 以哪个用户启动
}

type Runner struct {
	RunnerConfig
	stdout io.WriteCloser
	stderr io.WriteCloser
	cmd    *exec.Cmd
}

func NewExec(conf *RunnerConfig) (*Runner, error) {
	r := &Runner{
		RunnerConfig: *conf,
	}
	if r.Dir == "" {
		r.Dir, _ = os.Getwd()
	}

	r.cmd = exec.Command(r.Command, r.Args...)
	r.cmd.Dir = r.Dir
	r.cmd.Env = append([]string{}, r.Env...)

	if err := r.makeLogFile(); err != nil {
		return nil, err
	}

	if r.User != "" {
		if err := r.makeExecUser(); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Runner) makeLogFile() error {
	var ws []io.WriteCloser
	if r.Stdout != nil {
		ws = append(ws, r.Stdout)
	}
	if r.StdoutFile != "" {
		w, err := r.makeWriter(r.cmd.Dir, r.StdoutFile, r.StdoutFileMaxSize, r.StdoutFileMaxAge, r.StdoutFileMaxBackups)
		if err != nil {
			return err
		}
		ws = append(ws, w)
	}
	if len(ws) > 0 {
		r.stdout = NewMultiWriteCloser(ws...)
		r.cmd.Stdout = r.stdout
	}

	if r.RedirectStderr {
		r.stderr = r.stdout
		r.cmd.Stderr = r.stderr
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
		ws = append(ws, w)
	}
	if len(ws) > 0 {
		r.stderr = NewMultiWriteCloser(ws...)
		r.cmd.Stderr = r.stderr
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
func (r *Runner) makeExecUser() error {
	u, err := user.Lookup(r.User)
	if err != nil {
		return fmt.Errorf("无法获取用户id[%s], err: %s", r.User, err)
	}

	uid, _ := strconv.ParseUint(u.Uid, 10, 32)
	gid, _ := strconv.ParseUint(u.Gid, 10, 32)
	r.cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid:         uint32(uid),
			Gid:         uint32(gid),
			NoSetGroups: true,
		},
	}
	return nil
}

func (r *Runner) Run() error {
	if err := r.Start(); err != nil {
		return err
	}
	return r.Wait()
}
func (r *Runner) Start() error {
	return r.cmd.Start()
}
func (r *Runner) Wait() error {
	if err := r.cmd.Wait(); err != nil {
		return err
	}

	if r.stdout != nil {
		_ = r.stdout.Close()
	}
	if r.stderr != nil {
		_ = r.stderr.Close()
	}
	return nil
}

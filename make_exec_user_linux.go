/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/10/27
   Description :
-------------------------------------------------
*/

package zrunner

import (
	"fmt"
	"os/user"
	"strconv"
	"syscall"
)

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
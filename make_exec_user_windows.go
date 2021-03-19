/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/10/27
   Description :
-------------------------------------------------
*/

package zrunner

import (
	"errors"
)

func (r *Runner) makeExecUser() error {
	return errors.New("在windows下不能设置用户")
}

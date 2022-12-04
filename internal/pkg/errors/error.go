package ex

import (
	"github.com/lovechung/api-base/api/user"
)

var (
	UserNotFound = user.ErrorUserNotFound(user.UserServiceErrorReason_USER_IS_FREEZE.String() + "|该用户不存在")
	UserIsFreeze = user.ErrorUserIsFreeze(user.UserServiceErrorReason_USER_IS_FREEZE.String() + "|该用户已冻结，请联系管理员")
)

package ex

import "github.com/lovechung/api-base/api/user"

var (
	UserNotFound = user.ErrorUserNotFound("该用户不存在")
)

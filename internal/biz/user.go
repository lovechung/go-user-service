package biz

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"time"
)

type User struct {
	Id        int64
	Username  *string
	Password  *string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

type UserRepo interface {
	ListUser(ctx context.Context, page, pageSize int, username *string) ([]*User, int, error)
	GetById(ctx context.Context, id int64) (*User, error)
	GetUsername(ctx context.Context, id int64) (string, error)
	GetUsernameBatch(ctx context.Context, ids []int64) (map[int64]string, error)
	Save(context.Context, *User) (int64, error)
	Update(context.Context, *User) error
	Delete(ctx context.Context, id int64) error
}

type UserUseCase struct {
	r   UserRepo
	log *log.Helper
	tx  Transaction
}

func NewUserUseCase(r UserRepo, tx Transaction, logger log.Logger) *UserUseCase {
	return &UserUseCase{r: r, tx: tx, log: log.NewHelper(logger)}
}

func (uc *UserUseCase) ListUser(ctx context.Context, page, pageSize int, username *string) ([]*User, int, error) {
	return uc.r.ListUser(ctx, page, pageSize, username)
}

func (uc *UserUseCase) GetUserById(ctx context.Context, id int64) (*User, error) {
	return uc.r.GetById(ctx, id)
}

func (uc *UserUseCase) GetUsername(ctx context.Context, id int64) (string, error) {
	return uc.r.GetUsername(ctx, id)
}

func (uc *UserUseCase) GetUsernameBatch(ctx context.Context, ids []int64) (map[int64]string, error) {
	return uc.r.GetUsernameBatch(ctx, ids)
}

func (uc *UserUseCase) SaveUser(ctx context.Context, u *User) error {
	id, err := uc.r.Save(ctx, u)
	// 打印一条普通（非trace）日志
	uc.log.Infof("新增的用户id=%d", id)
	return err
}

func (uc *UserUseCase) UpdateUser(ctx context.Context, u *User) error {
	// 带有事务的操作
	if e := uc.tx.ExecTx(ctx, func(ctx context.Context) error {
		return uc.r.Update(ctx, u)
	}); e != nil {
		return e
	}
	return nil
}

func (uc *UserUseCase) DeleteUser(ctx context.Context, id int64) error {
	return uc.r.Delete(ctx, id)
}

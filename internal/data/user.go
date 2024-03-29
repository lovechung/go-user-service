package data

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/lovechung/go-kit/util/pagination"
	"time"
	"user-service/internal/biz"
	"user-service/internal/data/ent"
	"user-service/internal/data/ent/predicate"
	"user-service/internal/data/ent/user"
	ex "user-service/internal/pkg/errors"
)

type userRepo struct {
	data *Data
	log  *log.Helper
}

func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

var userCacheKey = func(userId string) string {
	return "user:info:" + userId
}

func (r userRepo) ListUser(ctx context.Context, page, pageSize int, username *string) ([]*biz.User, int, error) {
	// 该方法演示单表分页多条件查询
	q := r.data.db.User.Query()

	// 组装查询条件
	cond := make([]predicate.User, 0)
	if username != nil {
		cond = append(cond, user.UsernameContains(*username))
	}
	if len(cond) > 0 {
		q.Where(cond...)
	}

	// 查询总数
	total := q.CountX(ctx)

	// 查询列表
	users := q.Offset(pagination.GetOffset(page, pageSize)).
		Limit(pageSize).
		Order(ent.Desc(user.FieldCreatedAt)).
		AllX(ctx)

	// 组装返回参数
	list := make([]*biz.User, 0)
	for _, u := range users {
		list = append(list, ConvertToUser(u))
	}
	return list, total, nil
}

func (r userRepo) GetById(ctx context.Context, id int64) (*biz.User, error) {
	// 先从缓存中取
	cacheKey := userCacheKey(fmt.Sprintf("%d", id))
	u, err := r.getUserCache(ctx, cacheKey)

	if err != nil {
		// 缓存没有命中，则从数据库取
		u, err = r.data.db.User.Get(ctx, id)
		if err != nil {
			return nil, ex.UserNotFound
		}
		// 重新刷入缓存
		r.setUserCache(ctx, u, cacheKey)
	}
	return ConvertToUser(u), err
}

func (r userRepo) GetUsername(ctx context.Context, id int64) (string, error) {
	u, err := r.data.db.User.
		Query().
		Select(user.FieldUsername).
		Where(user.ID(id)).
		Only(ctx)
	if err != nil {
		return "", ex.UserNotFound
	}

	// 模拟一个异常
	if id == 1 {
		return "", ex.UserIsFreeze
	}

	return u.Username, nil
}

func (r userRepo) GetUsernameBatch(ctx context.Context, ids []int64) (map[int64]string, error) {
	users, err := r.data.db.User.
		Query().
		Select(user.FieldID, user.FieldUsername).
		Where(user.IDIn(ids...)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	nameMap := make(map[int64]string)
	for _, u := range users {
		nameMap[u.ID] = u.Username
	}
	return nameMap, nil
}

func (r userRepo) Save(ctx context.Context, u *biz.User) (int64, error) {
	rsp, err := r.data.db.User.
		Create().
		SetUser(u).
		SetCreatedAt(time.Now()).
		Save(ctx)
	return rsp.ID, err
}

func (r userRepo) Update(ctx context.Context, u *biz.User) error {
	// 带有事务的操作
	err := r.data.User(ctx).
		Update().
		Where(user.ID(u.Id)).
		SetUser(u).
		SetUpdatedAt(time.Now()).
		Exec(ctx)
	// 模拟一个异常
	if u.Password != nil && *u.Password == "123456" {
		err = ex.UserNotFound
	}
	return err
}

func (r userRepo) Delete(ctx context.Context, id int64) error {
	return r.data.db.User.
		DeleteOneID(id).
		Exec(ctx)
}

func ConvertToUser(u *ent.User) *biz.User {
	return &biz.User{
		Id:        u.ID,
		Username:  &u.Username,
		Password:  &u.Password,
		CreatedAt: &u.CreatedAt,
		UpdatedAt: &u.UpdatedAt,
	}
}

func (r *userRepo) getUserCache(ctx context.Context, key string) (*ent.User, error) {
	cacheUser := &ent.User{}
	err := r.data.rds.Do(ctx, r.data.rds.B().JsonGet().Key(key).Paths(".").Build()).DecodeJSON(cacheUser)
	if err != nil {
		return nil, err
	}
	return cacheUser, nil
}

func (r *userRepo) setUserCache(ctx context.Context, user *ent.User, key string) {
	val, _ := json.Marshal(user)
	r.data.rds.Do(ctx, r.data.rds.B().JsonSet().Key(key).Path(".").Value(string(val)).Build())
	r.data.rds.Do(ctx, r.data.rds.B().Expire().Key(key).Seconds(600).Build())
}

package data

import (
	"context"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"fmt"
	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	consulApi "github.com/hashicorp/consul/api"
	"github.com/rueian/rueidis"
	"github.com/rueian/rueidis/rueidiscompat"
	"github.com/rueian/rueidis/rueidisotel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"strings"
	"user-service/internal/biz"
	"user-service/internal/conf"
	"user-service/internal/data/ent"
)

var ProviderSet = wire.NewSet(
	NewTransaction,
	NewData,
	NewDB,
	NewRedis,
	NewRegistrar,
	NewUserRepo,
)

type Data struct {
	db     *ent.Client
	rds    rueidis.Client
	rdsCmd rueidiscompat.Cmdable
}

type contextTxKey struct{}

func (d *Data) ExecTx(ctx context.Context, f func(ctx context.Context) error) error {
	tx, err := d.db.Tx(ctx)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, contextTxKey{}, tx)
	if err := f(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (d *Data) User(ctx context.Context) *ent.UserClient {
	tx, ok := ctx.Value(contextTxKey{}).(*ent.Tx)
	if ok {
		return tx.User
	}
	return d.db.User
}

func NewTransaction(d *Data) biz.Transaction {
	return d
}

func NewData(db *ent.Client, rds rueidis.Client, logger log.Logger) (*Data, func(), error) {
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		if err := db.Close(); err != nil {
			log.Error(err)
		}
		rds.Close()
	}

	return &Data{
		db:     db,
		rds:    rds,
		rdsCmd: rueidiscompat.NewAdapter(rds),
	}, cleanup, nil
}

func NewDB(conf *conf.Data, logger log.Logger) *ent.Client {
	thisLog := log.NewHelper(logger)

	drv, err := sql.Open(
		conf.Database.Driver,
		conf.Database.Source,
	)
	// 打印sql日志
	sqlDrv := dialect.DebugWithContext(drv, func(ctx context.Context, i ...interface{}) {
		thisLog.WithContext(ctx).Debug(i...)
		// 开启db trace
		tracer := otel.Tracer("entgo.io/ent")
		// 组装sql信息
		fullSql := fmt.Sprint(i...)
		args := strings.Index(fullSql, "args=")
		_, span := tracer.Start(ctx,
			fullSql[strings.Index(fullSql, ".")+1:strings.Index(fullSql, ":")],
			trace.WithAttributes(
				attribute.String("sql", fullSql[strings.Index(fullSql, "=")+1:args]),
				attribute.String("sql.args", fullSql[args+6:len(fullSql)-1]),
			),
		)
		defer span.End()
	})
	db := ent.NewClient(ent.Driver(sqlDrv))

	if err != nil {
		thisLog.Fatalf("数据库连接失败: %v", err)
	}
	// 运行自动创建表
	//if err := db.Schema.Create(context.Background(), migrate.WithForeignKeys(false)); err != nil {
	//	thisLog.Fatalf("创建表失败: %v", err)
	//}
	return db
}

func NewRedis(conf *conf.Data, logger log.Logger) rueidis.Client {
	thisLog := log.NewHelper(logger)

	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:      []string{conf.Redis.Addr},
		Password:         conf.Redis.Password,
		SelectDB:         int(conf.Redis.Db),
		ConnWriteTimeout: conf.Redis.WriteTimeout.AsDuration(),
	})
	client = rueidisotel.WithClient(client)

	if err != nil {
		thisLog.Fatalf("redis连接失败: %v", err)
	}
	return client
}

func NewRegistrar(conf *conf.Registry) registry.Registrar {
	c := consulApi.DefaultConfig()
	c.Address = conf.Consul.Address
	c.Scheme = conf.Consul.Scheme
	c.Token = conf.Consul.Token
	cli, err := consulApi.NewClient(c)
	if err != nil {
		panic(err)
	}
	r := consul.New(cli, consul.WithHealthCheck(conf.Consul.HealthCheck))
	return r
}

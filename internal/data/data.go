package data

import (
	"context"
	"database/sql"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/XSAM/otelsql"
	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	consulApi "github.com/hashicorp/consul/api"
	"github.com/rueian/rueidis"
	"github.com/rueian/rueidis/rueidiscompat"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
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

	// 注册sql tracing
	driverName, err := otelsql.Register(
		conf.Database.Driver,
		otelsql.WithAttributes(semconv.DBSystemMariaDB),
		otelsql.WithSpanOptions(
			otelsql.SpanOptions{
				DisableErrSkip:       true,
				OmitConnectorConnect: true,
				OmitConnResetSession: true,
				OmitConnPrepare:      true,
				OmitRows:             true,
			}),
	)
	if err != nil {
		thisLog.Fatalf("sql tracing注册失败: %v", err)
	}

	// 连接数据库
	db, err := sql.Open(driverName, conf.Database.Source)
	if err != nil {
		thisLog.Fatalf("数据库连接失败: %v", err)
	}

	// 初始化ent客户端
	drv := entsql.OpenDB(conf.Database.Driver, db)
	// 配置sql日志打印
	sqlDrv := dialect.DebugWithContext(drv, func(ctx context.Context, i ...interface{}) {
		thisLog.WithContext(ctx).Debug(i...)
	})
	client := ent.NewClient(ent.Driver(sqlDrv))

	// 运行自动创建表
	//if err := db.Schema.Create(context.Background(), migrate.WithForeignKeys(false)); err != nil {
	//	thisLog.Fatalf("创建表失败: %v", err)
	//}
	return client
}

func NewRedis(conf *conf.Data, logger log.Logger) rueidis.Client {
	thisLog := log.NewHelper(logger)

	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress:      []string{conf.Redis.Addr},
		Password:         conf.Redis.Password,
		SelectDB:         int(conf.Redis.Db),
		ConnWriteTimeout: conf.Redis.WriteTimeout.AsDuration(),
	})
	// todo 等rueidis升级otel版本
	//client = rueidisotel.WithClient(client)

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

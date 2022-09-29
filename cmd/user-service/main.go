package main

import (
	"flag"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/lovechung/go-kit/bootstrap"
	"user-service/internal/conf"
)

// go build -ldflags "-X main.Service.Version=x.y.z"
var (
	Service = bootstrap.NewServiceInfo(
		"prd",
		"user-service",
		"1.0.0",
		"",
	)

	Flags = bootstrap.NewCommandFlags()
)

func init() {
	Flags.Init()
}

func newApp(logger log.Logger, gs *grpc.Server, rr registry.Registrar) *kratos.App {
	// 自定义服务端点
	//endpointStr := Flags.Endpoint
	//if endpointStr == "" {
	//	endpointStr = "127.0.0.1:9000"
	//}
	//endpoint, _ := url.Parse(fmt.Sprintf("grpc://%s?isSecure=false", endpointStr))
	return kratos.New(
		kratos.ID(Service.GetInstanceId()),
		kratos.Name(Service.Name+"-"+Service.Env),
		kratos.Version(Service.Version),
		kratos.Metadata(Service.Metadata),
		//kratos.Endpoint(endpoint),
		kratos.Logger(logger),
		kratos.Server(gs),
		kratos.Registrar(rr),
	)
}

// 加载启动配置
func loadConfig() (*conf.Bootstrap, *conf.Registry) {
	c := bootstrap.NewConfigProvider(Flags.Conf,
		Flags.ConfigType,
		Flags.ConfigHost,
		Flags.ConfigToken,
		"conf/"+Service.Name+"/"+Service.Env+"/data")
	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	var rc conf.Registry
	if err := c.Scan(&rc); err != nil {
		panic(err)
	}

	return &bc, &rc
}

// 加载otel配置
func loadOtel(bc *conf.Bootstrap) {
	bootstrap.NewTracerProvider(bc.Otel.Endpoint, Service.Env, &Service)
	bootstrap.NewMetricProvider(bc.Otel.Endpoint, Service.Env, &Service)
}

// 加载日志配置
func loadLogger(bc *conf.Bootstrap) log.Logger {
	return bootstrap.NewLoggerProvider(Service.Env, bc.Log.File, &Service)
}

func main() {
	flag.Parse()

	bc, rc := loadConfig()
	if bc == nil || rc == nil {
		panic("load config failed")
	}

	loadOtel(bc)

	logger := loadLogger(bc)

	app, cleanup, err := wireApp(bc.Server, bc.Data, rc, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}

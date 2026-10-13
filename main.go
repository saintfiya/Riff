package main

// @title Riff API
// @version 1.0
// @description Riff 社区讨论项目 API 文档
// @termsOfService http://swagger.io/terms/

// @contact.name sssss61616@gmail.com
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host 127.0.0.1:8888
// @BasePath /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"riff/conf"
	"riff/controller"
	"riff/dao/mysql"
	"riff/dao/redis"
	"riff/logger"
	"riff/pkg/jwt"
	"riff/pkg/snowflake"
	"riff/router"
	"riff/setting"
	"time"

	"go.uber.org/zap"
)

func main() {
	//加载配置文件
	if err := setting.Init(); err != nil {
		zap.L().Error("setting.Init failedd")
		return
	}
	//初始化日志
	if err := logger.InitLogger(&conf.Conf.Log, conf.Conf.App.Mode); err != nil {
		zap.L().Error("logger.InitLogger failed")
		return
	}
	defer func() { _ = zap.L().Sync() }()
	zap.L().Debug("InitLogger success...")
	if err := jwt.Init(conf.Conf.Auth.JWTSecret); err != nil {
		zap.L().Error("jwt.Init failed", zap.Error(err))
		return
	}
	//初始化mysql
	if err := mysql.InitMySQL(&conf.Conf.MySQL); err != nil {
		zap.L().Error("mysql.InitMySQL failed")
		return
	}
	defer mysql.Close()
	//初始化redis
	if err := redis.InitRedis(&conf.Conf.Redis); err != nil {
		zap.L().Error("redis.InitRedis failed")
		return
	}

	if err := snowflake.Init(conf.Conf.App.StartTime, conf.Conf.App.MachineID); err != nil {
		zap.L().Error("snowflake.Init failed")
		return
	}

	if err := controller.InitTrans("zh"); err != nil {
		zap.L().Error("InitTrans failed")
		return
	}

	defer redis.Close()
	//注册路由
	r := router.SetupRouter(conf.Conf.App.Mode)
	// r.Run(fmt.Sprintf(":%d", conf.Conf.App.Port))
	// //启动服务(优雅退出)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", conf.Conf.App.Port),
		Handler: r,
	}

	go func() {
		// 开启一个goroutine启动服务
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("listen:%s\n", zap.Error(err))
		}
	}()

	// 等待中断信号来优雅地关闭服务器，为关闭服务器操作设置一个5秒的超时
	quit := make(chan os.Signal, 1) // 创建一个接收信号的通道
	// kill 默认会发送 syscall.SIGTERM 信号
	// kill -2 发送 syscall.SIGINT 信号，我们常用的Ctrl+C就是触发系统SIGINT信号
	// kill -9 发送 syscall.SIGKILL 信号，但是不能被捕获，所以不需要添加它
	// signal.Notify把收到的 syscall.SIGINT或syscall.SIGTERM 信号转发给quit
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // 此处不会阻塞
	<-quit                                               // 阻塞在此，当接收到上述两种信号时才会往下执行
	zap.L().Info("Shutdown Server...")
	// 创建一个5秒超时的context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 5秒内优雅关闭服务（将未处理完的请求处理完再关闭服务），超过5秒就超时退出
	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("Server Shutdown", zap.Error(err))
	}

	zap.L().Info("Server exiting...")
}

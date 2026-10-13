package router

import (
	"net/http"
	"riff/controller"
	_ "riff/docs" // 千万不要忘了导入把你上一步生成的docs
	"riff/logger"
	middlewares "riff/middlerwares"

	files "github.com/swaggo/files"
	gs "github.com/swaggo/gin-swagger"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

func SetupRouter(mode string) *gin.Engine {
	if mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode) //gin设置成发布模式
	}

	r := gin.New()
	// r.Use(logger.GinLogger(), logger.GinRecovery(true), middlewares.RateLimitMiddleware(time.Second, 1))
	r.Use(logger.GinLogger(), logger.GinRecovery(true))

	r.LoadHTMLFiles("./templates/index.html")
	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	r.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.GET("/swagger/*any", gs.WrapHandler(files.Handler))

	v1 := r.Group("/api/v1")

	v1.POST("/signup", controller.SignUpHandler)
	v1.POST("/login", controller.LoginHandler)
	v1.GET("/posts", controller.PostList)
	v1.GET("/posts2", controller.GetPostListHandler2)
	v1.Use(middlewares.JWTAuthMiddleware())
	{
		v1.GET("/community", controller.CommunityList)
		v1.GET("/community/:id", controller.CommunityDetail)

		v1.POST("/post", controller.CreatePost)

		v1.GET("/post/:id", controller.GetPostByID)

		v1.POST("/vote", controller.VoteHandler)
	}

	if mode != gin.ReleaseMode {
		pprof.Register(r)
	}
	return r
}

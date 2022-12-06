package router

import (
	"crack_front/src/master/controller"
	"crack_front/src/master/middleware"
	"github.com/gin-gonic/gin"
)

// 路由单例(gin引擎单例)
var(
	Router			*gin.Engine
)

func init() {
	if Router == nil {
		var (
			adminRouter *gin.RouterGroup
		)
		// 设置为release模式

		gin.SetMode(gin.ReleaseMode)

		// 初始化路由(初始化gin引擎)
		Router = gin.Default()
		// 全局路由中间件
		Router.Use(middleware.Cors(), middleware.GinLog())

		// 注册和登录
		Router.POST("/register", controller.Register)
		Router.POST("/login", controller.Login)

		// 分组路由(需要鉴权)
		adminRouter = Router.Group("/api/v1", middleware.AuthMiddleware())
		{
			adminRouter.POST("/crack_identify", controller.CrackIdentify)

			adminRouter.DELETE("/delete", controller.RemoveTask)

			adminRouter.GET("/list", controller.GetTasks)

			adminRouter.POST("/kill", controller.KillTask)

			adminRouter.GET("/log", controller.QueryTaskLog)

			adminRouter.GET("/worker", controller.GetWorkers)
		}
	}
}

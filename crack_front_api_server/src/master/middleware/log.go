package middleware

import (
	"crack_front/src/master/logger"
	"github.com/gin-gonic/gin"
	"time"
)

// 日志

func GinLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		var(
			startTime 			time.Time
			endTime 			time.Time
			formatStr			string
		)
		// 月份 1,01,Jan,January
		// 日　 2,02,_2
		// 时　 3,03,15,PM,pm,AM,am
		// 分　 4,04
		// 秒　 5,05
		// 年　 06,2006
		// 时区 -07,-0700,Z0700,Z07:00,-07:00,MST
		// 周几 Mon,Monday
		formatStr = "[2006_Jan_02 15:04:05]"

		startTime = time.Now()
		// 处理请求
		c.Next()
		endTime = time.Now()
		logger.Logger.InfoLog("[GIN]", c.Request.Method, c.Request.RequestURI, c.Writer.Status(), startTime.Format(formatStr), endTime.Format(formatStr), endTime.Sub(startTime).Milliseconds(), "ms")
	}
}

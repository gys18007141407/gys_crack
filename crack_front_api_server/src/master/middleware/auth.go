package middleware

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"net/http"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			token				string
			jwtToken			*jwt.Token
			claims				*Claims
			err 				error
		)
		// 获取token
		token = c.GetHeader("Authorization")
		// 验证token
		jwtToken, claims, err = ParseToken(token)
		if err != nil || !jwtToken.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "token错误,请先登录!",
			})
			c.Abort()
			return
		}
		// 通过验证，用户已经登录,将UserId写入上下文
		c.Set("UserId", claims.UserId)
		c.Next()
	}
}

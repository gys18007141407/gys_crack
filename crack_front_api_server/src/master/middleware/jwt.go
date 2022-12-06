package middleware

import (
	"crack_front/src/common"
	"github.com/dgrijalva/jwt-go"
	"time"
)

var (
	jwtSecretKey  	= []byte("University Of Electric and Science Technology of China")
	tokenDuration  	= 1 * 24 * time.Hour
)

type Claims struct {
	UserId uint
	jwt.StandardClaims
}

// 分发token
func ReleaseToken(curUser *common.User) (token string, err error) {
	var (
		expirationTime		int64
		claims 				*Claims
	)
	expirationTime = time.Now().Add(tokenDuration).Unix()

	claims = &Claims{
		UserId: curUser.UserId,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime,
			IssuedAt:  time.Now().Unix(),
			Issuer:    "crack front api server",
			Subject:   "user token",
		},
	}

	if token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecretKey); err != nil {
		return "", err
	}
	return token, nil
}

// 解析token
func ParseToken(token string) (*jwt.Token, *Claims, error) {
	var (
		jwtToken		*jwt.Token
		claims 			*Claims
		err 			error
	)
	claims = &Claims{}

	jwtToken, err = jwt.ParseWithClaims(token, claims, func(jwtToken *jwt.Token) (i interface{}, err error) {
		return jwtSecretKey, nil
	})
	return jwtToken, claims, err
}


// jwt 工具
package jwtutil

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims jwt载荷
type Claims struct {
	UserId   int64  `json:"user"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = jwt.ErrTokenExpired
	ErrTokenSignatureErr = jwt.ErrTokenSignatureInvalid
)

// GenerateAccessToken 生成AccessToken 短时效的
func GenerateAccessToken(userId int64, username string, secret string, expHour int) (string, error) {
	claims := Claims{
		UserId:   userId,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expHour))), //过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),                                         //签发时间
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) //使用HS256(HMAC‑SHA256)签名算法
	return t.SignedString([]byte(secret))                  //组装JWTtoken
}

// GenerateRefreshToken 生成RefreshToken 定时刷新的
func GenerateRefreshToken(userId int64, username string, secret string, expDay int) (string, error) {
	claims := Claims{
		UserId:   userId,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(0, 0, expDay)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// ParseToken 解析+校验签名+校验是否过期
// 传入jwt原始字符串和密钥
func ParseToken(tokenStr string, secret string) (*Claims, error) {
	//ParseWithClaims 传入待解析字符串，载荷指针，回调函数（返回密钥）
	//返回签名密钥切片
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenSignatureErr
		}
		//字符串密钥转成字节切片
		return []byte(secret), nil
	})
	//出错，判断token非法
	if err != nil {
		return nil, err
	}

	//类型断言，把 token.Claims 转换成*Claims 判断token是否合法
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrInvalidToken
}

// IsTokenExpired 判断错误是否是token过期
func IsTokenExpired(err error) bool {
	return errors.Is(err, ErrTokenExpired)
}

// IsTokenSignatureError 判断是否是签名错误
func IsTokenSignatureError(err error) bool {
	return errors.Is(err, ErrTokenSignatureErr)
}

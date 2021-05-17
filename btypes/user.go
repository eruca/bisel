package btypes

import (
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/eruca/bisel/utils"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Loginer interface {
	GetAccount() PairStringer
	GetPassword() PairStringer
}

// LoginTabler 同时又Tabler与Login,就是该表作为用户登录表
type LoginTabler interface {
	Loginer
	Tabler
}

type Defaulter interface {
	UserID() uint
	Default() Defaulter
}

func login(db *DB, logger Logger, loginTabler LoginTabler, jwtSession Defaulter) (Tabler, error) {
	loginer := loginTabler.New().(LoginTabler)

	account := loginTabler.GetAccount()
	password := loginTabler.GetPassword()

	if err := db.Gorm.
		Model(loginTabler).
		Where(fmt.Sprintf("%s = ?", account.Key), account.Value).
		First(loginer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotExistOrPasswordNotCorrect
		}
		logger.Errorf("登录时，查找账号发生错误: %s", err.Error())
		panic("登录查询数据库错误")
	}

	pswdFromDb := loginer.GetPassword().Value.String()

	if err := bcrypt.CompareHashAndPassword([]byte(pswdFromDb), []byte(password.Value.String())); err != nil {
		logger.Infof(err.Error())
		return nil, ErrAccountNotExistOrPasswordNotCorrect
	}
	// 如果jwtSession为空，直接返回
	if jwtSession == nil {
		return loginer, nil
	}

	if err := mapstructure.Decode(loginer, jwtSession); err != nil {
		logger.Errorf("将登录用户转码值jwtSession时发生错误: %s", err.Error())
		panic("mapstructure.Decode(loginer,jwtSession) failed")
	}
	return loginer, nil
}

// 登录成功后产生的jwt返回给客户端
// todo: 1. 写在header() 2.写在payload里
func generate_jwt(jwtSession Defaulter, expire int, salt []byte) (string, error) {
	sess := utils.Struct2Map(jwtSession)
	sess["exp"] = time.Now().Add(time.Duration(expire) * time.Hour).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(sess))
	return token.SignedString(salt)
}

func loginJWT(injected Injected, loginTabler LoginTabler, jwtSession Defaulter) (Result, error) {
	loginer, err := login(injected.DB, injected.Logger, loginTabler, jwtSession)
	if err != nil {
		return Result{}, err
	}
	token, err := generate_jwt(jwtSession, injected.Expire, []byte(injected.JWTConfig.Salt))
	if err != nil {
		panic(err)
	}
	// todo: 是返回给Header还是Payload
	pairs := Pairs{Pair{Key: "token", Value: token}, Pair{Key: "user", Value: loginer}}
	return Result{pairs, false}, nil
}

func ParseToken(tokenString string, jwtSession Defaulter, salt []byte) error {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) { return salt, nil })
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				// Token is expired
				return ErrTokenExpired
			} else {
				return ErrInvalidToken
			}
		}
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		err = mapstructure.Decode(claims, jwtSession)
		if err != nil {
			panic(err)
		}
		return nil
	}
	return ErrInvalidToken
}

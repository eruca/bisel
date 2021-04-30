package btypes

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/eruca/bisel/utils"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	expireDuration = time.Hour * 24
	salt           = "big city"
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
	Default() Defaulter
}

func login(db *DB, loginTabler LoginTabler, jwtSession Defaulter) error {
	result := map[string]interface{}{}

	account := loginTabler.GetAccount()
	password := loginTabler.GetPassword()

	if err := db.Gorm.
		Model(loginTabler).
		Where(fmt.Sprintf("%s = ?", account.Key), account.Value).
		First(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAccountNotExistOrPasswordNotCorrect
		}
		panic(err)
	}

	passwordFromDb, ok := result[password.Key]
	if !ok {
		panic(fmt.Sprintf("%q 不存在于 数据库:%q中", password.Key, loginTabler.TableName()))
	}
	pswdFromDb, ok := passwordFromDb.(string)
	if !ok {
		panic(fmt.Sprintf("%s 应该是 string 类型", password.Key))
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password.Value.String()), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(pswdFromDb))
	if err != nil {
		log.Println(err)
		return ErrAccountNotExistOrPasswordNotCorrect
	}
	// 如果jwtSession为空，直接返回
	if jwtSession == nil {
		return nil
	}

	err = mapstructure.Decode(result, jwtSession)
	if err != nil {
		panic(err)
	}
	return nil
}

// 登录成功后产生的jwt返回给客户端
// todo: 1. 写在header() 2.写在payload里
func generate_jwt(jwtSession Defaulter) (string, error) {
	sess := utils.Struct2Map(jwtSession)
	sess["exp"] = time.Now().Add(expireDuration).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(sess))
	return token.SignedString([]byte(salt))
}

func login_jwt(db *DB, loginTabler LoginTabler, jwtSession Defaulter) (Pairs, bool, error) {
	err := login(db, loginTabler, jwtSession)
	if err != nil {
		return nil, false, err
	}
	token, err := generate_jwt(jwtSession)
	if err != nil {
		panic(err)
	}
	// todo: 是返回给Header还是Payload
	pairs := Pairs{Pair{Key: "token", Value: token}}
	return pairs, false, nil
}

func ParseToken(tokenString string, jwtSession Defaulter) error {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) { return []byte(salt), nil })
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

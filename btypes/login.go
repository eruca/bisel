package btypes

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
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

func login(db *DB, loginTabler LoginTabler) error {
	var passwordFromDB string

	account := loginTabler.GetAccount()
	password := loginTabler.GetPassword()

	if err := db.Gorm.
		Model(loginTabler).
		Select(password.Key).
		Where(fmt.Sprintf("%s = ?", account.Key), account.Value).
		Scan(&passwordFromDB).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAccountNotExistOrPasswordNotCorrect
		}
		panic(err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password.Value.String()), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(passwordFromDB))
	if err != nil {
		log.Println(err)
		return ErrAccountNotExistOrPasswordNotCorrect
	}
	return nil
}

type ClaimContent struct {
	ID uint
}

type Claims struct {
	ClaimContent
	jwt.StandardClaims
}

// 登录成功后产生的jwt返回给客户端
// todo: 1. 写在header() 2.写在payload里
func generate_jwt(tabler Tabler) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodES256,
		Claims{
			ClaimContent: ClaimContent{ID: tabler.Model().ID},
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(expireDuration).Unix(),
			},
		})
	return token.SignedString([]byte(salt))
}

func login_jwt(db *DB, loginTabler LoginTabler) (Pairs, error) {
	err := login(db, loginTabler)
	if err != nil {
		return nil, nil
	}
	token, err := generate_jwt(loginTabler)
	if err != nil {
		panic(err)
	}
	// todo: 是返回给Header还是Payload
	pairs := Pairs{Pair{Key: "token", Value: token}}
	return pairs, nil
}

func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{},
		func(t *jwt.Token) (interface{}, error) { return []byte(salt), nil })

	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, ErrInvalidToken
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				// Token is expired
				return nil, ErrTokenExpired
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, ErrInvalidToken
			} else {
				return nil, ErrInvalidToken
			}
		}
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrInvalidToken
}

package middlewares

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/eruca/bisel/btypes"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	key    = "Logio"
	jwtKey = "JWT"
)

var (
	ErrAccountNotExistOrPasswordNotCorrect = errors.New("账号不存在或密码错误")
)

type Loginer interface {
	GetAccount() btypes.PairStringer
	GetPassword() btypes.PairStringer
	DeletePassword()
}

// ParamLogio 代表登录与登出的Signal
type ParamLogio uint8

const (
	ParamLogin ParamLogio = iota
	ParamLogout
)

func (p ParamLogio) String() string {
	switch p {
	case ParamLogin:
		return "Param Login"
	case ParamLogout:
		return "Param Logout"
	default:
		panic("should not happened")
	}
}

type ParameterLogio struct {
	ParamLogio
	btypes.Tabler
	salt   string
	expire int
}

func (p *ParameterLogio) FromRawMessage(tabler btypes.Tabler, rm json.RawMessage) {
	err := json.Unmarshal(rm, tabler)
	if err != nil {
		panic(err)
	}
	p.Tabler = tabler
}

func (*ParameterLogio) Status() btypes.RequestStatus { return btypes.StatusNoop }
func (*ParameterLogio) ReadForceUpdate() bool        { return false }
func (*ParameterLogio) BuildCacheKey(string) string  { return "" }
func (p *ParameterLogio) JwtCheck() bool {
	switch p.ParamLogio {
	case ParamLogin:
		return false
	case ParamLogout:
		return true
	default:
		return true
	}
}

func (p *ParameterLogio) Call(c *btypes.Context, tabler btypes.Tabler) (result btypes.Result, err error) {
	switch p.ParamLogio {
	case ParamLogin:
		loginer, err := LoginAssert(c)
		if err == nil {
			token, err := Generate_jwt(c.JwtSess, p.expire, []byte(p.salt))
			if err != nil {
				panic(err)
			}
			// 删除密码再返回
			loginer.(Loginer).DeletePassword()
			// todo: 是返回给Header还是Payload
			pess := make([]string, 0, len(c.PessimisticLock))
			for k := range c.PessimisticLock {
				pess = append(pess, k)
			}

			pairs := btypes.Pairs{
				btypes.Pair{Key: "token", Value: token},
				btypes.Pair{Key: "user", Value: loginer},
				btypes.Pair{Key: "pess_lock", Value: pess},
			}
			result.Payloads = pairs
		}
	case ParamLogout:
		result.Payloads.Push("msg", "logout success")
	default:
		panic("should not happened")
	}

	return
}

// expire: 默认jwt token过期时间
// salt: jwt添加的salt
func ConfigLoginHandler(expire int, salt string) func(btypes.Tabler, btypes.JwtSession, ...btypes.Action) btypes.ContextConfig {
	return func(tabler btypes.Tabler, jwt btypes.JwtSession, actions ...btypes.Action) btypes.ContextConfig {
		return btypes.HandlerFunc(tabler,
			&ParameterLogio{
				ParamLogio: ParamLogin,
				expire:     expire,
				salt:       salt,
			}, jwt, actions...)
	}
}

func LogoutHandler(tabler btypes.Tabler, actions ...btypes.Action) btypes.ContextConfig {
	return btypes.HandlerFunc(tabler, &ParameterLogio{ParamLogio: ParamLogout}, nil, actions...)
}

// todo: JWT CHECK
func JWTAuthorize(jwt btypes.JwtSession, salt string) btypes.Action {
	var jwtPool = sync.Pool{
		New: func() interface{} {
			return jwt.New()
		},
	}

	return func(c *btypes.Context) btypes.PairStringer {
		var token string
		if !c.Parameter.JwtCheck() {
			c.Next()
			return btypes.PairStringer{Key: jwtKey, Value: btypes.ValueString(fmt.Sprintf("%s 白名单", c.Parameter))}
		}
		if c.ConnectionType == btypes.HTTP {
			if v := c.HttpReq.Header.Get("Authorization"); len(v) > 7 && strings.ToLower(v[:6]) == "bearer" {
				// 要去掉bearer后面的一个空格，所以时7开始
				token = v[7:]
				return parse(c, token, salt, &jwtPool)
			}
		}

		if c.Request.Token == "" {
			c.Responder = btypes.BuildErrorResposeFromRequest(c.ConfigResponseType, c.Request, btypes.ErrInvalidToken)
			return btypes.PairStringer{Key: jwtKey, Value: btypes.ValueString(btypes.ErrInvalidToken.Error())}
		}
		return parse(c, c.Request.Token, salt, &jwtPool)
	}
}

func parse(c *btypes.Context, token, salt string, jwtSessionPool *sync.Pool) btypes.PairStringer {
	sess := jwtSessionPool.Get().(btypes.JwtSession)
	err := parseToken(token, sess, []byte(salt))
	if err != nil {
		c.Responder = btypes.BuildErrorResposeFromRequest(c.ConfigResponseType, c.Request, err)
		return btypes.PairStringer{Key: jwtKey, Value: btypes.ValueString(err.Error())}
	}
	c.JwtSess = sess
	c.Next()

	jwtSessionPool.Put(sess)
	return btypes.PairStringer{Key: jwtKey, Value: btypes.ValueString("JWT authority success")}
}

func parseToken(tokenString string, jwtSession btypes.JwtSession, salt []byte) error {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) { return salt, nil })
	if err != nil {
		return btypes.ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		err = mapstructure.Decode(claims, jwtSession)
		if err != nil {
			panic(err)
		}
		return nil
	}
	return btypes.ErrInvalidToken
}

func LoginAssert(c *btypes.Context) (btypes.Tabler, error) {
	param, ok := c.Parameter.(*ParameterLogio)
	if !ok {
		panic("Parameter 不是 *Param")
	}

	// 来之客户端的数据
	loginer, ok := param.Tabler.(Loginer)
	if !ok {
		panic("loginer 必须实现 Loginer 接口")
	}
	// 作为登录成功后数据的接收者
	tabler := param.Tabler.New()

	account := loginer.GetAccount()
	password := loginer.GetPassword()

	if err := c.DB.Gorm.
		Model(tabler).
		Where(fmt.Sprintf("%s = ?", account.Key), account.Value).
		First(tabler).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAccountNotExistOrPasswordNotCorrect
		}
		c.Logger.Errorf("登录时，查找账号发生错误: %s", err.Error())
		panic("登录查询数据库错误")
	}
	c.Logger.Infof("loginer from db: %v", loginer)

	pswdFromDb := tabler.(Loginer).GetPassword().Value.String()
	c.Logger.Infof("password from db: %s => %s", pswdFromDb, password.Value.String())
	if err := bcrypt.CompareHashAndPassword([]byte(pswdFromDb), []byte(password.Value.String())); err != nil {
		c.Logger.Warnf(err.Error())
		return nil, ErrAccountNotExistOrPasswordNotCorrect
	}
	// 如果jwtSession为空，直接返回
	if c.JwtSess == nil {
		return tabler, nil
	}

	if err := mapstructure.Decode(loginer, c.JwtSess); err != nil {
		c.Logger.Errorf("将登录用户转码值jwtSession时发生错误: %s", err.Error())
		panic("mapstructure.Decode(loginer,jwtSession) failed")
	}
	return tabler, nil
}

// 登录成功后产生的jwt返回给客户端
// todo: 1. 写在header() 2.写在payload里
func Generate_jwt(jwtSession btypes.JwtSession, expire int, salt []byte) (string, error) {
	sess := btypes.Struct2Map(jwtSession)
	sess["exp"] = time.Now().Add(time.Duration(expire) * time.Hour).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(sess))
	return token.SignedString(salt)
}

func Struct2Map(obj interface{}) map[string]interface{} {
	t := reflect.TypeOf(obj)

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v)

	var data = make(map[string]interface{})

top:
	for i := 0; i < t.NumField(); i++ {
		name, ok := t.Field(i).Tag.Lookup("gorm")
		if ok {
		inner:
			for _, part := range strings.Split(name, ";") {
				kv := strings.Split(part, ":")
				if len(kv) != 2 {
					continue inner
				}
				if kv[0] == "column" {
					data[kv[1]] = v.Field(i).Interface()
					continue top
				}
			}
		}

		name, ok = t.Field(i).Tag.Lookup("json")
		if ok && name != "-" {
			name = strings.TrimSuffix(name, ",omitempty")
			data[name] = v.Field(i).Interface()
			continue
		}

		data[camelToSnake(t.Field(i).Name)] = v.Field(i).Interface()
	}
	return data
}

func camelToSnake(s string) string {
	out := make([]rune, 0, len(s)*2)
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i == 0 {
				out = append(out, c-'A'+'a')
			} else {
				out = append(out, '_', c-'A'+'a')
			}
		} else {
			out = append(out, c)
		}
	}
	return string(out)
}

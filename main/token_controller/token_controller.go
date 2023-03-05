package token_controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"main/db"
	"math/rand"
	"strings"
	"time"
)

var (
	secret      string
	ErrIllegal  = errors.New("token illegal")
	ErrExpired  = errors.New("token expire")
	ErrMismatch = errors.New("token mismatch")
)

type TokenInfo struct {
	Uid string
	Exp int
}

func init() {
	rand.Seed(time.Now().Unix())
	secret = fmt.Sprintf("%08d", rand.Intn(100000000))
}

func hmacSha256(data string, secret string) (string, error) {
	h := hmac.New(sha256.New, []byte(secret))
	_, err := h.Write([]byte(data))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func CreateToken(uid string, exp time.Duration) (string, error) {
	payload := `{"Uid":"` + uid + `", "Exp":` + fmt.Sprintf("%d", time.Now().Add(exp).Unix()) + "}"
	data := base64.URLEncoding.EncodeToString([]byte(payload))
	signature, err := hmacSha256(data, secret)
	if err != nil {
		return "", err
	}
	return data + "." + signature, err
}

func IsTokenlegal(token string) (bool, error) {
	// 判断 token 是否被人为修改
	a := strings.Split(token, ".")
	trueSignature, err := hmacSha256(a[0], secret)
	if err != nil {
		return false, err
	}

	if trueSignature == a[1] {
		return true, nil
	} else {
		return false, nil
	}
}

func GetTokenInfo(token string) (TokenInfo, error) {
	a := strings.Split(token, ".")
	var tokenInfo TokenInfo
	b, err := base64.URLEncoding.DecodeString(a[0])
	if err != nil {
		return tokenInfo, err
	}
	err = json.Unmarshal(b, &tokenInfo)
	return tokenInfo, err
}

func IsTokenExpired(tokenInfo TokenInfo) (bool, error) {
	if tokenInfo.Exp >= int(time.Now().Unix()) {
		return false, nil
	} else {
		return true, nil
	}
}

func IsTokenBelongToUser(tokenInfo TokenInfo, username string) (bool, error) {
	// 验证token是否属于该用户
	var s string
	err := db.Mysql.QueryRow(`select 1 from table_users where delete_at is null and id = ? and name = ? limit 1;`, tokenInfo.Uid, username).Scan(&s)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		} else {
			return false, err
		}
	} else {
		return true, nil
	}
}

func ParseToken(token string, username string) (TokenInfo, error) {
	tokenInfo := TokenInfo{}
	b, err := IsTokenlegal(token)
	if err != nil {
		return tokenInfo, err
	}
	if !b {
		return tokenInfo, ErrIllegal
	}

	tokenInfo, err = GetTokenInfo(token)
	if err != nil {
		return tokenInfo, err
	}
	b, err = IsTokenExpired(tokenInfo)
	if err != nil {
		return tokenInfo, err
	}
	if b {
		return tokenInfo, ErrExpired
	}
	b, err = IsTokenBelongToUser(tokenInfo, username)
	if err != nil {
		return tokenInfo, err
	}
	if !b {
		return tokenInfo, ErrMismatch
	}
	return tokenInfo, nil
}

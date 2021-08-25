package sf

import (
	"github.com/golang-jwt/jwt"
	"time"
)

type JwtInfo struct {
	Ext      map[string]interface{}
	Issuer   string
	Subject  string
	Audience string
}

type CustomClaims struct {
	Ext map[string]interface{} `json:"ext"`
	jwt.StandardClaims
}

func CreateJwt(secret string, issuer string, subject string, audience string, claims map[string]interface{}) (string, error) {
	mySigningKey := []byte(secret)

	after30Seconds := time.Now().Add(time.Second * 30)

	cclaims := CustomClaims{
		Ext: claims,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: after30Seconds.UTC().Unix(),
			Issuer:    issuer,
			Subject:   subject,
			Audience:  audience,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, cclaims)
	return token.SignedString(mySigningKey)
}

func DecodeJwt(secret string, tokenString string) (JwtInfo, error) {

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return JwtInfo{}, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return JwtInfo{Subject: claims.Subject, Audience: claims.Audience, Issuer: claims.Issuer, Ext: claims.Ext}, nil
	} else {
		return JwtInfo{}, err
	}
}

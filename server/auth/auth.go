package auth

import (
	"crypto/sha512"
	"encoding/hex"
	"golang.org/x/crypto/bcrypt"
	"regexp"
	"strings"

	"crypto/sha256"
	"github.com/gin-gonic/gin"
	"net/http"
)

var algorithms = map[string]func(string, string) bool{
	`plain`: func(hashed, password string) bool {
		return hashed == password
	},
	`sha256`: func(hashed, password string) bool {
		hash := sha256.Sum256([]byte(password))
		return hashed == hex.EncodeToString(hash[:])
	},
	`sha512`: func(hashed, password string) bool {
		hash := sha512.Sum512([]byte(password))
		return hashed == hex.EncodeToString(hash[:])
	},
	`bcrypt`: func(hashed, password string) bool {
		return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)) == nil
	},
}

func BasicAuth(accounts map[string]string, realm string) gin.HandlerFunc {
	type cipher struct {
		algorithm string
		password  string
	}
	if len(realm) == 0 {
		realm = `Authorization Required`
	}
	reg := regexp.MustCompile(`^\$([a-zA-Z0-9]+)\$(.*)$`)
	stdAccounts := make(map[string]cipher)
	for user, pass := range accounts {
		if match := reg.FindStringSubmatch(pass); len(match) > 0 {
			match[1] = strings.ToLower(match[1])
			if _, ok := algorithms[match[1]]; ok {
				stdAccounts[user] = cipher{
					algorithm: match[1],
					password:  match[2],
				}
				continue
			}
		}
		stdAccounts[user] = cipher{
			algorithm: `plain`,
			password:  pass,
		}
	}

	return func(c *gin.Context) {
		user, pass, ok := c.Request.BasicAuth()
		if !ok {
			c.Header(`WWW-Authenticate`, `Basic realm=`+realm)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if account, ok := stdAccounts[user]; ok {
			if check, ok := algorithms[account.algorithm]; ok {
				if check(account.password, pass) {
					c.Set(`user`, user)
					return
				}
			}
		}
		c.Header(`WWW-Authenticate`, `Basic realm=`+realm)
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}

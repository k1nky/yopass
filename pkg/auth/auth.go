package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"gopkg.in/yaml.v3"
)

type Auth interface {
	Authorize(r *http.Request) (result []byte, user *User, err error)
	AuthorizeRequest(r *http.Request) (user *User, err error)
}

type User struct {
	Username string
	Password string
	Role     string
}

type JwtAuthOptions struct {
	Secret []byte
	Users  []User
}

type JwtAuth struct {
	Path    string
	Options *JwtAuthOptions
}

type NoAuth struct{}

func GenerateKey(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:length], nil
}

func (auth *JwtAuth) Load(filename string) (err error) {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	auth.Path = filename
	auth.Options = &JwtAuthOptions{}
	err = yaml.NewDecoder(f).Decode(auth.Options)
	return
}

func (auth *JwtAuth) Save() error {
	f, err := os.Create(auth.Path)
	if err != nil {
		return err
	}

	return yaml.NewEncoder(f).Encode(auth.Options)
}

func NewJwtAuth(filename string, adminName string) *JwtAuth {
	secret, _ := GenerateKey(64)
	password, _ := GenerateKey(10)
	return &JwtAuth{
		Path: filename,
		Options: &JwtAuthOptions{
			Secret: []byte(secret),
			Users: []User{
				{
					Username: adminName,
					Password: password,
					Role:     "admin",
				},
			},
		},
	}
}

func (auth *JwtAuth) ValidateUser(user *User) bool {
	for _, u := range auth.Options.Users {
		if u.Username == user.Username && u.Password == user.Password {
			user.Role = u.Role
			return true
		}
	}
	return false
}

func (auth *JwtAuth) GenerateToken(u User) []byte {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"role": u.Role,
		"name": u.Username,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(auth.Options.Secret)
	if err != nil {
		fmt.Println(err)
	}
	return []byte(tokenString)
}

func (auth *JwtAuth) AuthorizeRequest(r *http.Request) (*User, error) {
	if r.Header["Token"] == nil {
		return nil, fmt.Errorf("no auth token found")
	}
	tokenString := r.Header["Token"][0]
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("there was an error in parsing")
		}
		return auth.Options.Secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token")
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &User{
			Username: claims["name"].(string),
			Role:     claims["role"].(string),
		}, nil
	}
	return nil, fmt.Errorf("unauthorized token")
}

func (auth *JwtAuth) Authorize(r *http.Request) (result []byte, user *User, err error) {
	user = &User{}
	if err = json.NewDecoder(r.Body).Decode(user); err != nil {
		return result, nil, fmt.Errorf("invalid request")
	}
	if !auth.ValidateUser(user) {
		return result, user, fmt.Errorf("username or password is incorrect")
	}
	token := auth.GenerateToken(*user)
	return token, user, nil
}

func (auth *NoAuth) Authorize(r *http.Request) (result []byte, user *User, err error) {
	return
}

func (auth *NoAuth) AuthorizeRequest(r *http.Request) (user *User, err error) {
	return
}

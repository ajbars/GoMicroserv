package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt"
)

var lg = log.New(os.Stdout, "authapi: ", log.Lshortfile)
var rdb = redis.NewClient(&redis.Options{
	Addr: os.Getenv("REDIS_HOST"),
})

func main() {

	rdbTokenKey := func(m jwt.MapClaims) (string, error) {
		i, ok := m["firstname"]

		if !ok {
			return "", errors.New("firstname is required")
		}

		u, ok := i.(string)

		if !ok {
			return "", errors.New("firstname must be a string")
		}

		return u, nil
	}

	http.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		retErr := func(msg string, code int) {
			lg.Printf("create error: %s, code %d", msg, code)
			http.Error(w, msg, code)
		}

		b, err := io.ReadAll(r.Body)

		if err != nil {
			retErr(err.Error(), http.StatusInternalServerError)
			return
		}

		defer r.Body.Close()

		m := jwt.MapClaims{}

		if err := json.Unmarshal(b, &m); err != nil {
			retErr(err.Error(), http.StatusBadRequest)
			return
		}

		t, err := create(m)

		if err != nil {
			retErr(err.Error(), http.StatusInternalServerError)
			return
		}

		key, err := rdbTokenKey(m)

		if err != nil {
			retErr(err.Error(), http.StatusBadRequest)
			return
		}

		err = rdb.Set(r.Context(), key, t, time.Hour*166).Err()

		if err != nil {
			retErr(err.Error(), http.StatusInternalServerError)
			return
		}

		lg.Println("token created")

		w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", t))
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		retErr := func(msg string, code int) {
			lg.Printf("verify error: %s, code %d", msg, code)
			http.Error(w, msg, code)
		}

		vals := strings.Split(r.Header.Get("Authorization"), "Bearer ")

		if len(vals) != 2 {
			retErr(
				"invalid token format: expects \"Bearer tokenstring\"",
				http.StatusUnauthorized,
			)
			return
		}

		t, m, err := parse(vals[1])

		if err != nil || !t.Valid {
			if err == nil {
				err = errors.New("invalid token")
			}
			retErr(err.Error(), http.StatusUnauthorized)
			return
		}

		key, err := rdbTokenKey(*m)

		if err != nil {
			retErr(err.Error(), http.StatusBadRequest)
			return
		}

		tDb, err := rdb.Get(r.Context(), key).Result()

		if err != nil {
			retErr(err.Error(), http.StatusBadRequest)
			return
		}

		if tDb == "" {
			retErr("token not found, please login in again", http.StatusBadRequest)
			return
		}

		if tDb != vals[1] {
			retErr("tokens not matched", http.StatusBadRequest)
			return
		}

		lg.Println("valided successfuly")
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		retErr := func(msg string, code int) {
			lg.Printf("logout error: %s, code %d", msg, code)
			http.Error(w, msg, code)
		}

		vals := strings.Split(r.Header.Get("Authorization"), "Bearer ")

		if len(vals) != 2 {
			retErr(
				"invalid token format: expects \"Bearer tokenstring\"",
				http.StatusBadRequest,
			)
			return
		}

		_, m, err := parse(vals[1])

		if err != nil {
			retErr(err.Error(), http.StatusBadRequest)
			return
		}

		key, err := rdbTokenKey(*m)

		if err != nil {
			retErr(err.Error(), http.StatusBadRequest)
			return
		}

		if res := rdb.Del(r.Context(), key); res.Err() != nil {
			retErr(res.Err().Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	lg.Println("started")
	lg.Fatal(http.ListenAndServe(":8082", nil))
	// lg.Fatal(http.ListenAndServe(":8089", nil))

}

func create(m jwt.MapClaims) (string, error) {
	b, err := loadKey()

	if err != nil {
		return "", err
	}

	pk, err := jwt.ParseECPrivateKeyFromPEM(b)

	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, m)

	return token.SignedString(pk)
}

func parse(raw string) (*jwt.Token, *jwt.MapClaims, error) {
	claims := &jwt.MapClaims{}

	t, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		b, err := loadKey()
		if err != nil {
			return nil, err
		}

		pk, err := jwt.ParseECPrivateKeyFromPEM(b)

		if err != nil {
			return nil, err
		}

		return pk.Public(), err
	})

	return t, claims, err
}

func loadKey() ([]byte, error) {
	f, err := os.Open(os.Getenv("PRIVATE_KEY_PATH"))

	if err != nil {
		return nil, err
	}

	defer f.Close()
	return io.ReadAll(f)
}

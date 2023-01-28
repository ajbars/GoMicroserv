package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var lg = log.New(os.Stdout, "proxy: ", log.Lshortfile)
var client = &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
	MaxIdleConns:       10,
	IdleConnTimeout:    30 * time.Second,
	DisableCompression: true,
}}

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// verify JWT
		authReq, err := http.NewRequestWithContext(
			r.Context(),
			"POST",
			fmt.Sprintf("http://%s/verify", os.Getenv("AUTHAPI_HOST")),
			// fmt.Sprintf("http://%s/verify", ("127.0.0.1:8082")),

			nil,
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		authReq.Header.Add("Authorization", r.Header.Get("Authorization"))
		resp, err := client.Do(authReq)

		defer lg.Printf("authapi response %v, error %v", resp, err)

		if err != nil || resp.StatusCode != http.StatusOK {
			retErr(w, "invalid token", err, resp)
			return
		}

		resp.Body.Close()

		resp, err = transmitReq(client, r)
		if err != nil {
			http.Error(w, err.Error(), resp.StatusCode)
			return
		}

		defer resp.Body.Close()
		io.Copy(w, resp.Body)
		copyHeader(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
	})

	//TODO work on architecture - handlers to a separate file, doubling code goes to separate function 69-81 encapsulate in a struct, return res, err, same with auth api, handlers and code goes out

	http.HandleFunc("/signup", loginOrRegisterHndl) //check post only

	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) { // check post only
		authReq, err := http.NewRequestWithContext(
			r.Context(),
			"POST",
			// fmt.Sprintf("http://%s/logout", os.Getenv("AUTHAPI_HOST")),
			fmt.Sprintf("http://%s/logout", "127.0.0.1:8082"),

			nil,
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		authReq.Header.Add("Authorization", r.Header.Get("Authorization"))
		resp, err := client.Do(authReq)

		defer lg.Printf("authapi response %v, error %v", resp, err)

		if err != nil || resp.StatusCode != http.StatusOK {
			retErr(w, "logout failed", err, resp)
			return
		}

		defer resp.Body.Close()

		w.WriteHeader(http.StatusOK)
	})

	lg.Println("started")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func loginOrRegisterHndl(w http.ResponseWriter, r *http.Request) {
	resp, err := transmitReq(client, r)

	if err != nil || resp.StatusCode != http.StatusOK {
		retErr(w, "request failed", err, resp)
		return
	}

	defer resp.Body.Close()

	var buf bytes.Buffer
	io.Copy(&buf, resp.Body)

	authReq, err := http.NewRequestWithContext(
		r.Context(),
		"POST",
		// fmt.Sprintf("http://%s/create", os.Getenv("AUTHAPI_HOST")),
		fmt.Sprintf("http://%s/create", os.Getenv("127.0.0.1:8082")),

		// "http://localhost:8089/create",

		&buf,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp2, err := client.Do(authReq)

	defer lg.Printf("authapi response %v, error %v", resp2, err)

	if err != nil || resp2.StatusCode != http.StatusOK {
		retErr(w, "token creation failed", err, resp2)
		return
	}

	defer resp2.Body.Close()

	w.Header().Set("Authorization", resp2.Header.Get("Authorization"))
	w.WriteHeader(http.StatusOK)
}

func transmitReq(c *http.Client, r *http.Request) (*http.Response, error) {
	// r.URL.Host = os.Getenv("MAINAPI_HOST")
	r.URL.Host = "mainapi:8080"

	r.URL.Scheme = "http"
	return c.Transport.RoundTrip(r)
}

func retErr(w http.ResponseWriter, msg string, err error, resp *http.Response) {
	if err == nil {
		err = errors.New(msg)
	}
	var code int
	if resp == nil {
		code = http.StatusInternalServerError
	} else {
		code = resp.StatusCode
		resp.Body.Close()
	}

	http.Error(w, err.Error(), code)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

package main

import (
	"net/http"
)

type A struct {
}

func (_ *A) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hello"))
}

func main() {
	http.ListenAndServe(":9122", &A{})
}

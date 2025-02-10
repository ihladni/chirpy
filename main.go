package main

import (
	"net/http"
)

func main() {
	mux := http.NewServeMux() // *ServeMux

	var srv *http.Server = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	err := srv.ListenAndServe()
	if err != nil {
		println(err)
	}
}

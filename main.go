package main

import (
	"log"
	"net/http"
)

// build cmd: go build -o out && ./out

func main() {
	var port string = "8080"
	mux := http.NewServeMux() // *ServeMux

	var srv *http.Server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
	/*
		err := srv.ListenAndServe()
		if err != nil {
			println(err)
		}
	*/
}

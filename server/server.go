package server

import (
	"fmt"
	"log"
	"net/http"
)

const PORT = "8080"

func Start(){
	mutex := http.NewServeMux()

	server := http.Server{
		Addr : ":" + PORT,
		Handler: mutex,
	}
	
	mutex.Handle("/",http.FileServer(http.Dir(".")))
	mutex.HandleFunc("/healthz",)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Server listening on port: %s",PORT)
}

func handlerHealthz(w http.ResponseWriter,req *http.Request){
	res := http.Response{}
	res.Header.Set("Content-Type","text/plain")
	
}
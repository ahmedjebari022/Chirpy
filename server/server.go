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
	
	mutex.Handle("/app/",http.FileServer(http.Dir(".")))
	mutex.HandleFunc("/healthz",handlerHealthz)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Server listening on port: %s",PORT)
}

func handlerHealthz(w http.ResponseWriter,req *http.Request){
	
	w.Header().Set("Content-Type","text/plain")
	w.Header().Set("charset","utf-8")
	w.WriteHeader(200)
	w.Write([]byte("Ok"))
	
}
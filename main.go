package main

import (
	"github.com/ahmedjebari022/Chripy/server"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)
func main(){
	godotenv.Load()


	server.Start()

}
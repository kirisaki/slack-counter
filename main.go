package main

import (
	"fmt"
	"net/http"
	"log"
)

func hello(w http.ResponseWriter, r *http.Request){
	fmt.Fprintf(w, "nyaan")
}

func main(){
	http.HandleFunc("/", hello)
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

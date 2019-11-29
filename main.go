package main

import (
	"net/http"
	"log"
	"time"

	_ "github.com/influxdata/influxdb1-client" 
	client "github.com/influxdata/influxdb1-client/v2"
)

func hello(w http.ResponseWriter, r *http.Request){
	c, err := client.NewHTTPClient(client.HTTPConfig{Addr: "http://localhost:8086"})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: "test",
		Precision: "s",
	})
	if err != nil {
		log.Fatal(err)
	}

	tags := map[string]string{"tick": "tick"}
	fields := map[string]interface{}{
		"foo": "hoge",
	}

	pt, err := client.NewPoint("tick", tags, fields, time.Now())
	if err != nil {
		log.Fatal(err)
	}
	bp.AddPoint(pt)

	c.Write(bp)
}

func main(){

	http.HandleFunc("/", hello)
	log.Print("connect influxDB at localhost")

	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

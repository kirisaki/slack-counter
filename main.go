package main

import (
	"net/http"
	"log"
	"time"
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"strconv"

	_ "github.com/influxdata/influxdb1-client" 
	client "github.com/influxdata/influxdb1-client/v2"

	"github.com/nlopes/slack/slackevents"

)

type Setting struct {
	SlackToken string
	SlackVerifyToken string
	InfluxDBURL string
	InfluxDBName string
}

type DailyResponse struct {
	Daily [][][]int `json:"daily"`
}

type ErrorResponse struct {
	Msg string `json:"msg"`
}

func (s Setting)eventHandler(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	body := buf.String()
	eventsAPIEvent, e := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: s.SlackVerifyToken}))
	if e != nil {
		log.Print(e)
		w.WriteHeader(http.StatusInternalServerError)
	}

	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err := json.Unmarshal([]byte(body), &r)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "text")
		w.Write([]byte(r.Challenge))
	}
	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			s.measure(ev)
		}
	}
}

func (s Setting) measure(ev *slackevents.MessageEvent){
	c, err := client.NewHTTPClient(client.HTTPConfig{Addr: s.InfluxDBURL})
	if err != nil {
		log.Print(err)
	}
	defer c.Close()

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: "test",
		Precision: "us",
	})
	if err != nil {
		log.Print(err)
	}

	ut := strings.Split(ev.TimeStamp, ".")
	if len(ut) != 2 {
		log.Print("invalid timestamp")
		return
	}
	sec, err0 := strconv.ParseInt(ut[0], 10, 64)
	nsec, err1 := strconv.ParseInt(ut[1], 10, 64)
	if err0 != nil || err1 != nil {
		log.Print("failed parsing number")
		return
	}
	ts := time.Unix(sec, nsec)
	tags := map[string]string{"channel": ev.Channel}
	fields := map[string]interface{}{
		"user": ev.User,
	}

	pt, err := client.NewPoint("activity", tags, fields, ts)
	if err != nil {
		log.Print(err)
	}
	bp.AddPoint(pt)

	log.Print("write measurement")
	er := c.Write(bp)
	if er != nil {
		log.Print(er)
	}
}

func (s Setting) queryHandler(w http.ResponseWriter, r *http.Request) {
	year := r.URL.Query().Get("year")
	month := r.URL.Query().Get("month")
	day := r.URL.Query().Get("day")
	//channel := r.URL.Query().Get("channel")
	end, err := time.Parse("2006-1-2", year + "-" + month + "-" + day)
	if err != nil {
		log.Print(err)
		resp, _ := json.Marshal(ErrorResponse{"invalid query: " + r.URL.RawQuery})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(resp)
	}
	days, err := strconv.ParseInt(r.URL.Query().Get("day"), 10, 64)
	if err != nil {
		log.Print(err)
		resp, _ := json.Marshal(ErrorResponse{"invalid query: " + r.URL.RawQuery})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(resp)
		return
	}

	start := end.Add(time.Hour * -24 * time.Duration(days))
	log.Print(start)

	c, err := client.NewHTTPClient(client.HTTPConfig{Addr: s.InfluxDBURL})
	if err != nil {
		log.Print(err)
	}
	defer c.Close()

	qstr := "SELECT COUNT() FROM \"activity\""
	q := client.NewQuery(qstr, s.InfluxDBName, "us")
	if resp, err := c.Query(q); err == nil && resp.Error() == nil {
		log.Print(resp.Results)
	} else {
		log.Print(err)
		log.Print(resp.Error())
	}
}

func main(){
	t := os.Getenv("SLACK_TOKEN")
	if t == "" {
		log.Fatal("set SLACK_TOKEN")
	}

	vt := os.Getenv("SLACK_VERIFY_TOKEN")
	if vt == "" {
		log.Fatal("set SLACK_VERIY_TOKEN")
	}

	u := os.Getenv("INFLUX_DB_URL")
	if u == "" {
		log.Fatal("set INFLUX_DB_URL")
	}

	p := os.Getenv("SERVER_PORT")
	if p == "" {
		log.Fatal("set SERVER_PORT")
	}

	db := os.Getenv("INFLUX_DB_NAME")
	if db == "" {
		log.Fatal("set INFLUX_DB_NAME")
	}
	/*
	user := os.Getenv("DASHBOARD_USER")
	if user == "" {
		log.Fatal("set DASHBOARD_USER")
	}

	pass := os.Getenv("DASHBOARD_PASS")
	if pass == "" {
		log.Fatal("set DASHBOARD_PASS")
	}
*/
	setting := Setting{
		SlackToken: t,
		SlackVerifyToken: vt,
		InfluxDBURL: u,
		InfluxDBName: db,
	}
	http.HandleFunc("/event", setting.eventHandler)
	http.HandleFunc("/query", setting.queryHandler)

	err := http.ListenAndServe(":" + p, nil)
	if err != nil {
		log.Fatal(err)
	}
}

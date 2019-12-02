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
	influxdb "github.com/influxdata/influxdb1-client/v2"

	redis "github.com/go-redis/redis/v7"

	//"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"

)

type Setting struct {
	SlackToken string
	SlackVerifyToken string
	InfluxDB influxdb.Client
	InfluxDBName string
	Redis *redis.Client
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
			s.measure(ev, eventsAPIEvent.TeamID)
		}
	}
}

func (s Setting) measure(ev *slackevents.MessageEvent, team string){
	bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
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
	tags := map[string]string{"team": team}
	fields := map[string]interface{}{
		"user": ev.User,
		"channel": ev.Channel,
	}

	pt, err := influxdb.NewPoint("activity", tags, fields, ts)
	if err != nil {
		log.Print(err)
	}
	bp.AddPoint(pt)

	log.Print("write measurement")
	er := s.InfluxDB.Write(bp)
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

	qstr := "SELECT COUNT(\"user\") FROM \"activity\""
	q := influxdb.NewQuery(qstr, s.InfluxDBName, "us")
	if resp, err := s.InfluxDB.Query(q); err == nil && resp.Error() == nil {
		log.Print(resp.Results)
	} else {
		if err != nil {
			log.Print(err)
		} else {
			log.Print(resp.Error())
		}
	}
}

func (s Setting)initialize(){
	qstr := "CREATE DATABASE " + s.InfluxDBName
	q := influxdb.NewQuery(qstr, s.InfluxDBName, "us")
	if resp, err := s.InfluxDB.Query(q); err == nil && resp.Error() == nil {
		log.Print(resp.Results)
	} else {
		if err != nil {
			log.Print(err)
		} else {
			log.Print(resp.Error())
		}
	}
	ks := s.Redis.Keys("*")
	log.Print(ks.Val())
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

	iu := os.Getenv("INFLUX_DB_URL")
	if iu == "" {
		iu = "http://localhost:8086"
	}

	ru := os.Getenv("INFLUX_DB_URL")
	if ru == "" {
		ru = "localhost:6379"
	}

	p := os.Getenv("SERVER_PORT")
	if p == "" {
		p = "8080"
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
	ic, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{Addr: iu})
	if err != nil {
		log.Print(err)
	}
	defer ic.Close()
	rc := redis.NewClient(&redis.Options{
		Addr: ru,
		Password: "",
		DB: 0,
	})
	defer rc.Close()
	s := Setting{
		SlackToken: t,
		SlackVerifyToken: vt,
		InfluxDBName: db,
		InfluxDB: ic,
		Redis: rc,
	}

	s.initialize()

	http.HandleFunc("/event", s.eventHandler)
	http.HandleFunc("/query", s.queryHandler)

	err0 := http.ListenAndServe(":" + p, nil)
	if err0 != nil {
		log.Fatal(err0)
	}
}

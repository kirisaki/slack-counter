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

	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"

	//"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"

)

type Setting struct {
	SlackToken string
	SlackVerifyToken string
	InfluxDB influxdb.Client
	InfluxDBName string
	DB *gorm.DB
}

type Team struct {
	gorm.Model
	TeamId string
	ChannelId string
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
	eventsAPIEvent, e := slackevents.ParseEvent(
		json.RawMessage(body),
		slackevents.OptionVerifyToken(
			&slackevents.TokenComparator{
				VerificationToken: s.SlackVerifyToken,
			},
		),
	)
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

	team := r.URL.Query().Get("team")
	if team == "" {
		resp, _ := json.Marshal(ErrorResponse{"empty team"})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(resp)
	}
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		resp, _ := json.Marshal(ErrorResponse{"empty channel"})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(resp)
	}


	qstr := "SELECT COUNT(\"user\") FROM \"activity\" WHERE \"team\"='"+ team + "'AND \"channel\"='" + channel + "'"
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

	s.DB.AutoMigrate(&Team{})
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

	inf := os.Getenv("INFLUX_DB_NAME")
	if inf == "" {
		log.Fatal("set INFLUX_DB_NAME")
	}

	ic, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{Addr: iu})
	if err != nil {
		log.Fatal(err)
	}
	defer ic.Close()

	db, err := gorm.Open("sqlite3", "test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	s := Setting{
		SlackToken: t,
		SlackVerifyToken: vt,
		InfluxDBName: inf,
		InfluxDB: ic,
		DB: db,
	}

	

	s.initialize()

	http.HandleFunc("/event", s.eventHandler)
	http.HandleFunc("/query", s.queryHandler)

	err0 := http.ListenAndServe(":" + p, nil)
	if err0 != nil {
		log.Fatal(err0)
	}
}

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

type setting struct {
	token string
	verifyToken string
}

func (s setting)eventHandler(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	body := buf.String()
	eventsAPIEvent, e := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: s.verifyToken}))
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

func (s setting) measure(ev *slackevents.MessageEvent){
	c, err := client.NewHTTPClient(client.HTTPConfig{Addr: "http://influxdb:8086"})
	if err != nil {
		log.Print(err)
	}
	defer c.Close()

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: "test",
		Precision: "u",
	})
	if err != nil {
		log.Fatal(err)
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


func main(){
	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		log.Fatal("set SLACK_TOKEN")
	}

	verifyToken := os.Getenv("SLACK_VERIFY_TOKEN")
	if token == "" {
		log.Fatal("set SLACK_VERIY_TOKEN")
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
	setting := setting{
		token: token,
		verifyToken: verifyToken,
	}
	http.HandleFunc("/event", setting.eventHandler)
	//	http.HandleFunc("/", makeDashboardHandler(user, pass))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"net/http"
	"log"
	//"time"
	"bytes"
	"encoding/json"
	"os"

	//_ "github.com/influxdata/influxdb1-client" 
	//client "github.com/influxdata/influxdb1-client/v2"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"

)
func makeEventHandler(token string, verifyToken string)(func (http.ResponseWriter, *http.Request)) {
	api := slack.New(token)
	return func (w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		body := buf.String()
		eventsAPIEvent, e := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: verifyToken}))
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
			case *slackevents.AppMentionEvent:
					log.Print("nyaan")
			api.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
			default:
				log.Print(ev)
			}
		}
	}
}


/*
func eventHandler(w http.ResponseWriter, r *http.Request){
	
	c, err := client.NewHTTPClient(client.HTTPConfig{Addr: "http://influxdb:8086"})
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

	log.Print("write measurement")
	er := c.Write(bp)
	if er != nil {
		log.Print(er)
	}
}
*/

func main(){
	token := os.Getenv("SLACK_TOKEN")
	if token == "" {
		log.Fatal("set SLACK_TOKEN")
	}
	verifyToken := os.Getenv("SLACK_VERIFY_TOKEN")
	if token == "" {
		log.Fatal("set SLACK_VERIY_TOKEN")
	}

	http.HandleFunc("/event", makeEventHandler(token, verifyToken))
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

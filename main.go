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
	"math"

	_ "github.com/influxdata/influxdb1-client" 
	influxdb "github.com/influxdata/influxdb1-client/v2"

	"github.com/nlopes/slack"
	"github.com/nlopes/slack/slackevents"

)

type Setting struct {
	Slack *slack.Client
	SlackToken string
	SlackVerifyToken string
	InfluxDB influxdb.Client
	InfluxDBName string
	TeamID string
	ChannelID string
}

type DailyActivity struct {
	Start int `json:"start"`
	Activity []int `json:"activity"`
}

type Activities struct {
	Activities []DailyActivity `json:"activities"`
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
		Database: s.InfluxDBName,
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

	duration, err := strconv.ParseInt(r.URL.Query().Get("duration"), 10, 64)
	if err != nil {
		log.Print(err)
		resp, _ := json.Marshal(ErrorResponse{"invalid query: " + r.URL.RawQuery})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(resp)
		return
	}
	start := end.Add(time.Hour * -24 * time.Duration(duration))
	log.Print(start)

	qstr := "SELECT COUNT(\"user\") FROM activity WHERE" +
		" \"team\"='" + s.TeamID + "' AND \"channel\"='" + s.ChannelID + "' " +
		"AND '" + start.Format(time.RFC3339) + "' <= time AND time <= '" + end.Format(time.RFC3339) +"' " +
		"GROUP BY time(1h)"
	log.Print(qstr)
	q := influxdb.NewQuery(qstr, s.InfluxDBName, "us")
	if resp, err := s.InfluxDB.Query(q); err == nil && resp.Error() == nil {
		results := []DailyActivity{}
		for _, r := range(resp.Results) {
			for _, s := range(r.Series) {
				acts := []int{}
				ts := 0
				for i, v := range(s.Values) {
					if i % 24 == 0 {
						if v0, ok := v[0].(json.Number); ok {
							x, _ := v0.Int64()
							ts = int(x) / 1000
						} else {
							log.Print("invalid type of timestamp")
							return
						}
					}
					if v1, ok := v[1].(json.Number); ok {
						x, _ := v1.Int64()
						acts = append(acts, int(x))
					} else {
						log.Print("invalid type of activity")
						return
					}
					if i % 24 == 23 {
						results = append(results, DailyActivity{
							Start: ts,
							Activity: acts,
						})
						acts = []int{}
					}
				}
			}
		}
		body, _ := json.Marshal(Activities{Activities: results})
		w.Write(body)
	} else {
		if err != nil {
			log.Print(err)
		} else {
			log.Print(resp.Error())
		}
	}
}

func (s Setting)initialize(){

	for i := 0; i < 5; i++ {
		time.Sleep(time.Second * time.Duration(math.Pow(2, float64(i))))
		_, _, err :=  s.InfluxDB.Ping(time.Second)
		if err == nil {
			break
		}
	}

	qstr0 := "CREATE DATABASE " + s.InfluxDBName
	q0:= influxdb.NewQuery(qstr0, s.InfluxDBName, "us")
	if resp, err := s.InfluxDB.Query(q0); err == nil && resp.Error() == nil {
		log.Print(resp.Results)
	} else {
		if err != nil {
			log.Print(err)
		} else {
			log.Print(resp.Error())
		}
	}

	qstr1 := "SELECT COUNT(\"user\") FROM \"activity\" WHERE \"team\"='" +
		s.TeamID + "'AND \"channel\"='" + s.ChannelID + "'"
	q1 := influxdb.NewQuery(qstr1, s.InfluxDBName, "us")
	i := 0
	if resp, err := s.InfluxDB.Query(q1); err == nil && resp.Error() == nil {
		for _, r := range(resp.Results) {
			for _, s := range(r.Series) {
				i += len(s.Values)
			}
		}
	} else {
		if err != nil {
			log.Print(err)
		} else {
			log.Print(resp.Error())
		}
	}

	if i < 1000 {
		log.Print("insert initial measurements from channel.history")
		body := bytes.NewBufferString("token=" + s.SlackToken + "&channel=" + s.ChannelID)
		resp, err := http.Post("https://slack.com/api/channels.history", "application/x-www-form-urlencoded", body)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		var hist struct {
			Messages []struct {
				User string `json:"user"`
				Timestamp string `json:"ts"`
			}
		}
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&hist); err != nil {
			log.Fatal(err)
		}

		bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
			Database: s.InfluxDBName,
			Precision: "us",
		})
		if err != nil {
			log.Fatal(err)
		}

		for _, m := range(hist.Messages) {
			ut := strings.Split(m.Timestamp, ".")
			if len(ut) != 2 {
				log.Fatal("invalid timestamp")
			}
			sec, err0 := strconv.ParseInt(ut[0], 10, 64)
			nsec, err1 := strconv.ParseInt(ut[1], 10, 64)
			if err0 != nil || err1 != nil {
				log.Fatal("failed parsing number")
				return
			}
			ts := time.Unix(sec, nsec)
			tags := map[string]string{"team": s.TeamID}
			fields := map[string]interface{}{
				"user": m.User,
				"channel": s.ChannelID,
			}

			pt, err := influxdb.NewPoint("activity", tags, fields, ts)
			if err != nil {
				log.Print(err)
			}
			bp.AddPoint(pt)
		}
		er := s.InfluxDB.Write(bp)
		if er != nil {
			log.Fatal(er)
		}
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
	team := os.Getenv("TEAM_ID")
	if team == "" {
		log.Fatal("set TEAM_ID")
	}
	channel := os.Getenv("CHANNEL_ID")
	if channel == "" {
		log.Fatal("set CHANNEL_ID")
	}

	ic, err := influxdb.NewHTTPClient(influxdb.HTTPConfig{Addr: iu})
	if err != nil {
		log.Fatal(err)
	}
	defer ic.Close()

	s := Setting{
		Slack: slack.New(t),
		SlackToken: t,
		SlackVerifyToken: vt,
		InfluxDBName: inf,
		InfluxDB: ic,
		TeamID: team,
		ChannelID: channel,
	}

	

	s.initialize()

	http.HandleFunc("/event", s.eventHandler)
	http.HandleFunc("/query", s.queryHandler)
	http.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-type", "text/html")
		http.ServeFile(w, r, "/client/index.html")
	})
	http.HandleFunc("/main.js", func (w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-type", "application/javascript")
		http.ServeFile(w, r, "/client/main.js")
	})

	err0 := http.ListenAndServe(":" + p, nil)
	if err0 != nil {
		log.Fatal(err0)
	}
}

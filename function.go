package function

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"os"
)

var webhookURL = os.Getenv("WEBHOOK_URL")

type AlertLevel int

const (
	Danger AlertLevel = iota
	Warn
	Health
)

var colors = map[AlertLevel]string{
	Danger: "#fc2f2f",
	Warn:   "#ffcc14",
	Health: "#27d871",
}

// Grafana alert json:
// {
//   "dashboardId":1,
//   "evalMatches":[
//     {
//       "value":1,
//       "metric":"Count",
//       "tags":{}
//     }
//   ],
//   "imageUrl":"https://grafana.com/assets/img/blog/mixed_styles.png",
//   "message":"Notification Message",
//   "orgId":1,
//   "panelId":2,
//   "ruleId":1,
//   "ruleName":"Panel Title alert",
//   "ruleUrl":"http://localhost:3000/d/hZ7BuVbWz/test-dashboard?fullscreen\u0026edit\u0026tab=alert\u0026panelId=2\u0026orgId=1",
//   "state":"alerting",
//   "tags":{
//     "tag name":"tag value"
//   },
//   "title":"[Alerting] Panel Title alert"
// }

type Alert struct {
	DashboardID int64                    `json:"dashboardId"`
	EvalMatches []map[string]interface{} `json:"evalMatches"`
	ImageURL    string                   `json:"imageUrl"`
	Message     string                   `json:"message"`
	OrgID       int64                    `json:"orgId"`
	PanelID     int64                    `json:"panelID"`
	RuleID      int64                    `json:"ruleID"`
	RuleName    string                   `json:"ruleName"`
	RuleURL     string                   `json:"ruleUrl"`
	State       string                   `json:"state"`
	Tags        map[string]string        `json:"tags"`
	Title       string                   `json:"title"`
}

//{
//  "text": "",
//  "cards": [
//    {
//      "sections": [
//        {
//          "widgets": [
//            {
//              "textParagraph": {
//                "text": "<b>Roses</b> are <font color=\"#ff0000\">red</font>,<br><i>Violets</i> are <font color=\"#0000ff\">blue</font>"
//              }
//            }
//          ]
//        }
//      ]
//    }
//  ]
//}

type GChatParam struct {
	Text  string `json:"text"`
	Cards []Card `json:"cards"`
}

type Card struct {
	Sections []Section `json:"sections"`
}

type Section struct {
	Widgets []Widget `json:"widgets"`
}

type Widget map[string]interface{}

func NotifyGrafanaAlertToGChat(w http.ResponseWriter, r *http.Request) {
	alert := Alert{}

	if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
		log.Printf("[error] decode alert error, %s.", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Printf("[alert log] %#v", alert)

	var mention = "<users/all>"
	var color = colors[Warn]
	if strings.HasPrefix(alert.RuleName, "[DANGER]") {
		mention = "<users/all>"
		color = colors[Danger]
	}
	if alert.State == "ok" {
		color = colors[Health]
	}

	var textDetail string
	for _, eval := range alert.EvalMatches {
		textDetail = fmt.Sprintf("metric: %f, value: %s\n", eval["value"], eval["metric"])
	}

	text := fmt.Sprintf("<font color=\"%s\">%s</font>\n%s",
		color,
		alert.Title,
		textDetail,
	)

	headWidgets := []Widget{
		Widget{
			"textParagraph": map[string]interface{}{
				"text": text,
			},
		},
		Widget{
			"buttons": []map[string]interface{}{
				map[string]interface{}{
					"textButton": map[string]interface{}{
						"text": "URL",
						"onClick": map[string]interface{}{
							"openLink": map[string]interface{}{
								"url": alert.RuleURL,
							},
						},
					},
				},
			},
		},
		Widget{
			"image": map[string]interface{}{
				"imageUrl": alert.ImageURL,
			},
		},
	}

	keyValueSecWidgets := []Widget{
		Widget{
			"keyValue": map[string]interface{}{
				"topLabel":         "State",
				"content":          alert.State,
				"contentMultiline": true,
			},
		},
		Widget{
			"keyValue": map[string]interface{}{
				"topLabel":         "Message",
				"content":          alert.Message,
				"contentMultiline": true,
			},
		},
	}

	params := GChatParam{
		Text: mention,
		Cards: []Card{
			{
				Sections: []Section{
					{
						Widgets: headWidgets,
					},
					{
						Widgets: keyValueSecWidgets,
					},
				},
			},
		},
	}

	b, err := json.Marshal(params)
	if err != nil {
		log.Printf("[error] marshal error, %s.", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(b))
	if err != nil {
		log.Printf("[error] new request error, %s.", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[error] post form error, %s.", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("%s", err.Error())

	} else {
		log.Printf("[gchat response] %s", body)

	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(b); err != nil {
		log.Printf("[error] write error, %s.", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

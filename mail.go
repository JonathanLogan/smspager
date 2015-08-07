package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/JonathanLogan/smtpclient"
	"github.com/sloonz/go-iconv"
)

func makeMail(sender, recipient, message string) string {
	return "From: <" + sender + ">\r\nTo: <" + recipient + ">\r\nSubject: " + message + "\r\n" +
		"Date: " + time.Now().Format(time.RFC1123Z) + "\r\n\r\n" +
		message + "\r\n"
}

func sendMail(route Route, message string) error {
	mail := makeMail(route.Sender, route.Recipient, message)
	mailclient := smtpclient.MailClient{
		User:      route.User,
		Password:  route.Password,
		Port:      route.Port,
		SmartHost: route.Server,
	}
	return mailclient.SendMail(route.Recipient, route.Sender, []byte(mail))
	// return smtp.SendMail(server, nil, sender, []string{recipient}, []byte(mail))
}

// Route contains a route
type Route struct {
	Selector   string
	Sender     string
	Recipient  string
	User       string
	Password   string
	Server     string
	Port       int
	MaxLength  int
	WithSender int
}

// Router implements a router
type Router struct {
	routes map[string][]Route
}

func (r *Router) SendMail(senderNum, message string) {
	recipient, newmessage := splitMessage(message)
	fmt.Printf("Sending from <%s> to <%s>: \"%s\"\n", senderNum, recipient, message)
	routeS := r.Route(recipient)
	if routeS == nil {
		fmt.Printf("Could not route: %s\n", message)
	}
	for _, route := range routeS {
		rmessage := newmessage
		if route.WithSender == 1 {
			rmessage = "<" + senderNum + ">: " + newmessage
		}
		messageParts := mkMultiPart(rmessage, route.MaxLength)
		for _, multimessage := range messageParts {
			err := sendMail(route, multimessage)
			if err != nil {
				fmt.Printf("Send error (%s): %s\n", message, err)
			}
		}
	}
}

func mkMultiPart(message string, maxlength int) []string {
	var ret []string
	if len(message) <= maxlength {
		ret = append(ret, message)
		return ret
	}
	maxlength = maxlength - 4
	parts := len(message) / maxlength
	if len(message)%maxlength > 0 {
		parts++
	}
	for i := 0; i < parts; i++ {
		var cut string
		if (i+1)*maxlength < len(message) {
			cut = message[i*maxlength : (i+1)*maxlength]
		} else {
			cut = message[i*maxlength:]
		}
		ret = append(ret, fmt.Sprintf("%d/%d %s", i+1, parts, cut))
	}
	return ret
}

func splitMessage(message string) (recipient, newmessage string) {
	m := strings.Trim(message, " \t")
	cpos := strings.Index(m, ":")
	spos := strings.Index(m, " ")
	rec := ""
	if cpos > 0 {
		if spos > 0 && spos < cpos {
			rec = ""
		} else {
			rec = m[:cpos]
			m = m[cpos:]
		}
	}
	m2, err := iconv.Conv(m, "ASCII//TRANSLIT", "UTF-8")
	if err != nil {
		return rec, m
	}
	return rec, m2
}

func (r *Router) Route(rec string) []Route {
	rec = strings.ToLower(rec)
	if d, ok := r.routes[rec]; ok {
		return d
	}
	if d, ok := r.routes[""]; ok {
		return d
	}
	return nil
}

// LoadRouter loads a router
func LoadRouter(routes []byte) *Router {
	var routesI []Route
	err := json.Unmarshal(routes, &routesI)
	if err != nil {
		panic(err)
	}
	r := new(Router)
	r.routes = make(map[string][]Route)
	for _, oneRoute := range routesI {
		if _, ok := r.routes[oneRoute.Selector]; !ok {
			r.routes[oneRoute.Selector] = make([]Route, 0)
		}
		r.routes[oneRoute.Selector] = append(r.routes[oneRoute.Selector], oneRoute)
	}
	return r
}

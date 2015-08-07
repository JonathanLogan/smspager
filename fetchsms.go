package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tarm/serial"
)

// ChatLine contains a single chat line
type ChatLine struct {
	Method   string
	Command  string
	Response string
	Enforce  bool
}

var setupChat = []ChatLine{
	ChatLine{
		Method:   "Send",
		Command:  "ATE0",
		Response: "OK",
		Enforce:  false,
	},
	ChatLine{
		Method:   "Send",
		Command:  "AT^CURC=0",
		Response: "OK",
		Enforce:  true,
	},
	ChatLine{
		Method:   "Send",
		Command:  "AT+CPMS=\"SM\"",
		Response: "+CPMS: 0,30,0,30,0,30\r\n\r\nOK",
		Enforce:  false,
	},
	ChatLine{
		Method:   "Send",
		Command:  "AT+CMGF=1",
		Response: "OK",
		Enforce:  true,
	},
	ChatLine{
		Method:   "Send",
		Command:  "AT+CNMI=1,1,0,1,0",
		Response: "OK",
		Enforce:  true,
	},
	ChatLine{
		Method:   "Wait",
		Command:  "",
		Response: "+CMTI:",
		Enforce:  true,
	},
}

// SMSClient implements an SMSClient
type SMSClient struct {
	Port    *serial.Port
	Routing *Router
}

// New returns an SMSClient
func New(name string, baud int, routes []byte) (*SMSClient, error) {
	var err error
	sc := new(SMSClient)
	sc.Port, err = serial.OpenPort(&serial.Config{Name: name, Baud: baud, ReadTimeout: 3})
	if err != nil {
		return nil, err
	}
	sc.Routing = LoadRouter(routes)
	return sc, nil
}

// SendCommand sends a command
func (sc *SMSClient) SendCommand(command string) (resp string, err error) {
	if len(command) > 0 {
		sendString := []byte(command + "\r\n")
		_, err = sc.Port.Write(sendString)
		if err != nil {
			return "", err
		}
	}
	buf := make([]byte, 4096)
	n, err := sc.Port.Read(buf)
	if err != nil {
		return "", err
	}
	return strings.Trim(string(buf[:n]), " \t\r\n"), nil
}

// Wait for a line containing match
func (sc *SMSClient) Wait(match string) (resp string) {
	for {
		buf := make([]byte, 4096)
		n, err := sc.Port.Read(buf)
		if err != nil {
			if err == io.EOF {
				time.Sleep(time.Second / 4)
			}
		}
		res := strings.Trim(string(buf[:n]), " \t\r\n")
		if len(res) >= len(match) && match == res[:len(match)] {
			return res
		}
	}
}

func (sc *SMSClient) ForwardSMS() error {
	for _, e := range setupChat {
		if e.Method == "Send" {
			resp, err := sc.SendCommand(e.Command)
			if err != nil {
				return err
			}
			if e.Enforce {
				if resp != e.Response {
					return fmt.Errorf("%s -> '%s' != '%s'", e.Command, e.Response, resp)
				}
			}
		} else if e.Method == "Wait" {
			for {
				resp := parseCMTI(sc.Wait(e.Response))
				message, err := sc.SendCommand("AT+CMGR=" + resp)
				if err == nil {
					sc.SendCommand("AT+CMGD=" + resp)
					// fmt.Printf("Message received: %s\n", message)
					sender, text := parseMessage(message)
					sc.Routing.SendMail(sender, text)
				}
			}
		}
	}
	return sc.Port.Close()
}

func parseMessage(str string) (sender, message string) {
	parts := strings.Split(str, "\r\n")
	header := parts[0]
	headerParts := strings.Split(header, "\"")
	if len(headerParts) >= 4 {
		sender = headerParts[3]
	}
	if len(parts) < 4 {
		return sender, ""
	}
	message = strings.Join(parts[1:len(parts)-2], " ")
	return sender, message
}

func parseCMTI(str string) string {
	pos := strings.Index(str, ",")
	if pos > 0 && pos+1 < len(str) {
		return str[pos+1:]
	}
	return ""
}

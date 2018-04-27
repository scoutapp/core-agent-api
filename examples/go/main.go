package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

type CoreAgent struct{}

type CoreAgentMessage interface {
	MarshalJSON() ([]byte, error)
}

type Register struct {
	App     string
	Key     string
	Version string
}

type Request struct {
	RequestId string
}

type Span struct {
	RequestId string
	SpanId    string
	Operation string
}

type StartRequest Request
type FinishRequest Request
type StartSpan Span
type StopSpan Span

func main() {
	http.Handle("/hello", scoutHandler(helloHandler))
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello %s!", r.URL.Path[1:])
}

func scoutHandler(h http.HandlerFunc) http.HandlerFunc {
	coreAgent := &CoreAgent{}
	coreAgent.Send(Register{App: os.Getenv("SCOUT_NAME"), Key: os.Getenv("SCOUT_KEY")})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestId, _ := newUUID()
		coreAgent.Send(StartRequest{RequestId: requestId})

		spanId, _ := newUUID()
		coreAgent.Send(StartSpan{RequestId: requestId, SpanId: spanId, Operation: "Controller/" + r.URL.Path})
		h.ServeHTTP(w, r)
		coreAgent.Send(StopSpan{RequestId: requestId, SpanId: spanId})
		coreAgent.Send(FinishRequest{RequestId: requestId})
	})
}

func (ca *CoreAgent) Send(msg CoreAgentMessage) {
	c, err := net.Dial("unix", "/tmp/core-agent.sock")
	if err != nil {
		return
	}
	defer c.Close()

	jsonBytes, err := msg.MarshalJSON()

	if err != nil {
		return
	}
	binary.Write(c, binary.LittleEndian, len(jsonBytes))
	c.Write(jsonBytes)
}

func (r Register) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("{\"Register\":{\"app\":\"%s\",\"key\":\"%s\",\"version\":\"1.0\"}}", r.App, r.Key)), nil
}

func (req StartRequest) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("{\"StartRequest\":{\"request_id\":\"%s\"}}", req.RequestId)), nil
}

func (req FinishRequest) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("{\"FinishRequest\":{\"request_id\":\"%s\"}}", req.RequestId)), nil
}

func (s StartSpan) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("{\"StartSpan\":{\"request_id\":\"%s\",\"span_id\":\"%s\",\"operation\":\"%s\"}}", s.RequestId, s.SpanId, s.Operation)), nil
}

func (s StopSpan) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("{\"StopSpan\":{\"request_id\":\"%s\",\"span_id\":\"%s\"}}", s.RequestId, s.SpanId)), nil
}

// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

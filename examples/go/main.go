/*
 * This example traces a Go http.HandlerFunc request via the Scout Core Agent API.
 * Traces and metrics appear at Scoutapp.com.
 *
 * Quickstart:
 * 1. Download and extract a core agent binary.
 *    URL for OSX: http://s3-us-west-1.amazonaws.com/scout-public-downloads/apm_core_agent/release/scout_apm_core-latest-x86_64-apple-darwin.tgz
 * 2. Start the core agent: `~/Downloads/scout_apm_core-latest-x86_64-apple-darwin/core-agent start --socket /tmp/core-agent.sock`
 * 3. Build this Go file: `go build`
 * 4. Run the app, passing a name for your app and Scout agent key as env vars: `env SCOUT_NAME="YOUR APP" SCOUT_KEY="YOUR KEY" ./go`
 * 5. Hit the listening http endpoint: `curl localhost:4000/hello`
 */
package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
)

// The global variable holding a reference to the CoreAgent struct
var coreAgent *CoreAgent

func main() {
	// Communicate w/the Core Agent via this socket path
	coreAgent = NewCoreAgent("/tmp/core-agent.sock")
	err := coreAgent.Open()
	if err != nil {
		log.Println("Unable to open socket to Core Agent:", err)
	}
	coreAgent.Register()

	http.HandleFunc("/hello", helloHandler)
	log.Fatal(http.ListenAndServe(":4000", nil))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	// Start the the trace of the transaction
	requestId, _ := newUUID()
	coreAgent.Send(StartRequest{RequestId: requestId})

	// Create the span ... traces are composed of spans.
	spanId, _ := newUUID()
	// For now, at least of the spans in a transaction must start with 'Controller'
	coreAgent.Send(StartSpan{RequestId: requestId, SpanId: spanId, Operation: "Controller" + r.URL.Path})

	// The actual work to instrument
	fmt.Fprint(w, "Hello World!")

	// Stop the span and finish the request
	coreAgent.Send(StopSpan{RequestId: requestId, SpanId: spanId})
	coreAgent.Send(FinishRequest{RequestId: requestId})
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

/////////////////////////////////////////////////
//  The Core Agent related structs and methods //
/////////////////////////////////////////////////

type CoreAgent struct {
	sync.Mutex
	Connected  bool
	SocketPath string
	Socket     net.Conn
}

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

func NewCoreAgent(socketPath string) *CoreAgent {
	return &CoreAgent{SocketPath: socketPath}
}

func (ca *CoreAgent) Open() error {
	ca.Lock()
	defer ca.Unlock()

	c, err := net.Dial("unix", ca.SocketPath)
	if err != nil {
		ca.Connected = false
		return err
	}
	ca.Socket = c
	ca.Connected = true
	return nil
}

func (ca *CoreAgent) Close() error {
	if ca.Connected {
		ca.Socket.Close()
		ca.Connected = false
	}
	return nil
}

func (ca *CoreAgent) Register() error {
	return ca.Send(Register{App: os.Getenv("SCOUT_NAME"), Key: os.Getenv("SCOUT_KEY")})
}

// Sends a message (a CoreAgentMessage interface) to the core agent via a Socket
func (ca *CoreAgent) Send(msg CoreAgentMessage) error {
	ca.Lock()
	defer ca.Unlock()

	if !ca.Connected {
		return fmt.Errorf("Core Agent not connected")
	}

	jsonBytes, err := msg.MarshalJSON()
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(len(jsonBytes)))
	ca.Socket.Write(buf.Bytes())
	ca.Socket.Write(jsonBytes)
	return nil
}

func (r Register) MarshalJSON() ([]byte, error) {
	json := []byte(fmt.Sprintf("{\"Register\":{\"app\":\"%s\",\"key\":\"%s\",\"api_version\":\"1.0\"}}", r.App, r.Key))
	return json, nil
}

func (req StartRequest) MarshalJSON() ([]byte, error) {
	json := []byte(fmt.Sprintf("{\"StartRequest\":{\"request_id\":\"%s\"}}", req.RequestId))
	return json, nil
}

func (req FinishRequest) MarshalJSON() ([]byte, error) {
	json := []byte(fmt.Sprintf("{\"FinishRequest\":{\"request_id\":\"%s\"}}", req.RequestId))
	return json, nil
}

func (s StartSpan) MarshalJSON() ([]byte, error) {
	json := []byte(fmt.Sprintf("{\"StartSpan\":{\"request_id\":\"%s\",\"span_id\":\"%s\",\"operation\":\"%s\"}}", s.RequestId, s.SpanId, s.Operation))
	return json, nil
}

func (s StopSpan) MarshalJSON() ([]byte, error) {
	json := []byte(fmt.Sprintf("{\"StopSpan\":{\"request_id\":\"%s\",\"span_id\":\"%s\"}}", s.RequestId, s.SpanId))
	return json, nil
}

package transaction

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stefankopieczek/gossip/base"
	"github.com/stefankopieczek/gossip/log"
	"github.com/stefankopieczek/gossip/parser"
	"github.com/stefankopieczek/gossip/transport"
)

var c_SERVER string = "192.168.1.41:5060"
var c_CLIENT string = "192.168.1.39:5060"
var c_LISTEN string = "192.168.1.41:5060"
var userid string = "user91"
var testLog *log.Logger = log.New(os.Stdout, ">>> ", 0)

func TestAAAASetup(t *testing.T) {
	log.SetDefaultLogLevel(log.WARN)
	testLog.Level = log.INFO
}

// func TestSendInviteUDP(t *testing.T) {
// 	invite, err := request([]string{
// 		"INVITE sip:joe@bloggs.com SIP/2.0",
// 		"Via: SIP/2.0/UDP " + c_CLIENT + ";branch=z9hG4bK776asdhds",
// 		"",
// 		"",
// 	})
// 	assertNoError(t, err)

// 	test := transactionTest{actions: []action{
// 		&clientSend{invite},
// 		// &serverRecv{invite},
// 	}}
// 	test.Execute(t)
// }

func TestSendRegisterUDP(t *testing.T) {
	register, err := request([]string{
		"REGISTER sip:ims.hom SIP/2.0",
		"Via: SIP/2.0/UDP " + c_CLIENT + ";rport;branch=z9hG4bK996329494",
		"Route: <sip:pcscf.ims.hom;lr>",
		"From: <sip:"+userid+"@ims.hom>;tag=1224392795",
		"To: <sip:"+userid+"@ims.hom>",
		"Call-ID: 1379275388",
		"CSeq: 1 REGISTER",
		"Contact: <sip:"+userid+"@192.168.1.39:5060>",
		"Authorization: Digest username=\""+userid+"@ims.hom\", realm=\"ims.hom\", nonce=\" \", uri=\"sip:ims.hom\", response=\" \"",
		"Max-Forwards: 70",
		"User-Agent: eXosip/3.1.0",
		"Expires: 600000",
		"Supported: path",
		"Supported: gruu",
		"Content-Length: 0",
		"",
		"",
	})
	assertNoError(t, err)
	// ok, err := response([]string{
	// 	"SIP/2.0 200 OK",
	// 	"CSeq: 1 INVITE",
	// 	"Via: SIP/2.0/UDP " + c_SERVER + ";branch=z9hG4bK996329494",
	// 	"",
	// 	"",
	// })

	fmt.Println("Now proceeding further")
	test := transactionTest{actions: []action{
		&clientSend{register},
		// &clientRecv{ok},
	}}
	test.Execute(t)
}

// func TestReceiveOKUDP(t *testing.T) {
// 	invite, err := request([]string{
// 		"INVITE sip:joe@bloggs.com SIP/2.0",
// 		"CSeq: 1 INVITE",
// 		"Via: SIP/2.0/UDP " + c_CLIENT + ";branch=z9hG4bK776asdhds",
// 		"",
// 		"",
// 	})
// 	assertNoError(t, err)

// 	ok, err := response([]string{
// 		"SIP/2.0 200 OK",
// 		"CSeq: 1 INVITE",
// 		"Via: SIP/2.0/UDP " + c_SERVER + ";branch=z9hG4bK776asdhds",
// 		"",
// 		"",
// 	})
// 	assertNoError(t, err)

// 	test := transactionTest{actions: []action{
// 		&clientSend{invite},
// 		&serverRecv{invite},
// 		&serverSend{ok},
// 		&clientRecv{ok},
// 	}}
// 	test.Execute(t)
// }

type action interface {
	Act(test *transactionTest) error
}

type transactionTest struct {
	actions    []action
	client     *Manager
	server     *transport.Manager
	serverRecv chan base.SipMessage
	lastTx     *ClientTransaction
}

func (test *transactionTest) Execute(t *testing.T) {
	fmt.Println("Starting Execute")
	var err error
	test.client, err = NewManager("udp", c_CLIENT)
	assertNoError(t, err)
	defer test.client.Stop()

	test.client_l, err = transport.NewManager("udp")
	assertNoError(t, err)
	defer test.client_l.Stop()
	test.client_lRecv = test.client_l.GetChannel()
	test.client_l.Listen(c_LISTEN)

	for _, actn := range test.actions {
		testLog.Debug("Performing action %v", actn)
		actn.Act(test)
	}
}

type clientSend struct {
	msg *base.Request
}

func (actn *clientSend) Act(test *transactionTest) error {
	test.lastTx = test.client.Send(actn.msg, c_SERVER)

	fmt.Println("Checking Responses")
	responses := test.lastTx.Responses()
	select {
	case response, ok := <-responses:
		if !ok {
			fmt.Println("Response channel prematurely closed")
			return fmt.Errorf("Response channel prematurely closed")
		} else {
			fmt.Println("Response from server:\n %s", response.String())
			return nil
		}
	case <-time.After(time.Duration(2)*time.Second):
		fmt.Println("Timed out waiting for response")
		return fmt.Errorf("Timed out waiting for response")
	}

	return nil
}

type serverSend struct {
	msg base.SipMessage
}

func (actn *serverSend) Act(test *transactionTest) error {
	return test.server.Send(c_CLIENT, actn.msg)
}

type clientRecv struct {
	expected *base.Response
}

func (actn *clientRecv) Act(test *transactionTest) error {
	fmt.Println("Checking Responses")
	responses := test.lastTx.Responses()
	select {
	case response, ok := <-responses:
		if !ok {
			fmt.Println("Response channel prematurely closed")
			return fmt.Errorf("Response channel prematurely closed")
		} else if response.String() != actn.expected.String() {
			fmt.Println("Unexpected response:\n%s", response.String())
			return fmt.Errorf("Unexpected response:\n%s", response.String())
		} else {
			fmt.Println("Response from server:\n %s", response.String())
			return nil
		}
	case <-time.After(time.Duration(2)*time.Second):
		fmt.Println("Timed out waiting for response")
		return fmt.Errorf("Timed out waiting for response")
	}
}

type serverRecv struct {
	expected base.SipMessage
}

func (actn *serverRecv) Act(test *transactionTest) error {
	select {
	case msg, ok := <-test.serverRecv:
		if !ok {
			return fmt.Errorf("Server receive channel prematurely closed")
		} else if msg.String() != actn.expected.String() {
			return fmt.Errorf("Unexpected message arrived at server:\n%s", msg.String())
		} else {
			return nil
		}
	case <-time.After(time.Duration(2)*time.Second):
		return fmt.Errorf("Timed out waiting for message on server")
	}
}

func assert(t *testing.T, b bool, msg string) {
	if !b {
		t.Errorf(msg)
	}
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
}

func message(rawMsg []string) (base.SipMessage, error) {
	return parser.ParseMessage([]byte(strings.Join(rawMsg, "\r\n")))
}

func request(rawMsg []string) (*base.Request, error) {
	msg, err := message(rawMsg)
	fmt.Println(strings.Join(rawMsg, "\r\n"))
	fmt.Println([]byte(strings.Join(rawMsg, "\r\n")))
	if err != nil {
		return nil, err
	}

	switch msg.(type) {
	case *base.Request:
		return msg.(*base.Request), nil
	default:
		return nil, fmt.Errorf("%s is not a request", msg.Short)
	}
}

func response(rawMsg []string) (*base.Response, error) {
	msg, err := message(rawMsg)

	if err != nil {
		return nil, err
	}

	switch msg.(type) {
	case *base.Response:
		return msg.(*base.Response), nil
	default:
		return nil, fmt.Errorf("%s is not a response", msg.Short)
	}
}

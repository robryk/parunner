package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"time"
)

var traceCommunications = flag.Bool("trace_comm", false, "Wypisz na standardowe wyjście diagnostyczne informację o każdej wysłanej i odebranej wiadomości")

const magic = 1736434764
const recvResponseMagic = magic + 1
const sendOpType = 3
const recvOpType = 4

type header struct {
	Magic     uint32
	NodeCount int32
	NodeID    int32
}

type recvResponse struct {
	RecvResponseMagic uint32
	SourceID          int32
	Length            int32
	// Message []byte
}

type recvHeader struct {
	// OpType byte
	SourceID int32
	Time     int32 // milliseconds
}

type sendHeader struct {
	// OpType byte
	TargetID int32
	Time     int32 // milliseconds
	Length   int32
	// Message []byte
}

const MessageCountLimit = 1000
const MessageSizeLimit = 8 * 1024 * 1024

type Message struct {
	Source   int
	Target   int
	SendTime time.Duration
	Message  []byte
}

var ErrMessageCount = fmt.Errorf("przekroczony limit (%d) liczby wysłanych wiadomości", MessageCountLimit)
var ErrMessageSize = fmt.Errorf("przekroczony limit (%d bajtów) sumarycznego rozmiaru wysłanych wiadomości", MessageSizeLimit)

func writeMessage(w io.Writer, message *Message) error {
	rr := recvResponse{
		RecvResponseMagic: recvResponseMagic,
		SourceID:          int32(message.Source),
		Length:            int32(len(message.Message)),
	}
	if err := binary.Write(w, binary.LittleEndian, &rr); err != nil {
		return err
	}
	if n, err := w.Write(message.Message); n < len(message.Message) {
		if err == nil {
			err = io.ErrShortWrite
		}
		return err
	}
	return nil
}

func writeHeader(w io.Writer, id int, instanceCount int) error {
	h := header{
		Magic:     magic,
		NodeCount: int32(instanceCount),
		NodeID:    int32(id),
	}
	return binary.Write(w, binary.LittleEndian, &h)
}

const (
	requestSend = iota
	requestRecv
	requestRecvAny
	// requestNop
)

type request struct {
	requestType int
	time        time.Duration

	// for requestSend:
	destination int
	message     []byte

	// for requestRecv:
	source int
}

func (req request) Describe() string {
	switch req.requestType {
	case requestSend:
		return fmt.Sprintf("wysłanie wiadomości (%d bajtów) do instancji %d", len(req.message), req.destination)
	case requestRecv:
		return fmt.Sprintf("odbieranie wiadomości od instancji %d", req.source)
	case requestRecvAny:
		return "odbieranie dowolnej wiadomości"
	default:
		return fmt.Sprintf("nieznane żądanie typu %d", req.requestType)
	}
}

func (req request) hasResponse() bool {
	switch req.requestType {
	case requestRecv:
		return true
	case requestRecvAny:
		return true
	default:
		return false
	}
}

type response struct {
	message *Message
}

func readRequest(r io.Reader) (*request, error) {
	var opType [1]byte
	if _, err := r.Read(opType[:]); err != nil {
		return nil, err
	}
	switch opType[0] {
	case sendOpType:
		var sh sendHeader
		if err := binary.Read(r, binary.LittleEndian, &sh); err != nil {
			return nil, err
		}
		if sh.Length < 0 || sh.Length > MessageSizeLimit {
			return nil, fmt.Errorf("invalid size of a message to be sent: %d", sh.Length)
		}
		if sh.TargetID < 0 || sh.TargetID >= MaxInstances {
			return nil, fmt.Errorf("invalid target instance in a send request: %d", sh.TargetID)
		}
		message := make([]byte, sh.Length)
		if _, err := io.ReadFull(r, message); err != nil {
			return nil, err
		}
		return &request{
			requestType: requestSend,
			time:        time.Duration(sh.Time) * time.Millisecond,
			destination: int(sh.TargetID),
			message:     message}, nil
	case recvOpType:
		var rh recvHeader
		if err := binary.Read(r, binary.LittleEndian, &rh); err != nil {
			return nil, err
		}
		if rh.SourceID < -1 || rh.SourceID >= MaxInstances {
			return nil, fmt.Errorf("invalid source instance in a receive request: %d", rh.SourceID)
		}
		if rh.SourceID == -1 {
			return &request{requestType: requestRecvAny, time: time.Duration(rh.Time) * time.Millisecond}, nil
		} else {
			return &request{requestType: requestRecv, time: time.Duration(rh.Time) * time.Millisecond, source: int(rh.SourceID)}, nil
		}
	default:
		return nil, fmt.Errorf("invalid operation type 0x%x", opType[0])
	}
}

func (i *Instance) communicate(r io.Reader, w io.Writer, reqCh chan<- *request, respCh <-chan *response) error {
	// TODO: Export this from the Instance
	i.TimeBlocked = time.Duration(0)
	// TODO: Figure out what errors should be returned from this function. We currently error if the instance fails to read the header (which is mitigated by delaying the closure of other ends of the pipes), for example.
	if err := writeHeader(w, i.ID, i.TotalInstances); err != nil {
		return err
	}
	for {
		req, err := readRequest(r)
		if err != nil {
			if err == io.EOF {
				//return nil
			}
			return err
		}
		req.time += i.TimeBlocked
		if *traceCommunications {
			log.Printf("W momencie %v instancja %d: %s", req.time, i.ID, req.Describe())
		}
		if req.requestType == requestSend {
			i.MessagesSent++
			if i.MessagesSent > MessageCountLimit {
				return ErrMessageCount
			}
			i.MessageBytesSent += len(req.message)
			if i.MessageBytesSent > MessageSizeLimit {
				return ErrMessageSize
			}
		}
		currentTime := req.time
		hasResponse := req.hasResponse()
		reqCh <- req
		if hasResponse {
			resp, ok := <-respCh
			if !ok {
				return fmt.Errorf("Received no response for a receive request")
			}
			if *traceCommunications {
				log.Printf("W momencie %v instancja %d odebrała wiadomość od instancji %d.", resp.message.SendTime, i.ID, resp.message.Source)
			}
			if resp.message.SendTime > currentTime {
				i.TimeBlocked += resp.message.SendTime - currentTime
			}
			if err := writeMessage(w, resp.message); err != nil {
				return err
			}
		}
	}
}

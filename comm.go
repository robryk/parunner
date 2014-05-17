package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
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
}

type sendHeader struct {
	// OpType byte
	TargetID int32
	Length   int32
	// Message []byte
}

const MessageCountLimit = 1000
const MessageSizeLimit = 8 * 1024 * 1024

type Message struct {
	Source  int
	Target  int
	Message []byte
}

func (i *Instance) PutMessage(message Message) {
	i.queues[message.Source].Put() <- message
}

func (i *Instance) sendMessage(targetID int, message []byte) error {
	i.messagesSent++
	if i.messagesSent > MessageCountLimit {
		return fmt.Errorf("przekroczony limit (%d) liczby wysłanych wiadomości", MessageCountLimit)
	}
	i.messageBytesSent += len(message)
	if i.messageBytesSent > MessageSizeLimit {
		return fmt.Errorf("przekroczony limit (%d bajtów) sumarycznego rozmiaru wysłanych wiadomości", MessageSizeLimit)
	}
	i.outgoingMessages <- Message{Source: i.id, Target: targetID, Message: message}
	return nil
}

func (i *Instance) receiveMessage(sourceID int) Message {
	// XXX unblocking when exiting
	message := (<-i.queues[sourceID].Get()).(Message)
	return message
}

func (i *Instance) receiveAnyMessage() Message {
	// XXX unblocking when exiting
	mq := <-i.selector
	message := (<-mq.Get()).(Message)
	return message
}

func (i *Instance) communicate(r io.Reader, w io.Writer) error {
	h := header{
		Magic:     magic,
		NodeCount: int32(i.totalInstances),
		NodeID:    int32(i.id),
	}
	if err := binary.Write(w, binary.LittleEndian, &h); err != nil {
		return err
	}
	for {
		var opType [1]byte
		if _, err := r.Read(opType[:]); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		switch opType[0] {
		case sendOpType:
			var sh sendHeader
			if err := binary.Read(r, binary.LittleEndian, &sh); err != nil {
				return err
			}
			if sh.Length < 0 || sh.Length > MessageSizeLimit {
				return fmt.Errorf("invalid size of a message to be sent: %d", sh.Length)
			}
			if sh.TargetID < 0 || sh.TargetID >= 100 {
				return fmt.Errorf("invalid target instance in a send request: %d", sh.TargetID)
			}
			if *traceCommunications {
				log.Printf("Instancja %d wysyła %d bajtów do instancji %d.", i.id, sh.Length, sh.TargetID)
			}
			message := make([]byte, sh.Length)
			if _, err := io.ReadFull(r, message); err != nil {
				return err
			}
			i.sendMessage(int(sh.TargetID), message)
		case recvOpType:
			var rh recvHeader
			if err := binary.Read(r, binary.LittleEndian, &rh); err != nil {
				return err
			}
			if rh.SourceID < -1 || rh.SourceID >= 100 {
				return fmt.Errorf("invalid source instance in a receive request: %d", rh.SourceID)
			}
			var message Message
			if rh.SourceID == -1 {
				if *traceCommunications {
					log.Printf("Instancja %d czeka na wiadomość od dowolnej innej instancji.", i.id)
				}
				message = i.receiveAnyMessage()
			} else {
				if *traceCommunications {
					log.Printf("Instancja %d czeka na wiadomość od instancji %d.", i.id, rh.SourceID)
				}
				message = i.receiveMessage(int(rh.SourceID))
			}
			if *traceCommunications {
				log.Printf("Instancja %d odebrała wiadomość od instancji %d.", i.id, message.Source)
			}
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
		default:
			return fmt.Errorf("invalid operation type %x", opType[0])
		}
	}
}

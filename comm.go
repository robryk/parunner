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
	NodeId    int32
}

type recvResponse struct {
	RecvResponseMagic uint32
	SourceId          int32
	Length            int32
	// Message []byte
}

type recvHeader struct {
	// OpType byte
	SourceId int32
}

type sendHeader struct {
	// OpType byte
	TargetId int32
	Length   int32
	// Message []byte
}

const MessageCountLimit = 1000
const MessageSizeLimit = 8 * 1024 * 1024

func (i *Instance) sendMessage(target int, message []byte) error {
	i.messagesSent++
	if i.messagesSent > MessageCountLimit {
		return fmt.Errorf("Przekroczony limit (%d) liczby wysłanych wiadomości", MessageCountLimit)
	}
	i.messageBytesSent += len(message)
	if i.messageBytesSent > MessageSizeLimit {
		return fmt.Errorf("Przekroczony limit (%d bajtów) sumarycznego rozmiaru wysłanych wiadomości", MessageSizeLimit)
	}
	return nil
}

func (i *Instance) communicate(r io.ReadCloser, w io.WriteCloser) error {
	defer w.Close()
	defer r.Close()
	h := header{
		Magic:     magic,
		NodeCount: int32(len(i.instances)),
		NodeId:    int32(i.id),
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
				return fmt.Errorf("Invalid size of a message to be sent: %d", sh.Length)
			}
			if sh.TargetId < 0 || sh.TargetId >= 100 {
				return fmt.Errorf("Invalid target instance in a send request: %d", sh.TargetId)
			}
			if *traceCommunications {
				log.Printf("Instancja %d wysyła %d bajtów do instancji %d.", i.id, sh.Length, sh.TargetId)
			}
			message := make([]byte, sh.Length)
			if _, err := io.ReadFull(r, message); err != nil {
				return err
			}
			i.sendMessage(int(sh.TargetId), message)
		case recvOpType:
			var rh recvHeader
			if err := binary.Read(r, binary.LittleEndian, &rh); err != nil {
				return err
			}
			if rh.SourceId < -1 || rh.SourceId >= 100 {
				return fmt.Errorf("Invalid source instance in a receive request: %d", rh.SourceId)
			}
			if rh.SourceId == -1 {
				if *traceCommunications {
					log.Printf("Instancja %d czeka na wiadomość od dowolnej innej instancji.", i.id)
				}
			} else {
				if *traceCommunications {
					log.Printf("Instancja %d czeka na wiadomość od instancji %d.", i.id, rh.SourceId)
				}
			}
			// receive message
			rr := recvResponse{
				RecvResponseMagic: recvResponseMagic,
				SourceId:          0,
				Length:            5,
			}
			message := []byte("abcde")
			if err := binary.Write(w, binary.LittleEndian, &rr); err != nil {
				return err
			}
			if n, err := w.Write(message); n < len(message) {
				if err == nil {
					err = io.ErrShortWrite
				}
				return err
			}
		default:
			return fmt.Errorf("Invalid operation type %x", opType[0])
		}
	}
}

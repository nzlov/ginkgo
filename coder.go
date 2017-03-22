package ginkgo

import (
	"reflect"

	"github.com/Sirupsen/logrus"
	hio "github.com/hprose/hprose-golang/io"
)

type coderMessageType int

const (
	coderMessageType_Unknown = iota
	coderMessageType_Register
	coderMessageType_Binary
	coderMessageType_Invoke
	coderMessageType_InvokeRecive
	coderMessageType_InvokeNameError
)

func (c coderMessageType) String() (r string) {
	switch c {
	case coderMessageType_Unknown:
		r = "coderMessageType_Unknown"
	case coderMessageType_Register:
		r = "coderMessageType_Register"
	case coderMessageType_Binary:
		r = "coderMessageType_Binary"
	case coderMessageType_Invoke:
		r = "coderMessageType_Invoke"
	case coderMessageType_InvokeRecive:
		r = "coderMessageType_InvokeRecive"
	case coderMessageType_InvokeNameError:
		r = "coderMessageType_InvokeNameError"
	}
	return
}

type CoderMessage struct {
	ID   string
	Type coderMessageType
	Name string
	Msg  []reflect.Value
}

type Coder interface {
	Encoder(CoderMessage) []byte
	Decoder([]byte) (CoderMessage, error)
}

type HproseCoder struct {
}

func NewHproseCoder() *HproseCoder {
	return &HproseCoder{}
}

func (g *HproseCoder) Encoder(message CoderMessage) []byte {
	//w := hio.NewWriter(true)
	//switch message.Type {
	//case coderMessageType_Invoke:
	//	w.WriteByte(hio.TagCall)
	//case coderMessageType_InvokeRecive:
	//	w.WriteByte(hio.TagResult)
	//}
	//w.WriteString(message.ID)
	//w.WriteString(message.Name)
	//w.WriteInt(int64(len(message.Msg)))
	//w.WriteSlice(message.Msg)
	//return w.Bytes()
	return hio.Marshal(message)
}

func (g *HproseCoder) Decoder(data []byte) (CoderMessage, error) {
	//n := len(data)
	//if n == 0 {
	//	return CoderMessage{{}}, io.ErrUnexpectedEOF
	//}
	//reader := hio.AcquireReader(data, false)
	//defer hio.ReleaseReader(reader)
	//message := CoderMessage{}
	//tag, _ := reader.ReadByte()
	//switch tag {
	//case hio.TagCall:
	//	message.Type = coderMessageType_Invoke
	//case hio.TagResult:
	//	message.Type = coderMessageType_InvokeRecive
	//}
	//message.ID = reader.ReadString()
	//message.Name = reader.ReadString()
	//l := reader.ReadInt()
	//message.Msg = make([]reflect.Value, l)
	//reader.ReadSlice(message.Msg)
	//return message, nil
	var message CoderMessage
	logrus.Debugln("Coder", "Decoder", string(data))
	hio.Unmarshal(data, &message)
	return message, nil
}

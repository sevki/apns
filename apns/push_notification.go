// apns.go
//
// Documentation related to the JSON alert message:
// http://developer.apple.com/library/mac/#documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/ApplePushService.html
//
// Documentation related to the binary protocol:
// http://developer.apple.com/library/ios/#documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/Chapters/CommunicatingWIthAPS.html
package apns

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
)

// The first byte of the push notification payload is the command value,
// which at this time is fixed to 1 by Apple.
const PUSH_COMMAND_VALUE = 1

// The total length of the payload cannot exceed this amount.
const MAX_PAYLOAD_SIZE_BYTES = 256

type Payload struct {
	Alert interface{} `json:"alert,omitempty"`
	Badge int         `json:"badge,omitempty"`
	Sound string      `json:"sound,omitempty"`
}

// From the APN documentation:
// "You should use the ... alert dictionary in general only if you absolutely need to."
type AlertDictionary struct {
	Body         string   `json:"body,omitempty"`
	ActionLocKey string   `json:"action-loc-key,omitempty"`
	LocKey       string   `json:"loc-key,omitempty"`
	LocArgs      []string `json:"loc-args,omitempty"`
	LaunchImage  string   `json:"launch-image,omitempty"`
}

// The Envelope is the overall wrapper for the various push notification fields.
// The length fields are computed in ToBytes() and as such aren't represented
// in the struct itself.
type Envelope struct {
	Identifier  int32
	Expiry      uint32
	DeviceToken string

	payload map[string]interface{}
}

func (this *Envelope) AddPayload(p *Payload) {
	this.Set("aps", p)
}

// The push notification requests support arbitrary custom metadata.
// This method requires a string key and any type for the value.
func (this *Envelope) Set(key string, value interface{}) {
	if this.payload == nil {
		this.payload = make(map[string]interface{})
	}
	this.payload[key] = value
}

// This method looks up a string key and returns the value, or nil.
func (this *Envelope) Get(key string) interface{} {
	return this.payload[key]
}

// Returns the JSON structure as a byte array.
func (this *Envelope) PayloadJSON() ([]byte, error) {
	return json.Marshal(this.payload)
}

// Returns a byte array of the complete Envelope struct. This array
// is what should be transmitted to the APN Service.
func (this *Envelope) ToBytes() ([]byte, error) {
	token, err := hex.DecodeString(this.DeviceToken)
	if err != nil {
		return nil, err
	}
	payload, err := this.PayloadJSON()
	if err != nil {
		return nil, err
	}
	if len(payload) > MAX_PAYLOAD_SIZE_BYTES {
		return nil, errors.New("payload is larger than the " + strconv.Itoa(MAX_PAYLOAD_SIZE_BYTES) + " byte limit")
	}

	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, uint8(PUSH_COMMAND_VALUE))
	binary.Write(buffer, binary.BigEndian, uint32(this.Identifier))
	binary.Write(buffer, binary.BigEndian, uint32(this.Expiry))
	binary.Write(buffer, binary.BigEndian, uint16(len(token)))
	binary.Write(buffer, binary.BigEndian, token)
	binary.Write(buffer, binary.BigEndian, uint16(len(payload)))
	binary.Write(buffer, binary.BigEndian, payload)
	return buffer.Bytes(), nil
}

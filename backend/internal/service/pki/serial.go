package pki

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"
)

var requestSerialState struct {
	sync.Mutex
	lastMillis int64
}

// randomSerial matches pki-cert's timestamp plus four random bytes serial construction.
func randomSerial() (string, *big.Int, error) {
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", nil, err
	}

	serialText := strings.ToUpper(strconv.FormatInt(time.Now().UnixMilli(), 16)) + strings.ToUpper(hex.EncodeToString(randomBytes))
	if len(serialText)%2 != 0 {
		serialText = "0" + serialText
	}
	serial, ok := new(big.Int).SetString(serialText, 16)
	if !ok {
		return "", nil, errors.New("invalid certificate serial")
	}
	return serialText, serial, nil
}

func pendingSerial() string {
	requestSerialState.Lock()
	defer requestSerialState.Unlock()

	now := time.Now().UnixMilli()
	for now <= requestSerialState.lastMillis {
		time.Sleep(time.Millisecond)
		now = time.Now().UnixMilli()
	}
	requestSerialState.lastMillis = now
	return pendingSerialAt(time.UnixMilli(now))
}

func pendingSerialAt(createdAt time.Time) string {
	value := strconv.FormatInt(createdAt.UnixMilli(), 10)
	if len(value) > 8 {
		value = value[len(value)-8:]
	}
	return "REQ:" + value
}

func displayCertificateSerial(serial string) string {
	if strings.HasPrefix(serial, "REQ:") {
		return serial
	}
	return displayIssuedSerial(serial)
}

func displayIssuedSerial(serial string) string {
	var hexText strings.Builder
	for _, character := range serial {
		if ('0' <= character && character <= '9') || ('A' <= character && character <= 'F') || ('a' <= character && character <= 'f') {
			hexText.WriteRune(character)
		}
	}
	value := strings.ToUpper(hexText.String())
	if value == "" {
		return serial
	}
	if len(value)%2 != 0 {
		value = "0" + value
	}

	parts := make([]string, 0, len(value)/2)
	for index := 0; index < len(value); index += 2 {
		parts = append(parts, value[index:index+2])
	}
	return strings.Join(parts, ":")
}

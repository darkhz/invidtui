package resolver

import (
	"io"
	"sync"

	"github.com/ugorji/go/codec"
)

// Resolver describes an encoder/decoder handler.
type Resolver struct {
	jsonDecoder *codec.Decoder
	jsonEncoder *codec.Encoder
	jsonHandle  sync.Mutex

	simpleDecoder *codec.Decoder
	simpleEncoder *codec.Encoder
	simpleHandle  sync.Mutex

	setupHandler sync.Mutex

	init bool
}

var resolver Resolver

// setup sets up the resolver.
func (r *Resolver) setup() {
	r.setupHandler.Lock()
	defer r.setupHandler.Unlock()

	if resolver.init {
		return
	}

	r.jsonDecoder = codec.NewDecoder(nil, &codec.JsonHandle{})
	r.jsonEncoder = codec.NewEncoder(nil, &codec.JsonHandle{})

	r.simpleDecoder = codec.NewDecoder(nil, &codec.JsonHandle{})
	r.simpleEncoder = codec.NewEncoder(nil, &codec.JsonHandle{})

	r.init = true
}

// DecodeJSONReader decodes JSON data from a Reader.
func DecodeJSONReader(reader io.Reader, apply interface{}) error {
	resolver.setup()

	resolver.jsonHandle.Lock()
	defer resolver.jsonHandle.Unlock()

	resolver.jsonDecoder.Reset(reader)
	return resolver.jsonDecoder.Decode(apply)
}

// DecodeJSONBytes decodes JSON data from a byte array.
func DecodeJSONBytes(data []byte, apply interface{}) error {
	resolver.setup()

	resolver.jsonHandle.Lock()
	defer resolver.jsonHandle.Unlock()

	resolver.jsonDecoder.ResetBytes(data)
	return resolver.jsonDecoder.Decode(apply)
}

// DecodeSimpleReader decodes data from a Reader.
func DecodeSimpleReader(reader io.Reader, apply interface{}) error {
	resolver.setup()

	resolver.simpleHandle.Lock()
	defer resolver.simpleHandle.Unlock()

	resolver.simpleDecoder.Reset(reader)
	return resolver.simpleDecoder.Decode(apply)
}

// DecodeSimpleBytes decodes data from a byte array.
func DecodeSimpleBytes(data []byte, apply interface{}) error {
	resolver.setup()

	resolver.simpleHandle.Lock()
	defer resolver.simpleHandle.Unlock()

	resolver.simpleDecoder.ResetBytes(data)
	return resolver.simpleDecoder.Decode(apply)
}

// EncodeJSONBytes encodes data from a byte array.
func EncodeSimpleBytes(data *[]byte, apply interface{}) error {
	resolver.setup()

	resolver.simpleHandle.Lock()
	defer resolver.simpleHandle.Unlock()

	resolver.simpleEncoder.ResetBytes(data)
	return resolver.simpleEncoder.Encode(apply)
}

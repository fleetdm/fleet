package cryptoutil

import (
	"bytes"
	"encoding/base64"
	"errors"
	"testing"
	"time"
)

// indefSeq is a 2-byte BER header: constructed SEQUENCE with indefinite length.
var indefSeq = []byte{0x30, 0x80}

// eoc is the 2-byte end-of-contents marker that closes an indefinite-length
// container.
var eoc = []byte{0x00, 0x00}

// buildNestedIndefiniteSequences returns a BER payload that nests N
// indefinite-length SEQUENCEs inside each other.
func buildNestedIndefiniteSequences(depth int) []byte {
	out := make([]byte, 0, depth*4)
	for range depth {
		out = append(out, indefSeq...)
	}
	for range depth {
		out = append(out, eoc...)
	}
	return out
}

func TestValidateBERDepth_RealMdmSignature(t *testing.T) {
	for name, header := range map[string]string{
		"header1": mdmSignatureHeader1,
		"header2": mdmSignatureHeader2,
	} {
		t.Run(name, func(t *testing.T) {
			sig, err := base64.StdEncoding.DecodeString(header)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if err := ValidateBERDepth(sig, MaxBERDepth); err != nil {
				t.Fatalf("real Apple MDM signature rejected at depth %d: %v", MaxBERDepth, err)
			}
		})
	}
}

func TestValidateBERDepth_AcceptsExactlyAtCap(t *testing.T) {
	payload := buildNestedIndefiniteSequences(MaxBERDepth)
	if err := ValidateBERDepth(payload, MaxBERDepth); err != nil {
		t.Fatalf("depth %d should be accepted, got: %v", MaxBERDepth, err)
	}
}

func TestValidateBERDepth_RejectsOneOverCap(t *testing.T) {
	payload := buildNestedIndefiniteSequences(MaxBERDepth + 1)
	err := ValidateBERDepth(payload, MaxBERDepth)
	if !errors.Is(err, ErrBERTooDeep) {
		t.Fatalf("depth %d should be rejected with ErrBERTooDeep, got: %v", MaxBERDepth+1, err)
	}
}

func TestValidateBERDepth_RejectsDoSPayloadFast(t *testing.T) {
	// 10,000 nested indefinite-length SEQUENCEs: Our pre-walk must reject
	// this in milliseconds with negligible allocation.
	payload := buildNestedIndefiniteSequences(10_000)
	start := time.Now()
	err := ValidateBERDepth(payload, MaxBERDepth)
	elapsed := time.Since(start)
	if !errors.Is(err, ErrBERTooDeep) {
		t.Fatalf("expected ErrBERTooDeep on 10k-deep payload, got: %v", err)
	}
	if elapsed > 50*time.Millisecond {
		t.Fatalf("walker too slow: %v (expected <50ms)", elapsed)
	}
}

func TestValidateBERDepth_MalformedReturnsNil(t *testing.T) {
	cases := map[string][]byte{
		"empty":                          {},
		"single tag, no length":          {0x30},
		"truncated long-form length":     {0x30, 0x82, 0x01},
		"long-form length count zero":    {0x30, 0x80, 0x80}, // n=0 is reserved
		"long-form length count nine":    {0x30, 0x89, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		"length exceeds remaining bytes": {0x30, 0x05, 0x01},
		"indefinite on primitive tag":    {0x02, 0x80, 0x00, 0x00},
		"multi-byte tag truncated":       {0x1F, 0x81},
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			if err := ValidateBERDepth(payload, MaxBERDepth); err != nil {
				t.Fatalf("expected nil on malformed %q, got %v", name, err)
			}
		})
	}
}

func TestValidateBERDepth_FlatPrimitivesUnaffected(t *testing.T) {
	// A handful of stacked top-level primitive INTEGERs: depth 0, must pass.
	var b bytes.Buffer
	for i := range 100 {
		b.Write([]byte{0x02, 0x01, byte(i)})
	}
	if err := ValidateBERDepth(b.Bytes(), MaxBERDepth); err != nil {
		t.Fatalf("flat primitives rejected: %v", err)
	}
}

func TestValidateBERDepth_MixedDefiniteAndIndefinite(t *testing.T) {
	// SEQUENCE (definite, 6 bytes content) {
	//   SEQUENCE (indefinite) {
	//     INTEGER 0x01
	//   } EOC
	// }
	payload := []byte{
		0x30, 0x06,
		0x30, 0x80,
		0x02, 0x01, 0x01,
		0x00, 0x00,
	}
	if err := ValidateBERDepth(payload, MaxBERDepth); err != nil {
		t.Fatalf("mixed structure rejected: %v", err)
	}
	// Same shape, two levels deep, with cap=1 -> should reject.
	if err := ValidateBERDepth(payload, 1); !errors.Is(err, ErrBERTooDeep) {
		t.Fatalf("expected ErrBERTooDeep with cap=1, got: %v", err)
	}
}

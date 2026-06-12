package cryptoutil

import "errors"

// MaxBERDepth is the maximum allowed nesting depth of a BER-encoded structure
const MaxBERDepth = 64

// ErrBERTooDeep is returned by ValidateBERDepth when the input nests more
// constructed BER tags than the caller's cap.
var ErrBERTooDeep = errors.New("BER structure exceeds maximum nesting depth")

// frame describes an open BER constructed-tag container.
type berFrame struct {
	indefinite bool
	end        int // absolute byte offset where the container's content ends; ignored if indefinite.
}

// ValidateBERDepth walks data as a stream of BER TLV headers without
// allocating any output buffers. It returns ErrBERTooDeep if depth ever exceeds
// maxDepth.
//
// The walker is intentionally lenient: malformed BER (truncated lengths,
// indefinite length on a primitive tag, length exceeding remaining bytes)
// returns nil so that pkcs7.Parse can surface the real parse error
// downstream. The only error this function returns is ErrBERTooDeep.
func ValidateBERDepth(data []byte, maxDepth int) error {
	if maxDepth < 1 {
		return nil
	}
	stack := make([]berFrame, 0, 16)

	offset := 0
	for offset < len(data) {
		// Pop any definite-length frames we have walked past.
		for len(stack) > 0 && !stack[len(stack)-1].indefinite && offset >= stack[len(stack)-1].end {
			stack = stack[:len(stack)-1]
		}

		// Inside an indefinite-length frame, two consecutive zero bytes are
		// the end-of-contents marker and close the frame.
		if len(stack) > 0 && stack[len(stack)-1].indefinite {
			if offset+1 < len(data) && data[offset] == 0x00 && data[offset+1] == 0x00 {
				offset += 2
				stack = stack[:len(stack)-1]
				continue
			}
		}

		// Parse the tag.
		if offset >= len(data) {
			return nil
		}
		tagByte := data[offset]
		offset++
		constructed := (tagByte & 0x20) != 0
		if (tagByte & 0x1F) == 0x1F {
			// Multi-byte tag: read continuation bytes until one without the high bit.
			for {
				if offset >= len(data) {
					return nil
				}
				b := data[offset]
				offset++
				if b&0x80 == 0 {
					break
				}
			}
		}

		// Parse the length.
		if offset >= len(data) {
			return nil
		}
		lengthByte := data[offset]
		offset++
		var valueLen int
		indefinite := false
		switch {
		case lengthByte == 0x80:
			// Indefinite form: only valid for constructed tags.
			if !constructed {
				return nil
			}
			indefinite = true
		case lengthByte&0x80 == 0:
			// Short form.
			valueLen = int(lengthByte)
		default:
			// Long form: low 7 bits are the count of length bytes (1-8).
			n := int(lengthByte & 0x7F)
			if n == 0 || n > 8 {
				return nil
			}
			if offset+n > len(data) {
				return nil
			}
			var v uint64
			for i := range n {
				v = (v << 8) | uint64(data[offset+i])
			}
			offset += n
			if v > uint64(len(data)) {
				return nil
			}
			valueLen = int(v)
		}

		if constructed {
			if len(stack)+1 > maxDepth {
				return ErrBERTooDeep
			}
			if indefinite {
				stack = append(stack, berFrame{indefinite: true})
			} else {
				end := offset + valueLen
				if end > len(data) {
					return nil
				}
				stack = append(stack, berFrame{end: end})
			}
			continue
		}

		// Primitive tag: skip the value bytes.
		if len(data)-offset < valueLen {
			return nil
		}
		offset += valueLen
	}
	return nil
}

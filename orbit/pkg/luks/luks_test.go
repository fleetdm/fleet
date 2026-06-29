package luks

import (
	"testing"

	"github.com/tj/assert"
)

// output from cryptsetup luksDump /dev/sda3 --debug-json command
// on cryptsetup 2.2.2
var output = `# cryptsetup 2.2.2 processing "cryptsetup luksDump /dev/sda3 --debug-json"
# Running command luksDump.
# Locking memory.
# Installing SIGINT/SIGTERM handler.
# Unblocking interruption on signal.
# Allocating context for crypt device /dev/sda3.
# Trying to open and read device /dev/sda3 with direct-io.
# Initialising device-mapper backend library.
# Trying to load any crypt type from device /dev/sda3.
# Crypto backend (OpenSSL 1.1.1f  31 Mar 2020) initialized in cryptsetup library version 2.2.2.
# Detected kernel Linux 5.4.0-200-generic aarch64.
# Loading LUKS2 header (repair disabled).
# Acquiring read lock for device /dev/sda3.
# Opening lock resource file /run/cryptsetup/L_8:3
# Verifying lock handle for /dev/sda3.
# Device /dev/sda3 READ lock taken.
# Trying to read primary LUKS2 header at offset 0x0.
# Opening locked device /dev/sda3
# Veryfing locked device handle (bdev)
# LUKS2 header version 2 of size 16384 bytes, checksum sha256.
# Checksum:f16cfd36bbc588eb178d9f40e7da030edc6ed04cfb012ee14be14e3af459439b (on-disk)
# Checksum:f16cfd36bbc588eb178d9f40e7da030edc6ed04cfb012ee14be14e3af459439b (in-memory)
# Trying to read secondary LUKS2 header at offset 0x4000.
# Reusing open ro fd on device /dev/sda3
# LUKS2 header version 2 of size 16384 bytes, checksum sha256.
# Checksum:0ce47e7c0460addb7a5ee2546e8404521ca0caf2e02830b1d4ef7172bf617d84 (on-disk)
# Checksum:0ce47e7c0460addb7a5ee2546e8404521ca0caf2e02830b1d4ef7172bf617d84 (in-memory)
# Device size 65442676736, offset 16777216.
# Device /dev/sda3 READ lock released.
# Only 2 active CPUs detected, PBKDF threads decreased from 4 to 2.
# Not enough physical memory detected, PBKDF max memory decreased from 1048576kB to 1010800kB.
# PBKDF argon2i, time_ms 2000 (iterations 0), max_memory_kb 1010800, parallel_threads 2.
# {
  "keyslots":{
    "0":{
      "type":"luks2",
      "key_size":64,
      "af":{
        "type":"luks1",
        "stripes":4000,
        "hash":"sha256"
      },
      "area":{
        "type":"raw",
        "offset":"32768",
        "size":"258048",
        "encryption":"aes-xts-plain64",
        "key_size":64
      },
      "kdf":{
        "type":"argon2i",
        "time":5,
        "memory":1011024,
        "cpus":2,
        "salt":"b8+t4hTY/IFecqsKR20UZSUPFDZqyAtJ9lxYg5ye2Hg="
      }
    }
  },
  "tokens":{
  },
  "segments":{
    "0":{
      "type":"crypt",
      "offset":"16777216",
      "size":"dynamic",
      "iv_tweak":"0",
      "encryption":"aes-xts-plain64",
      "sector_size":512
    }
  },
  "digests":{
    "0":{
      "type":"pbkdf2",
      "keyslots":[
        "0"
      ],
      "segments":[
        "0"
      ],
      "hash":"sha256",
      "iterations":457493,
      "salt":"BbZbHfAY5e90aoKjTYSJoj8ZLiRueUofVJWWG/4trWw=",
      "digest":"f2sSMaJlh5qbdVUYT+RhabOmEit96KBIq0ltzAnfARc="
    }
  },
  "config":{
    "json_size":"12288",
    "keyslots_size":"16744448"
  }
}LUKS header information
Version:       	2
Epoch:         	5
Metadata area: 	16384 [bytes]
Keyslots area: 	16744448 [bytes]
UUID:          	3cf8a080-045d-4bc2-889c-994c9b4a18aa
Label:         	(no label)
Subsystem:     	(no subsystem)
Flags:       	(no flags)

Data segments:
  0: crypt
	offset: 16777216 [bytes]
	length: (whole device)
	cipher: aes-xts-plain64
	sector: 512 [bytes]

Keyslots:
  0: luks2
	Key:        512 bits
	Priority:   normal
	Cipher:     aes-xts-plain64
	Cipher key: 512 bits
	PBKDF:      argon2i
	Time cost:  5
	Memory:     1011024
	Threads:    2
	Salt:       6f cf ad e2 14 d8 fc 81 5e 72 ab 0a 47 6d 14 65
	            25 0f 14 36 6a c8 0b 49 f6 5c 58 83 9c 9e d8 78
	AF stripes: 4000
	AF hash:    sha256
	Area offset:32768 [bytes]
	Area length:258048 [bytes]
	Digest ID:  0
Tokens:
Digests:
  0: pbkdf2
	Hash:       sha256
	Iterations: 457493
	Salt:       05 b6 5b 1d f0 18 e5 ef 74 6a 82 a3 4d 84 89 a2
	            3f 19 2e 24 6e 79 4a 1f 54 95 96 1b fe 2d ad 6c
	Digest:     7f 6b 12 31 a2 65 87 9a 9b 75 55 18 4f e4 61 69
	            b3 a6 12 2b 7d e8 a0 48 ab 49 6d cc 09 df 01 17
# Releasing crypt device /dev/sda3 context.
# Releasing device-mapper backend.
# Closing read only fd for /dev/sda3.
# Unlocking memory.
Command successful.`

var extractedJSON = `{
  "keyslots":{
    "0":{
      "type":"luks2",
      "key_size":64,
      "af":{
        "type":"luks1",
        "stripes":4000,
        "hash":"sha256"
      },
      "area":{
        "type":"raw",
        "offset":"32768",
        "size":"258048",
        "encryption":"aes-xts-plain64",
        "key_size":64
      },
      "kdf":{
        "type":"argon2i",
        "time":5,
        "memory":1011024,
        "cpus":2,
        "salt":"b8+t4hTY/IFecqsKR20UZSUPFDZqyAtJ9lxYg5ye2Hg="
      }
    }
  },
  "tokens":{
  },
  "segments":{
    "0":{
      "type":"crypt",
      "offset":"16777216",
      "size":"dynamic",
      "iv_tweak":"0",
      "encryption":"aes-xts-plain64",
      "sector_size":512
    }
  },
  "digests":{
    "0":{
      "type":"pbkdf2",
      "keyslots":[
        "0"
      ],
      "segments":[
        "0"
      ],
      "hash":"sha256",
      "iterations":457493,
      "salt":"BbZbHfAY5e90aoKjTYSJoj8ZLiRueUofVJWWG/4trWw=",
      "digest":"f2sSMaJlh5qbdVUYT+RhabOmEit96KBIq0ltzAnfARc="
    }
  },
  "config":{
    "json_size":"12288",
    "keyslots_size":"16744448"
  }
}`

func TestExtractJson(t *testing.T) {
	json, err := extractJSON([]byte(output))
	assert.NoError(t, err)
	assert.Equal(t, extractedJSON, string(json))

	_, err = extractJSON([]byte("no json"))
	assert.Error(t, err)
}

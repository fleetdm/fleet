// Copyright (c) 2009 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package server

import (
	"fmt"
	"strings"
)

// matchErr is a test helper that verifies that an error is matched with an expected effor
// source:
// https://github.com/golang/go/blob/ffa2bd27a47ef16e4d6a404dd15781ed5ba21e5d/src/net/http/response_test.go#L865
// wantErr can be nil, an error value to match exactly, or type string to
// match a substring.
func matchErr(err error, wantErr interface{}) error {
	if err == nil {
		if wantErr == nil {
			return nil
		}
		if sub, ok := wantErr.(string); ok {
			return fmt.Errorf("unexpected success; want error with substring %q", sub)
		}
		return fmt.Errorf("unexpected success; want error %v", wantErr)
	}
	if wantErr == nil {
		return fmt.Errorf("%v; want success", err)
	}
	if sub, ok := wantErr.(string); ok {
		if strings.Contains(err.Error(), sub) {
			return nil
		}
		return fmt.Errorf("error = %v; want an error with substring %q", err, sub)
	}
	if err == wantErr {
		return nil
	}
	return fmt.Errorf("%v; want %v", err, wantErr)
}

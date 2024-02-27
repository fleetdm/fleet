package http

import (
	"net/http"
	"sync"
)

// MethodMux is an HTTP request multiplexer.
// It matches the HTTP method of each incoming request against a list of
// registered HTTP method names and calls the handler that matches.
type MethodMux struct {
	methodsMu sync.RWMutex
	methods   map[string]http.Handler
}

// NewMethodMux creates a new MethodMux.
func NewMethodMux() *MethodMux { return new(MethodMux) }

// Handle registers the handler for the given method.
// If handler already exists for the given method, Handle panics.
func (mux *MethodMux) Handle(method string, handler http.Handler) {
	if method == "" {
		panic("http: invalid method")
	}
	if handler == nil {
		panic("http: nil handler")
	}
	mux.methodsMu.Lock()
	defer mux.methodsMu.Unlock()
	if mux.methods == nil {
		mux.methods = make(map[string]http.Handler)
	} else if _, exists := mux.methods[method]; exists {
		panic("http: multiple registrations for " + method)
	}
	mux.methods[method] = handler
}

func (mux *MethodMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var next http.Handler
	mux.methodsMu.RLock()
	if mux.methods != nil {
		next = mux.methods[r.Method]
	}
	mux.methodsMu.RUnlock()
	if next == nil {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	next.ServeHTTP(w, r)
}

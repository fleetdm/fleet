//go:build windows
// +build windows

// Package wmi provides a basic interface for querying against
// wmi. It's based on some underlying examples using ole [1].
//
// We do _not_ use the stackdriver library [2], because that uses reflect
// and wants typed objects. Our use case is too dynamic.
//
// To understand the available classes, take a look at the Microsoft
// documention [3]
//
// Servers, Namespaces, and connection parameters:
//
// WMI has a fairly rich set of connection options. It allows querying
// on remote servers, via authenticated users names, in different name
// spaces... These options are exposed through functional arguments.
//
// References:
//
// 1. https://stackoverflow.com/questions/20365286/query-wmi-from-go
// 2. https://github.com/StackExchange/wmi
// 3. https://docs.microsoft.com/en-us/windows/win32/cimwin32prov/operating-system-classes
//
// Namespaces, ongoing:
//
// To list them: gwmi -namespace "root" -class "__Namespace" | Select Name
// To list classes: gwmi -namespace root\cimv2 -list
// Default: ROOT\CIMV2
//
// Get-WmiObject -Query "select * from win32_service where name='WinRM'"
// Get-WmiObject  -namespace root\cimv2\security\MicrosoftTpm -Query "SELECT * FROM Win32_Tpm"
// based on github.com/kolide/launcher/pkg/osquery/tables
package wmi

import (
	"context"
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"

	"github.com/scjalliance/comshim"
)

// S_FALSE is returned by CoInitializeEx if it was already called on this thread.
const S_FALSE = 0x00000001

// querySettings contains various options. Mostly for the
// connectServerArgs args. See
// https://docs.microsoft.com/en-us/windows/win32/wmisdk/swbemlocator-connectserver
// for details.
type querySettings struct {
	connectServer        string
	connectNamespace     string
	connectUser          string
	connectPassword      string
	connectLocale        string
	connectAuthority     string
	connectSecurityFlags uint
	whereClause          string
}

// ConnectServerArgs returns an array suitable for being passed to ole
// call ConnectServer
func (qs *querySettings) ConnectServerArgs() []interface{} {
	return []interface{}{
		qs.connectServer,
		qs.connectNamespace,
		qs.connectUser,
		qs.connectPassword,
		qs.connectLocale,
		qs.connectAuthority,
		qs.connectSecurityFlags,
	}
}

type Option func(*querySettings)

// ConnectServer sets the server to connect to. It defaults to "",
// which is localhost.
func ConnectServer(s string) Option {
	return func(qs *querySettings) {
		qs.connectServer = s
	}
}

// ConnectNamespace sets the namespace to query against. It defaults
// to "", which is the same as `ROOT\CIMV2`
func ConnectNamespace(s string) Option {
	return func(qs *querySettings) {
		qs.connectNamespace = s
	}
}

// ConnectUseMaxWait requires that ConnectServer use a timeout. The
// call is then guaranteed to return in 2 minutes or less. This option
// is strongly recommended, as without it calls can block forever.
func ConnectUseMaxWait() Option {
	return func(qs *querySettings) {
		// see the definition of iSecurityFlags in
		// https://docs.microsoft.com/en-us/windows/win32/wmisdk/swbemlocator-connectserver
		qs.connectSecurityFlags = qs.connectSecurityFlags & 128
	}
}

// WithWhere will be used for the optional WHERE clause in wmi.
func WithWhere(whereClause string) Option {
	return func(qs *querySettings) {
		qs.whereClause = whereClause
	}
}

func Query(ctx context.Context, logger log.Logger, className string, properties []string, opts ...Option) ([]map[string]interface{}, error) {
	handler := NewOleHandler(ctx, logger, properties)

	// settings
	qs := &querySettings{}
	for _, opt := range opts {
		opt(qs)
	}

	var whereClause string
	if qs.whereClause != "" {
		whereClause = fmt.Sprintf(" WHERE %s", qs.whereClause)
	}

	// If we query for the exact fields, _and_ one of the property
	// names is wrong, we get no results. (clearly an error. but I
	// can't find it) So query for `*`, and then fetch the
	// property. More testing might show this needs to change
	queryString := fmt.Sprintf("SELECT * FROM %s%s", className, whereClause)

	// Initialize the COM system.
	comshim.Add(1)
	defer comshim.Done()

	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return nil, fmt.Errorf("ole createObject: %w", err)
	}
	defer unknown.Release()

	wmi, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("query interface create: %w", err)
	}
	defer wmi.Release()

	// service is a SWbemServices
	serviceRaw, err := oleutil.CallMethod(wmi, "ConnectServer", qs.ConnectServerArgs()...)
	if err != nil {
		return nil, fmt.Errorf("wmi connectserver: %w", err)
	}
	defer serviceRaw.Clear()

	service := serviceRaw.ToIDispatch()
	defer service.Release()

	level.Debug(logger).Log("msg", "Running WMI query", "query", queryString)

	// result is a SWBemObjectSet
	resultRaw, err := oleutil.CallMethod(service, "ExecQuery", queryString)
	if err != nil {
		return nil, fmt.Errorf("Running query %s: %w", queryString, err)
	}
	defer resultRaw.Clear()

	result := resultRaw.ToIDispatch()
	defer result.Release()

	if err := oleutil.ForEach(result, handler.HandleVariant); err != nil {
		return nil, fmt.Errorf("ole foreach: %w", err)
	}

	return handler.results, nil
}

type oleHandler struct {
	logger     log.Logger
	results    []map[string]interface{}
	properties []string
}

func NewOleHandler(ctx context.Context, logger log.Logger, properties []string) *oleHandler {
	return &oleHandler{
		logger:     logger,
		properties: properties,
		results:    []map[string]interface{}{},
	}
}

func (oh *oleHandler) HandleVariant(v *ole.VARIANT) error {
	item := v.ToIDispatch()
	defer item.Release()

	result := make(map[string]interface{})

	for _, p := range oh.properties {
		val, err := oleutil.GetProperty(item, p)
		if err != nil {
			level.Debug(oh.logger).Log("msg", "Got error looking for property", "property", p, "err", err)
			continue
		}
		defer val.Clear()

		// Not sure if we need to special case the nil, or if Value() handles it.
		if val.VT == 0x1 { // VT_NULL
			result[p] = nil
			continue
		}

		// Attempt to handle arrays
		safeArray := val.ToArray()
		if safeArray != nil {
			// I would have expected to need
			// `defersafeArray.Release()` here, if I add
			// that, this routine stops working.
			result[p] = safeArray.ToValueArray()
		} else {
			result[p] = val.Value()
		}

	}
	if len(result) > 0 {
		oh.results = append(oh.results, result)
	}

	return nil
}

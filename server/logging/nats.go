package logging

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/vm"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/nats-io/nats.go"
	"github.com/valyala/fastjson"
	"github.com/valyala/fasttemplate"
)

// natsPublisher sends logs to a NATS server.
type natsPublisher interface {
	// Flush waits for all published logs to be acknowledged.
	Flush(context.Context) error

	// Publish sends the log message to the NATS server.
	Publish(context.Context, *nats.Msg) error
}

// natsRouter determines the NATS subject for a log.
type natsRouter interface {
	Route(json.RawMessage) (string, error)
}

// natsLogWriter represents a NATS log writer.
type natsLogWriter struct {
	// The NATS connection.
	client *nats.Conn

	// The optional compression algorithm to use.
	compression string

	// Whether to use JetStream.
	jetstream bool

	// The subject router.
	router natsRouter

	// The timeout for the writer.
	timeout time.Duration
}

// Define the NATS subject template tags.
const (
	natsSubjectTagStart = "{"
	natsSubjectTagStop  = "}"
)

// Define the supported compression algorithms.
var compressionOk = map[string]bool{
	"gzip":   true,
	"snappy": true,
	"zstd":   true,
}

// NewNatsLogWriter creates a new NATS log writer.
func NewNatsLogWriter(server, subject, credFile, nkeyFile, tlsClientCrtFile, tlsClientKeyFile, tlsCACrtFile, compression string, jetstream bool, timeout time.Duration, logger log.Logger) (*natsLogWriter, error) {
	// Ensure the NATS server is set.
	if server == "" {
		return nil, errors.New("nats server missing")
	}

	// Ensure the NATS subject is set.
	if subject == "" {
		return nil, errors.New("nats subject missing")
	}

	// Ensure credentials file and NKey file are not used together.
	if credFile != "" && nkeyFile != "" {
		return nil, errors.New("nats credentials and nkey files cannot be used together")
	}

	// Validate the compression algorithm if specified.
	if compression != "" && !compressionOk[compression] {
		return nil, fmt.Errorf("unsupported compression algorithm: %s", compression)
	}

	// Create the NATS connection options.
	opts := []nats.Option{nats.Name("NATS Fleet Writer")}

	// Is a credentials file set?
	if credFile != "" {
		level.Debug(logger).Log(
			"msg", "using credentials file",
			"file", credFile,
		)

		opts = append(opts, nats.UserCredentials(credFile))
	}

	// Is a NKey seed file set?
	if nkeyFile != "" {
		level.Debug(logger).Log(
			"msg", "using NKey file",
			"file", nkeyFile,
		)

		opt, err := nats.NkeyOptionFromSeed(nkeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create nats nkey option: %w", err)
		}

		opts = append(opts, opt)
	}

	// Is a TLS client certificate and key set?
	if tlsClientCrtFile != "" && tlsClientKeyFile != "" {
		level.Debug(logger).Log(
			"msg", "using TLS client certificate and key files",
			"crt", tlsClientCrtFile,
			"key", tlsClientKeyFile,
		)

		opts = append(opts, nats.ClientCert(tlsClientCrtFile, tlsClientKeyFile))
	}

	// Is a CA certificate set?
	if tlsCACrtFile != "" {
		level.Debug(logger).Log(
			"msg", "using CA certificate file",
			"file", tlsCACrtFile,
		)

		opts = append(opts, nats.RootCAs(tlsCACrtFile))
	}

	level.Debug(logger).Log(
		"msg", "connecting to NATS server",
		"server", server,
	)

	// Connect to the NATS server.
	client, err := nats.Connect(server, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to nats server: %w", err)
	}

	level.Debug(logger).Log(
		"msg", "connected to NATS server",
		"server", server,
	)

	var router natsRouter

	// Determine the router to use based on the subject.
	if strings.Contains(subject, natsSubjectTagStart) && strings.Contains(subject, natsSubjectTagStop) {
		router, err = newNatsTemplateRouter(subject)
	} else {
		router, err = newNatsConstantRouter(subject)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create nats router: %w", err)
	}

	// Return the NATS log writer.
	return &natsLogWriter{
		client:      client,
		compression: compression,
		jetstream:   jetstream,
		router:      router,
		timeout:     timeout,
	}, nil
}

// compress compresses the data using the configured algorithm.
func (w *natsLogWriter) compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	switch w.compression {
	case "gzip":
		gw := gzip.NewWriter(&buf)

		if _, err := gw.Write(data); err != nil {
			return nil, err
		}

		if err := gw.Close(); err != nil {
			return nil, err
		}

	case "snappy":
		// Use raw snappy encoding (not framed).
		data = snappy.Encode(nil, data)

		return data, nil

	case "zstd":
		zw, err := zstd.NewWriter(&buf)
		if err != nil {
			return nil, err
		}

		if _, err := zw.Write(data); err != nil {
			return nil, err
		}

		if err := zw.Close(); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", w.compression)
	}

	return buf.Bytes(), nil
}

// Write publishes the logs to the NATS server.
func (w *natsLogWriter) Write(ctx context.Context, logs []json.RawMessage) error {
	var pub natsPublisher
	var err error

	// Create the NATS publisher, according to the JetStream setting.
	if w.jetstream {
		pub, err = newNatsStreamPublisher(w.client)
	} else {
		pub, err = newNatsClientPublisher(w.client)
	}

	if err != nil {
		return fmt.Errorf("failed to create nats publisher: %w", err)
	}

	// Create a context with the configured timeout.
	ctx, cancel := context.WithTimeout(ctx, w.timeout)

	defer cancel()

	// Process each log, routing it to the appropriate subject.
	for _, log := range logs {
		sub, err := w.router.Route(log)
		if err != nil {
			return fmt.Errorf("failed to route log: %w", err)
		}

		// Create the message header.
		header := make(nats.Header)
		header.Set("Content-Type", "application/json")

		// If compression is enabled, compress the log and update the header.
		if w.compression != "" {
			log, err = w.compress(log)
			if err != nil {
				return fmt.Errorf("failed to compress log: %w", err)
			}

			// Set the compression header to indicate the algorithm used.
			header.Set("Content-Encoding", w.compression)
		}

		// Create the message.
		msg := &nats.Msg{
			Data:    log,
			Header:  header,
			Subject: sub,
		}

		// Publish the message.
		if err := pub.Publish(ctx, msg); err != nil {
			return fmt.Errorf("failed to publish log: %w", err)
		}
	}

	// Wait for all logs in this batch to be acknowledged.
	return pub.Flush(ctx)
}

// natsClientPublisher represents a core NATS publisher.
type natsClientPublisher struct {
	nc *nats.Conn
}

// newNatsClientPublisher creates a new core NATS publisher.
func newNatsClientPublisher(nc *nats.Conn) (*natsClientPublisher, error) {
	return &natsClientPublisher{nc}, nil
}

// Flush flushes the client.
func (p *natsClientPublisher) Flush(ctx context.Context) error {
	return p.nc.FlushWithContext(ctx)
}

// Publish publishes the message to the NATS server.
func (p *natsClientPublisher) Publish(ctx context.Context, msg *nats.Msg) error {
	return p.nc.PublishMsg(msg)
}

// natsStreamPublisher represents a JetStream publisher.
type natsStreamPublisher struct {
	js nats.JetStreamContext
}

// newNatsStreamPublisher creates a new JetStream publisher.
func newNatsStreamPublisher(nc *nats.Conn) (*natsStreamPublisher, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	return &natsStreamPublisher{js}, nil
}

// Flush waits for all published logs to be acknowledged.
func (p *natsStreamPublisher) Flush(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-p.js.PublishAsyncComplete():
		return nil
	}
}

// Publish publishes the message to the NATS server using the JetStream API.
func (p *natsStreamPublisher) Publish(ctx context.Context, msg *nats.Msg) error {
	_, err := p.js.PublishMsgAsync(msg)

	return err
}

// natsConstantRouter returns a constant subject for all logs.
type natsConstantRouter struct {
	sub string
}

// newNatsConstantRouter creates a new constant router.
func newNatsConstantRouter(sub string) (*natsConstantRouter, error) {
	return &natsConstantRouter{sub}, nil
}

// Route returns the constant subject.
func (r *natsConstantRouter) Route(_ json.RawMessage) (string, error) {
	return r.sub, nil
}

// natsTemplateEnv is the evaluation environment for the template expressions.
type natsTemplateEnv struct {
	Log *fastjson.Value `expr:"log"`
}

// natsTemplateGet returns the value of a field from a fastjson value. If it is
// a terminal type, it returns the string representation. Otherwise, it returns
// the fastjson value.
func natsTemplateGet(val *fastjson.Value, path string) any {
	v := val.Get(path)

	switch v.Type() {
	case fastjson.TypeFalse:
		return "false"

	case fastjson.TypeNull:
		return "null"

	case fastjson.TypeNumber:
		return strconv.FormatFloat(v.GetFloat64(), 'f', -1, 64)

	case fastjson.TypeString:
		return string(v.GetStringBytes())

	case fastjson.TypeTrue:
		return "true"
	}

	return v
}

// natsTemplatePatcher patches the expression's AST so it uses natsTemplateGet
// to get the value of a field from the log payload.
type natsTemplatePatcher struct{}

// Visit is called by the expression compiler to patch the AST.
func (p *natsTemplatePatcher) Visit(node *ast.Node) {
	if n, ok := (*node).(*ast.MemberNode); ok {
		ast.Patch(node, &ast.CallNode{
			Callee:    &ast.IdentifierNode{Value: "get"},
			Arguments: []ast.Node{n.Node, n.Property},
		})
	}
}

// natsTemplateRouter uses the log contents to create a subject.
type natsTemplateRouter struct {
	// The JSON parser pool.
	pp *fastjson.ParserPool

	// The compiled programs for each template tag expression.
	pr map[string]*vm.Program

	// The template for the subject.
	tp *fasttemplate.Template
}

// newNatsTemplateRouter creates a new template router.
func newNatsTemplateRouter(sub string) (*natsTemplateRouter, error) {
	// Initialize the programs map.
	pr := make(map[string]*vm.Program)

	// Initialize the template.
	tp := fasttemplate.New(sub,
		natsSubjectTagStart,
		natsSubjectTagStop,
	)

	// Define the expression compiler options.
	opts := []expr.Option{
		expr.Env(&natsTemplateEnv{}),
		expr.Patch(&natsTemplatePatcher{}),
		expr.AsKind(reflect.String),
		expr.Function(
			"get",
			func(params ...any) (any, error) {
				return natsTemplateGet(
					params[0].(*fastjson.Value),
					params[1].(string),
				), nil
			},
			natsTemplateGet,
		),
	}

	// Execute the template, compiling each tag expression.
	_, err := tp.ExecuteFuncStringWithErr(func(_ io.Writer, tag string) (int, error) {
		p, err := expr.Compile(tag, opts...)

		pr[tag] = p

		return 0, err
	})
	if err != nil {
		return nil, err
	}

	return &natsTemplateRouter{pp: new(fastjson.ParserPool), pr: pr, tp: tp}, nil
}

// Route returns the subject for a log.
func (r *natsTemplateRouter) Route(log json.RawMessage) (string, error) {
	// Get a JSON parser from the pool, and ensure it is released when done.
	p := r.pp.Get()
	defer r.pp.Put(p)

	// Parse the log contents into a fastjson value.
	v, err := p.ParseBytes(log)
	if err != nil {
		return "", fmt.Errorf("failed to parse log: %w", err)
	}

	// Define the function to evaluate the template for a tag.
	fn := func(w io.Writer, tag string) (int, error) {
		// Evaluate the tag expression.
		r, err := expr.Run(r.pr[tag], &natsTemplateEnv{Log: v})
		if err != nil {
			return 0, err
		}

		// If the returned value is a string, write it.
		if s, ok := r.(string); ok {
			return io.WriteString(w, s)
		}

		// A non-string value was returned.
		return 0, fmt.Errorf("expected string, got %T: %v", r, r)
	}

	return r.tp.ExecuteFuncStringWithErr(fn)
}

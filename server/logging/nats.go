// SPDX-License-Identifier: MIT Expat
package logging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/vm"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/nats-io/nats.go"
	"github.com/valyala/fastjson"
	"github.com/valyala/fasttemplate"
)

// natsPublisher sends logs to a NATS server.
type natsPublisher interface {
	// Flush waits for all published logs to be acknowledged.
	Flush(context.Context) error

	// Publish sends the log message to the NATS server.
	Publish(context.Context, string, json.RawMessage) error
}

// natsRouter determines the NATS subject for a log.
type natsRouter interface {
	Route(json.RawMessage) (string, error)
}

// natsLogWriter represents a NATS log writer.
type natsLogWriter struct {
	// The NATS connection.
	client *nats.Conn

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

// NewNatsLogWriter creates a new NATS log writer.
func NewNatsLogWriter(server, subject, credFile, nkeyFile, tlsClientCrtFile, tlsClientKeyFile, tlsCACrtFile string, jetstream bool, timeout time.Duration, logger log.Logger) (*natsLogWriter, error) {
	// Ensure the NATS server URL is set.
	if server == "" {
		return nil, errors.New("nats server URL missing")
	}

	// Ensure the NATS subject is set.
	if subject == "" {
		return nil, errors.New("nats subject missing")
	}

	// Ensure credentials file and NKey file are not used together.
	if credFile != "" && nkeyFile != "" {
		return nil, errors.New("nats credentials and nkey files cannot be used together")
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

	// Start by assuming a constant subject.
	var router natsRouter = newNatsConstantRouter(subject)

	// If the subject contains template tags, use a template router instead.
	if strings.Contains(subject, natsSubjectTagStart) && strings.Contains(subject, natsSubjectTagStop) {
		router = newNatsTemplateRouter(subject)
	}

	// Return the NATS log writer.
	return &natsLogWriter{
		client:    client,
		jetstream: jetstream,
		router:    router,
		timeout:   timeout,
	}, nil
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

	// Process each log in the batch, routing it to the appropriate subject.
	for _, log := range logs {
		sub, err := w.router.Route(log)
		if err != nil {
			return fmt.Errorf("failed to route log: %w", err)
		}

		if err := pub.Publish(ctx, sub, log); err != nil {
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

// Publish publishes the log to the specified subject.
func (p *natsClientPublisher) Publish(ctx context.Context, sub string, log json.RawMessage) error {
	return p.nc.Publish(sub, log)
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

// Publish publishes the log to the specified subject using the JetStream API.
func (p *natsStreamPublisher) Publish(ctx context.Context, sub string, log json.RawMessage) error {
	_, err := p.js.PublishAsync(sub, log)

	return err
}

// natsConstantRouter returns a constant subject for all logs.
type natsConstantRouter struct {
	sub string
}

// newNatsConstantRouter creates a new constant router.
func newNatsConstantRouter(sub string) *natsConstantRouter {
	return &natsConstantRouter{sub}
}

// Route returns the constant subject.
func (r *natsConstantRouter) Route(_ json.RawMessage) (string, error) {
	return r.sub, nil
}

// natsTemplateEnv is the evaluation environment for the template expressions.
type natsTemplateEnv struct {
	Log *fastjson.Value `expr:"log"`
}

// natsTemplateGet gets the value of a field from a fastjson value. If it is a
// terminal type, it returns the string representation. Otherwise, it returns
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
// to get the value of a field from the log.
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

	sync.Mutex
}

// newNatsTemplateRouter creates a new template router.
func newNatsTemplateRouter(sub string) *natsTemplateRouter {
	return &natsTemplateRouter{
		pp: new(fastjson.ParserPool),
		pr: make(map[string]*vm.Program),
		tp: fasttemplate.New(sub, natsSubjectTagStart, natsSubjectTagStop),
	}
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
		pr, err := r.compileTag(tag)
		if err != nil {
			return 0, err
		}

		// Evaluate the tag expression.
		r, err := expr.Run(pr, &natsTemplateEnv{Log: v})
		if err != nil {
			return 0, err
		}

		// If the returned value is a string, write it.
		if s, ok := r.(string); ok {
			return io.WriteString(w, s)
		}

		// Non-string type returned.
		return 0, fmt.Errorf("expected string, got %T: %v", r, r)
	}

	return r.tp.ExecuteFuncStringWithErr(fn)
}

// compileTag compiles a tag expression into a program.
func (r *natsTemplateRouter) compileTag(tag string) (*vm.Program, error) {
	// Prevent concurrent access to the programs map.
	r.Lock()
	defer r.Unlock()

	// If this is the first time we see this tag expression, compile it.
	if _, ok := r.pr[tag]; !ok {
		pr, err := expr.Compile(
			tag,
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
		)
		if err != nil {
			return nil, fmt.Errorf("failed to compile '%s': %w", tag, err)
		}

		r.pr[tag] = pr
	}

	return r.pr[tag], nil
}

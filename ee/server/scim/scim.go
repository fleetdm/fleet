package scim

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/errors"
	"github.com/elimity-com/scim/optional"
	"github.com/elimity-com/scim/schema"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/service/middleware/log"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const (
	maxResults = 1000
)

func RegisterSCIM(
	mux *http.ServeMux,
	ds fleet.Datastore,
	svc fleet.Service,
	logger kitlog.Logger,
) error {
	config := scim.ServiceProviderConfig{
		// TODO: DocumentationURI and Authentication scheme
		DocumentationURI: optional.NewString("https://fleetdm.com/docs/get-started/why-fleet"),
		SupportFiltering: true,
		SupportPatch:     true,
		MaxResults:       maxResults,
	}

	// The common attributes are id, externalId, and meta.
	// In practice only meta.resourceType is required, while the other four (created, lastModified, location, and version) are not strictly required.
	userSchema := schema.Schema{
		ID:          "urn:ietf:params:scim:schemas:core:2.0:User",
		Name:        optional.NewString("User"),
		Description: optional.NewString("SCIM User"),
		Attributes: []schema.CoreAttribute{
			schema.SimpleCoreAttribute(schema.SimpleStringParams(schema.StringParams{
				Name:       "userName",
				Required:   true,
				Uniqueness: schema.AttributeUniquenessServer(),
			})),
			schema.ComplexCoreAttribute(schema.ComplexParams{
				Description: optional.NewString("The components of the user's real name. Providers MAY return just the full name as a single string in the formatted sub-attribute, or they MAY return just the individual component attributes using the other sub-attributes, or they MAY return both. If both variants are returned, they SHOULD be describing the same name, with the formatted name indicating how the component attributes should be combined."),
				Name:        "name",
				SubAttributes: []schema.SimpleParams{
					schema.SimpleStringParams(schema.StringParams{
						Description: optional.NewString("The family name of the User, or last name in most Western languages (e.g., 'Jensen' given the full name 'Ms. Barbara J Jensen, III')."),
						Name:        "familyName",
					}),
					schema.SimpleStringParams(schema.StringParams{
						Description: optional.NewString("The given name of the User, or first name in most Western languages (e.g., 'Barbara' given the full name 'Ms. Barbara J Jensen, III')."),
						Name:        "givenName",
					}),
				},
			}),
			schema.ComplexCoreAttribute(schema.ComplexParams{
				Description: optional.NewString("Email addresses for the user. The value SHOULD be canonicalized by the service provider, e.g., 'bjensen@example.com' instead of 'bjensen@EXAMPLE.COM'. Canonical type values of 'work', 'home', and 'other'."),
				MultiValued: true,
				Name:        "emails",
				SubAttributes: []schema.SimpleParams{
					schema.SimpleStringParams(schema.StringParams{
						Description: optional.NewString("Email addresses for the user. The value SHOULD be canonicalized by the service provider, e.g., 'bjensen@example.com' instead of 'bjensen@EXAMPLE.COM'. Canonical type values of 'work', 'home', and 'other'."),
						Name:        "value",
					}),
					schema.SimpleStringParams(schema.StringParams{
						CanonicalValues: []string{"work", "home", "other"},
						Description:     optional.NewString("A label indicating the attribute's function, e.g., 'work' or 'home'."),
						Name:            "type",
					}),
					schema.SimpleBooleanParams(schema.BooleanParams{
						Description: optional.NewString("A Boolean value indicating the 'primary' or preferred attribute value for this attribute, e.g., the preferred mailing address or primary email address. The primary attribute value 'true' MUST appear no more than once."),
						Name:        "primary",
					}),
				},
			}),
			schema.SimpleCoreAttribute(schema.SimpleBooleanParams(schema.BooleanParams{
				Description: optional.NewString("A Boolean value indicating the User's administrative status."),
				Name:        "active",
			})),
		},
	}

	scimLogger := kitlog.With(logger, "component", "SCIM")
	resourceTypes := []scim.ResourceType{
		{
			ID:          optional.NewString("User"),
			Name:        "User",
			Endpoint:    "/Users",
			Description: optional.NewString("User Account"),
			Schema:      userSchema,
			Handler:     NewUserHandler(ds, scimLogger),
		},
	}

	serverArgs := &scim.ServerArgs{
		ServiceProviderConfig: &config,
		ResourceTypes:         resourceTypes,
	}

	serverOpts := []scim.ServerOption{
		scim.WithLogger(&scimErrorLogger{Logger: scimLogger}),
	}

	server, err := scim.NewServer(serverArgs, serverOpts...)
	if err != nil {
		return err
	}

	scimErrorHandler := func(w http.ResponseWriter, detail string, status int) {
		errorHandler(w, scimLogger, detail, status)
	}
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return err
	}

	// TODO: Add APM/OpenTelemetry tracing and Prometheus middleware
	applyMiddleware := func(prefix string, server http.Handler) http.Handler {
		handler := http.StripPrefix(prefix, server)
		handler = AuthorizationMiddleware(authorizer, scimLogger, handler)
		handler = auth.AuthenticatedUserMiddleware(svc, scimErrorHandler, handler)
		handler = log.LogResponseEndMiddleware(scimLogger, handler)
		handler = auth.SetRequestsContextMiddleware(svc, handler)
		return handler
	}

	mux.Handle("/api/v1/fleet/scim/", applyMiddleware("/api/v1/fleet/scim", server))
	mux.Handle("/api/latest/fleet/scim/", applyMiddleware("/api/latest/fleet/scim", server))
	return nil
}

func AuthorizationMiddleware(authorizer *authz.Authorizer, logger kitlog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := authorizer.Authorize(r.Context(), &fleet.ScimUser{}, fleet.ActionWrite)
		if err != nil {
			errorHandler(w, logger, err.Error(), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func errorHandler(w http.ResponseWriter, logger kitlog.Logger, detail string, status int) {
	scimErr := errors.ScimError{
		Status: status,
		Detail: detail,
	}
	raw, err := json.Marshal(scimErr)
	if err != nil {
		level.Error(logger).Log("msg", "failed marshaling scim error", "scimError", scimErr, "err", err)
		return
	}

	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(scimErr.Status)
	_, err = w.Write(raw)
	if err != nil {
		level.Error(logger).Log("msg", "failed writing response", "err", err)
	}
}

type scimErrorLogger struct {
	kitlog.Logger
}

var _ scim.Logger = &scimErrorLogger{}

func (l *scimErrorLogger) Error(args ...interface{}) {
	level.Error(l.Logger).Log(
		"error", fmt.Sprint(args...),
	)
}

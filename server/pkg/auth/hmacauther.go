package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/metal-stack/security"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contextKey string

const (
	// HmacDefaultKey is a exported constant for convenience
	HmacDefaultKey = "4Rahs0WnJ4rJE8ZiwiLec62z"

	// hmacMethod fictive non-rest-method used for HMAC-Token
	hmacMethod = "GRPC"

	// hmacAuthtype reflects the application for that the hmac is used (tenant masterdata-management)
	hmacAuthtype = "tmdm"

	// lifetime of the hmac token
	hmacLifetime = 15 * time.Second

	contextKeyUser = contextKey("user")
)

// HMACAuther provides means for generation/encoding and decoding/validation for grpc.
// This code is potentially re-usable for all grpc-based clients/services
// that want to use hmac-Authentification.
type HMACAuther struct {
	logger   *zap.Logger
	hmacAuth *security.HMACAuth
}

// NewHMACAuther creates a new HMACAuther with the given hmac-pre-shared-key and user.
func NewHMACAuther(logger *zap.Logger, hmacKey string, user security.User) (*HMACAuther, error) {

	var hmacAuth *security.HMACAuth
	if hmacKey != "" {
		auth := security.NewHMACAuth(hmacAuthtype, []byte(hmacKey), security.WithLifetime(hmacLifetime), security.WithUser(user))
		hmacAuth = &auth

		a := &HMACAuther{
			logger:   logger,
			hmacAuth: hmacAuth,
		}

		return a, nil
	}

	return nil, fmt.Errorf("error creating auther - no hmacKey specified")
}

// Auth returns a new Context with the authenticated "user" from the current request.
// If there is no authentication info in the request, or the verification of the HMAC
// fails an Status-Error is returned with Code Unauthenticated.
//
//  see GetUser()
//
// Used on the service/server-side.
func (a *HMACAuther) Auth(ctx context.Context) (context.Context, error) {

	rqd := security.RequestData{
		Method:          hmacMethod,
		AuthzHeader:     metautils.ExtractIncoming(ctx).Get(security.AuthzHeaderKey),
		TimestampHeader: metautils.ExtractIncoming(ctx).Get(security.TsHeaderKey),
		SaltHeader:      metautils.ExtractIncoming(ctx).Get(security.SaltHeaderKey),
	}

	user, err := a.hmacAuth.UserFromRequestData(rqd)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	newCtx := context.WithValue(ctx, contextKeyUser, user)
	return newCtx, nil
}

// GetUser gets the authenticated user from the given context.
// Note that it is necessary to call Auth() in an interceptor to put the user in the context.
//
// May return nil if no user is authenticated or not of the correct type.
//
// Used on the service/server-side.
func GetUser(ctx context.Context) *security.User {
	user := ctx.Value(contextKeyUser)
	if user == nil {
		return nil
	}
	u, _ := user.(*security.User)
	return u
}

// GetRequestMetadata gets the current request metadata, refreshing
// tokens if required. This should be called by the transport layer on
// each request, and the data should be populated in headers or other
// context.
//
// Used on the client-side.
func (a *HMACAuther) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {

	headers := a.hmacAuth.AuthHeaders(hmacMethod, time.Now())
	return headers, nil
}

// RequireTransportSecurity indicates whether the credentials requires
// transport security.
//
// Used on the client-side.
func (a *HMACAuther) RequireTransportSecurity() bool {
	return true
}

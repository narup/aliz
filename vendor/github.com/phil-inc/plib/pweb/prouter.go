package pweb

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"log"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	"github.com/phil-inc/plib/core/util"
)

// PhilRouter wraps httprouter, which is non-compatible with http.Handler to make it
// compatible by implementing http.Handler into a httprouter.Handler function.
type PhilRouter struct {
	r   *httprouter.Router
	Ctx context.Context
}

// NewPhilRouter returns new PhilRouter which wraps the httprouter
func NewPhilRouter(ctx context.Context) *PhilRouter {
	return &PhilRouter{httprouter.New(), ctx}
}

func (s *PhilRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	origin := req.Header.Get("Origin")
	if origin == "" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		corsList := util.Config("cors.allowed.list")
		if strings.Contains(corsList, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			WriteError(w, Forbidden)
			return
		}
	}

	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Requested-With, X-App-Source, X-Request-Id")
	if req.Method == "OPTIONS" {
		w.(http.Flusher).Flush()
	}
	s.r.ServeHTTP(w, req)
}

// wrapper around httprouter's HTTP methods to make it compatible with http.Handler interface

// Get wraps httprouter's GET function
func (s *PhilRouter) Get(path string, handler http.Handler) {
	s.r.GET(path, wrapHandler(s.Ctx, handler))
}

// Post wraps httprouter's POST function
func (s *PhilRouter) Post(path string, handler http.Handler) {
	s.r.POST(path, wrapHandler(s.Ctx, handler))
}

// Put wraps httprouter's PUT function
func (s *PhilRouter) Put(path string, handler http.Handler) {
	s.r.PUT(path, wrapHandler(s.Ctx, handler))
}

// Delete wraps httprouter's DELETE function
func (s *PhilRouter) Delete(path string, handler http.Handler) {
	s.r.DELETE(path, wrapHandler(s.Ctx, handler))
}

// wrapHandler - The problem with httprouter is its non-compatibility with
// http.Handler. To make it compatible with existing middlewares and contexts.
// this function wraps our middleware stack – implementing http.Handler – into
// a httprouter.Handler function.
func wrapHandler(ctx context.Context, h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		//instead of passing extra params to handler function use context
		if ps != nil {
			ctxParams := context.WithValue(r.Context(), Params, ps)
			r = r.WithContext(ctxParams)
		}
		h.ServeHTTP(w, r)
	}
}

// ErrMissingRequiredData error to represent missing data error
var ErrMissingRequiredData = errors.New("missing required data")

//ErrNotRecognized error for any unrecognized client
var ErrNotRecognized = errors.New("not recognized")

// APIResponse response data representation for API
type APIResponse struct {
	Error  string      `json:"error,omitempty"`
	Status string      `json:"status,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

// Write - Reponse interface implementation
func (res APIResponse) Write(w http.ResponseWriter, r *http.Request) {
	if res.Status == "ERROR" {
		log.Printf("[ERROR][API][PATH: %s]:: Error handling request. ERROR: %s. User agent: %s", r.RequestURI, res.Error, r.Header.Get("User-Agent"))
	}
	WriteJSON(w, res)
}

// DataResponse creates new API data response using the resource
func DataResponse(data interface{}) APIResponse {
	return APIResponse{Error: "", Status: "OK", Data: data}
}

// StringErrorResponse constructs error response based on input
func StringErrorResponse(err string) APIResponse {
	return APIResponse{Error: err, Status: "ERROR", Data: nil}
}

//ErrorResponse constructs error response from the API
func ErrorResponse(err error) APIResponse {
	return APIResponse{Error: err.Error(), Status: "ERROR", Data: nil}
}

// RequestBody returns the request body
func RequestBody(r *http.Request) interface{} {
	return r.Context().Value(Body)
}

// SessionUserID returns user id of the current session
func SessionUserID(r *http.Request) string {
	if jwtClaims, ok := r.Context().Value(util.SessionUserKey).(jwt.MapClaims); ok {
		return jwtClaims["uid"].(string)
	}
	return ""
}

// UserRoles current user roles
func UserRoles(r *http.Request) []string {
	if jwtClaims, ok := r.Context().Value(util.SessionUserKey).(jwt.MapClaims); ok {
		return jwtClaims["uid"].([]string)
	}
	return make([]string, 0)
}

// QueryParamByName returns the request param by name
func QueryParamByName(name string, r *http.Request) string {
	return r.URL.Query().Get(name)
}

// QueryParamsByName returns the request param by name
func QueryParamsByName(name string, r *http.Request) []string {
	values := r.URL.Query()
	return values[name]
}

// ParamByName returns the request param by name
func ParamByName(name string, r *http.Request) string {
	params := r.Context().Value(Params).(httprouter.Params)
	return params.ByName(name)
}

//Authorize checks if given request is authorized
func Authorize(w http.ResponseWriter, r *http.Request) {
	sid := SessionUserID(r)
	uid := ParamByName("uid", r)

	if sid != uid {
		WriteError(w, Forbidden)
	}
}

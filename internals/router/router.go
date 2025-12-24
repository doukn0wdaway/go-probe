package router

import (
	"context"
	"encoding/json"
	"fmt"
	"myapp/internals/validation"
	"net/http"
	"reflect"

	"github.com/go-chi/chi/v5"
)

type Request[Path any, Query any, Body any] struct {
	Ctx      context.Context
	Original *http.Request
	Path     Path
	Query    Query
	Body     Body
}

type Route struct {
	Method  string
	Path    string
	Handler interface{}
}

func wrapHandler(h interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hType := reflect.TypeOf(h)
		if hType.Kind() != reflect.Func || hType.NumIn() != 1 {
			http.Error(w, "handler must be func(Request[...])(...)", http.StatusInternalServerError)
			return
		}

		reqType := hType.In(0)
		if reqType.Kind() != reflect.Struct {
			// Если это дженерик, будет reflect.Struct с type params
		}

		reqValue := reflect.New(reqType).Elem()
		reqValue.FieldByName("Ctx").Set(reflect.ValueOf(r.Context()))
		reqValue.FieldByName("Original").Set(reflect.ValueOf(r))

		// --- BODY ---
		bodyField := reqValue.FieldByName("Body")
		if r.Method != http.MethodGet && bodyField.IsValid() && bodyField.CanSet() {
			bodyPtr := reflect.New(bodyField.Type())
			decoder := json.NewDecoder(r.Body)
			// decoder.DisallowUnknownFields()
			if err := decoder.Decode(bodyPtr.Interface()); err != nil {
				http.Error(w, fmt.Sprintf("failed to parse body: %s", err.Error()), http.StatusBadRequest)
				return
			}
			bodyField.Set(bodyPtr.Elem())

			bodyIface := bodyField.Addr().Interface()
			if v, ok := bodyIface.(interface {
				Validate() validation.ValidationErrors
			}); ok {
				if errs := v.Validate(); len(errs) > 0 {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]any{"errors": errs})
					return
				}
			}
		}

		// --- QUERY ---
		queryField := reqValue.FieldByName("Query")
		if queryField.IsValid() && queryField.CanSet() {
			queryPtr := reflect.New(queryField.Type())
			queryMap := make(map[string]string)
			for key, values := range r.URL.Query() {
				if len(values) > 0 {
					queryMap[key] = values[0]
				}
			}
			b, _ := json.Marshal(queryMap)
			if err := json.Unmarshal(b, queryPtr.Interface()); err != nil {
				http.Error(w, fmt.Sprintf("failed to parse query: %s", err.Error()), http.StatusBadRequest)
				return
			}
			queryField.Set(queryPtr.Elem())
		}

		// --- PATH ---
		pathField := reqValue.FieldByName("Path")
		if pathField.IsValid() && pathField.CanSet() {
			pathPtr := reflect.New(pathField.Type())
			pathMap := make(map[string]string)
			params := chi.RouteContext(r.Context()).URLParams
			for i := 0; i < len(params.Keys); i++ {
				pathMap[params.Keys[i]] = params.Values[i]
			}
			b, _ := json.Marshal(pathMap)
			if err := json.Unmarshal(b, pathPtr.Interface()); err != nil {
				http.Error(w, fmt.Sprintf("failed to parse path params: %s", err.Error()), http.StatusBadRequest)
				return
			}
			pathField.Set(pathPtr.Elem())
		}

		// --- Вызов обработчика ---
		out := reflect.ValueOf(h).Call([]reflect.Value{reqValue})

		res := out[0].Interface()
		var err error
		if !out[1].IsNil() {
			err = out[1].Interface().(error)
		}
		if err != nil {
			http.Error(w, fmt.Sprintf("handler error: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
	}
}

var AllRegisteredRoutes []Route

type RegisterRouteFn func(method string, path string, handler interface{})

func CreateRegistrationFunction(r *chi.Mux) RegisterRouteFn {
	return func(method, path string, handler interface{}) {
		AllRegisteredRoutes = append(AllRegisteredRoutes, Route{
			Method:  method,
			Path:    path,
			Handler: handler,
		})

		r.MethodFunc(method, path, wrapHandler(handler))
	}
}

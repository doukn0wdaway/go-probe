package main

import (
	"fmt"
	"net/http"

	"myapp/internals/openapi"
	"myapp/internals/router"
	"myapp/internals/user"

	"github.com/go-chi/chi/v5"
)

func main() {

	r := chi.NewRouter()

	regFn := router.CreateRegistrationFunction(r)
	user.RegisterRoutes(regFn)

	r.MethodFunc(http.MethodGet, "/swagger", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		html := openapi.GenerateSwaggerDocHtml(router.AllRegisteredRoutes)
		w.Write([]byte(html))
	})

	// Swagger JSON
	r.MethodFunc(http.MethodGet, "/swagger/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		jsonStr := openapi.GenerateJsonString(router.AllRegisteredRoutes)
		w.Write([]byte(jsonStr))
	})

	fmt.Println(r.Routes())
	addr := "127.0.0.1:8080"
	fmt.Println("Server running on", addr)
	http.ListenAndServe(addr, r)
}

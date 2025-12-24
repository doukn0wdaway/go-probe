package user

import (
	"myapp/internals/router"
	"myapp/internals/validation"
)

type CreateUserRequestBody struct {
	Email    string `json:"email"`
	Age      int    `json:"age"`
	Username string `json:"username"`
}

type CreateUserResponse struct {
	ID string `json:"id"`
}

func (b *CreateUserRequestBody) Validate() validation.ValidationErrors {
	var errs validation.ValidationErrors

	if b.Email == "" {
		errs = append(errs, "email is required")
	}
	if b.Username == "" {
		errs = append(errs, "username is required")
	}
	if b.Age == 0 {
		errs = append(errs, "age is required")
	}

	return errs
}

func CreateUser(req router.Request[any, any, CreateUserRequestBody]) (CreateUserResponse, error) {
	return CreateUserResponse{ID: "123"}, nil
}

func RegisterRoutes(r router.RegisterRouteFn) {
	r("POST", "/users", CreateUser)
}

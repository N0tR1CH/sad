package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/N0tR1CH/sad/internal/data"
	"github.com/N0tR1CH/sad/internal/mailer"
	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/components"
	"github.com/N0tR1CH/sad/views/pages"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

func (app *application) loginHandler(c echo.Context) error {
	return views.Render(
		c,
		http.StatusOK,
		pages.LoginPage(
			pages.LoginPageProps{
				PageTitle:       "Auth Page",
				PageDescription: "Provide your email and we will redirect you to correct action.",
				EmailFieldProps: pages.EmailFieldProps{
					IsInputWrong: false,
					InputValue:   "",
				},
				Fields: nil,
			},
		),
	)
}

func (app *application) createUserHandler(c echo.Context) error {
	app.sessionManager.Remove(c.Request().Context(), "usernameRight")
	app.sessionManager.Remove(c.Request().Context(), "passwordRight")

	var input struct {
		Email    string `form:"email" validate:"required,email"`
		Username string `form:"username" validate:"required,alphanum,min=3,max=30"`
		Password string `form:"password" validate:"required,min=8,max=64,containsany=!@#?*,containsany=ABCDEFGHIJKLMNOPQRSTUVWXYZ,containsany=123456789"`
	}

	if err := c.Bind(&input); err != nil {
		return views.Render(
			c,
			http.StatusOK,
			pages.LoginFormBody(
				pages.LoginPageProps{
					PageTitle:       "Auth Page",
					PageDescription: "Provide your email and we will redirect you to correct action.",
					EmailFieldProps: pages.EmailFieldProps{
						IsInputWrong: false,
						InputValue:   input.Email,
						ErrMsg:       "Values could not be bind.",
					},
					Fields: nil,
				},
			),
		)
	}

	if err := c.Validate(&input); err != nil {
		return views.Render(
			c,
			http.StatusOK,
			pages.LoginFormBody(
				pages.LoginPageProps{
					PageTitle:       "Auth Page",
					PageDescription: "Provide your email and we will redirect you to correct action.",
					EmailFieldProps: pages.EmailFieldProps{
						IsInputWrong: false,
						InputValue:   input.Email,
						ErrMsg:       "Values could not be validated.",
					},
					Fields: nil,
				},
			),
		)
	}

	u := &data.User{
		Email:     input.Email,
		Name:      input.Username,
		Activated: false,
	}
	if err := u.Password.Set(input.Password); err != nil {
		return views.Render(
			c,
			http.StatusOK,
			pages.LoginFormBody(
				pages.LoginPageProps{
					PageTitle:       "Auth Page",
					PageDescription: "Provide your email and we will redirect you to correct action.",
					EmailFieldProps: pages.EmailFieldProps{
						IsInputWrong: false,
						InputValue:   input.Email,
						ErrMsg:       "Password could not be set.",
					},
					Fields: nil,
				},
			),
		)
	}

	if err := app.models.Users.Insert(u); err != nil {
		return views.Render(
			c,
			http.StatusOK,
			pages.LoginFormBody(
				pages.LoginPageProps{
					PageTitle:       "Auth Page",
					PageDescription: "Provide your email and we will redirect you to correct action.",
					EmailFieldProps: pages.EmailFieldProps{
						IsInputWrong: false,
						InputValue:   input.Email,
						ErrMsg:       "User could not be created.",
					},
					Fields: nil,
				},
			),
		)
	}

	c.Response().Header().Set("HX-Push-Url", "/")
	c.Response().Header().Set("HX-Retarget", "#app-main-container")
	c.Response().Header().Set("HX-Reswap", "innerHTML")
	discussions, err := app.models.Discussions.GetAll()
	if err != nil {
		return views.Render(
			c,
			http.StatusOK,
			pages.LoginFormBody(
				pages.LoginPageProps{
					PageTitle:       "Auth Page",
					PageDescription: "Provide your email and we will redirect you to correct action.",
					EmailFieldProps: pages.EmailFieldProps{
						IsInputWrong: false,
						InputValue:   input.Email,
						ErrMsg:       "Problem with our service.",
					},
					Fields: nil,
				},
			),
		)
	}

	app.startBackgroundJob(func() {
		if err := app.mailer.Send(
			u.Email,
			mailer.MailSubject(),
			mailer.PlainBody(u.ID),
			mailer.HtmlBody(u.ID),
		); err != nil {
			app.logger.Info("user#create", "Err", err.Error())
		}
	})

	views.Render(
		c,
		http.StatusOK,
		components.WrapAndSwap(
			pages.SuccessfulAlert(),
			"",
			"afterbegin:#app-main-container",
		),
	)

	return views.Render(
		c,
		http.StatusOK,
		pages.HomeBody(
			pages.NewHomeViewModel(discussions),
		),
	)
}

func (app *application) validateUserEmailHandler(c echo.Context) error {
	app.sessionManager.Remove(c.Request().Context(), "usernameRight")
	app.sessionManager.Remove(c.Request().Context(), "passwordRight")

	var input struct {
		Email string `query:"email" validate:"required,email"`
	}

	if err := c.Bind(&input); err != nil {
		c.Response().Header().Set("HX-Redirect", "/login")
		return c.NoContent(http.StatusOK)
	}

	var errMsg string
	if err := c.Validate(&input); err != nil {
		tag := err.(validator.ValidationErrors)[0].Tag()
		switch tag {
		case "required":
			errMsg = "Field is required."
		case "email":
			errMsg = "Field must be of email format."
		}
	}

	// When email is of invalid format return default page
	if errMsg != "" {
		return views.Render(
			c,
			http.StatusOK,
			pages.LoginFormBody(
				pages.LoginPageProps{
					PageTitle:       "Auth Page",
					PageDescription: "Provide your email and we will redirect you to correct action.",
					EmailFieldProps: pages.EmailFieldProps{
						IsInputWrong: true,
						InputValue:   input.Email,
						ErrMsg:       errMsg,
					},
					Fields: nil,
				},
			),
		)
	}

	if _, err := app.models.Users.GetByEmail(input.Email); err != nil && errors.Is(err, data.ErrRecordNotFound) {
		c.Response().Header().Set("HX-Push-Url", "/register")
		return views.Render(
			c,
			http.StatusOK,
			pages.LoginFormBody(
				pages.LoginPageProps{
					PageTitle:       "Register",
					PageDescription: "Insert data in order to create new account.",
					EmailFieldProps: pages.EmailFieldProps{
						IsInputWrong: false,
						InputValue:   input.Email,
						ErrMsg:       errMsg,
					},
					Fields: pages.RegisterFields(),
				},
			),
		)
	}
	c.Response().Header().Set("HX-Push-Url", "/login")

	return views.Render(
		c,
		http.StatusOK,
		pages.LoginFormBody(
			pages.LoginPageProps{
				PageTitle:       "Login",
				PageDescription: "Insert data in order sign in.",
				EmailFieldProps: pages.EmailFieldProps{
					IsInputWrong: false,
					InputValue:   input.Email,
					ErrMsg:       errMsg,
				},
				Fields: pages.LoginFields(),
			},
		),
	)
}

func (app *application) validateUserUsernameHandler(c echo.Context) error {
	app.sessionManager.Remove(c.Request().Context(), "usernameRight")
	var input struct {
		Username string `query:"username" validate:"required,alphanum,min=3,max=30"`
	}
	if err := c.Bind(&input); err != nil {
		return views.Render(
			c,
			http.StatusBadRequest,
			pages.LoginErrorMessage("Bad request"),
		)
	}

	if err := c.Validate(&input); err != nil {
		var errMsg string
		vErr := err.(validator.ValidationErrors)[0]
		switch vErr.Tag() {
		case "required":
			errMsg = "Field is required."
		case "alphanum":
			errMsg = "Field must consist of only alphanumeric characters."
		case "min":
			minNumOfChars := vErr.Param()
			errMsg = fmt.Sprintf("Field minimum length is %s.", minNumOfChars)
		case "max":
			maxNumOfChars := vErr.Param()
			errMsg = fmt.Sprintf("Field maximum length is %s.", maxNumOfChars)
		}
		return views.Render(
			c,
			http.StatusOK,
			pages.UsernameField(
				pages.UsernameFieldProps{
					IsInputWrong: true,
					InputValue:   input.Username,
					ErrMsg:       errMsg,
				},
			),
		)
	}

	app.sessionManager.Put(c.Request().Context(), "usernameRight", true)
	parsedUrl, err := url.Parse(c.Request().Header.Get("HX-Current-URL"))
	if err != nil {
		c.Response().Header().Set("HX-Redirect", "/login")
		return c.NoContent(http.StatusOK)
	}
	submitButtonAction := parsedUrl.Path

	return views.Render(
		c,
		http.StatusOK,
		pages.UsernameField(
			pages.UsernameFieldProps{
				IsInputWrong: false,
				InputValue:   input.Username,
				IncludeSubmitButton: func() bool {
					return app.sessionManager.GetBool(
						c.Request().Context(),
						"passwordRight",
					) && app.sessionManager.GetBool(
						c.Request().Context(),
						"usernameRight",
					)
				}(),
				SubmitButtonAction: submitButtonAction,
			},
		),
	)
}

func (app *application) validateUserPasswordHandler(c echo.Context) error {
	app.sessionManager.Remove(c.Request().Context(), "passwordRight")
	var input struct {
		Password string `query:"password" validate:"required,min=8,max=64,containsany=!@#?*,containsany=ABCDEFGHIJKLMNOPQRSTUVWXYZ,containsany=123456789"`
	}
	if err := c.Bind(&input); err != nil {
		return views.Render(
			c,
			http.StatusBadRequest,
			pages.LoginErrorMessage("Bad request"),
		)
	}

	if err := c.Validate(&input); err != nil {
		var errMsg string
		vErr := err.(validator.ValidationErrors)[0]
		switch vErr.Tag() {
		case "required":
			errMsg = "Field is required."
		case "min":
			minNumOfChars := vErr.Param()
			errMsg = fmt.Sprintf("Field minimum length is %s.", minNumOfChars)
		case "max":
			maxNumOfChars := vErr.Param()
			errMsg = fmt.Sprintf("Field maximum length is %s.", maxNumOfChars)
		case "containsany":
			switch vErr.Param() {
			case "!@#?*":
				errMsg = fmt.Sprintf(
					"Field must contain any of these characters: \"%s\".",
					vErr.Param(),
				)
			case "ABCDEFGHIJKLMNOPQRSTUVWXYZ":
				errMsg = "Field must contain atleast one big letter."
			case "123456789":
				errMsg = "Field must contain atleast one digit."
			}
		}
		return views.Render(
			c,
			http.StatusOK,
			pages.PasswordField(
				pages.PasswordFieldProps{
					IsInputWrong: true,
					InputValue:   input.Password,
					ErrMsg:       errMsg,
				},
			),
		)
	}

	app.sessionManager.Put(c.Request().Context(), "passwordRight", true)
	parsedUrl, err := url.Parse(c.Request().Header.Get("HX-Current-URL"))
	if err != nil {
		c.Response().Header().Set("HX-Redirect", "/login")
		return c.NoContent(http.StatusOK)
	}
	submitButtonAction := parsedUrl.Path

	return views.Render(
		c,
		http.StatusOK,
		pages.PasswordField(
			pages.PasswordFieldProps{
				IsInputWrong: false,
				InputValue:   input.Password,
				IncludeSubmitButton: func() bool {
					switch submitButtonAction {
					case "/login":
						return true
					case "/register":
						return app.sessionManager.GetBool(
							c.Request().Context(),
							"passwordRight",
						) && app.sessionManager.GetBool(
							c.Request().Context(),
							"usernameRight",
						)
					default:
						return false
					}
				}(),
				SubmitButtonAction: submitButtonAction,
			},
		),
	)
}

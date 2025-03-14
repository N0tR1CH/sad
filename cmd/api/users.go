package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/N0tR1CH/sad/internal/data"
	"github.com/N0tR1CH/sad/internal/mailer"
	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/components"
	"github.com/N0tR1CH/sad/views/pages"
	"github.com/a-h/templ"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/h2non/bimg"
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

func (app *application) deauthenticateUserHandler(c echo.Context) error {
	var input struct {
		ID string `param:"id" validate:"number"`
	}
	if err := c.Bind(&input); err != nil {
		app.logger.Error("app#deauthenticateUserHandler-input-binding", "error", err.Error())
		return err
	}
	if err := c.Validate(&input); err != nil {
		app.logger.Error("app#deauthenticateUserHandler-validation-binding", "error", err.Error())
		return c.String(http.StatusBadRequest, "id must be a number")
	}

	currUId := app.sessionManager.GetInt(c.Request().Context(), "userID")
	if currUId == 0 {
		return c.String(http.StatusBadRequest, "user not logged in")
	}

	if id, err := strconv.Atoi(input.ID); err != nil || id != currUId {
		app.logger.Error(
			"app#deauthenticateUserHandler",
			"msg", "conversion of id param or id param dont match real user id",
			"error", err.Error(),
		)
		return err
	}

	exists, err := app.models.Users.Exists(currUId)
	if err != nil {
		app.logger.Error("app#deauthenticateUserHandler-database-problem", "error", err.Error())
		return err
	}

	if !exists {
		return c.String(http.StatusBadRequest, "user with such id does not exist or trying to log out other user")
	}

	app.sessionManager.Remove(c.Request().Context(), "userID")
	c.Response().Header().Set("HX-Location", "/")
	app.sessionManager.Put(
		c.Request().Context(),
		"alert",
		components.AlertProps{
			Title: "Logged out",
			Text:  "You have been successfully logged out!",
			Icon:  components.Success,
		},
	)
	return c.NoContent(http.StatusOK)
}

func (app *application) authenticateUserHandler(c echo.Context) error {
	app.sessionManager.Remove(c.Request().Context(), "usernameRight")
	app.sessionManager.Remove(c.Request().Context(), "passwordRight")

	var input struct {
		Email    string `form:"email" validate:"required,email"`
		Password string `form:"password" validate:"required,min=8,max=64,containsany=!@#?*,containsany=ABCDEFGHIJKLMNOPQRSTUVWXYZ,containsany=123456789"`
	}

	if err := c.Bind(&input); err != nil {
		app.logger.Error("app#authenticateUserHandler-input-binding", "error", err.Error())
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
		app.logger.Error("app#authenticateUserHandler-user-validation", "error", err.Error())
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

	u, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		app.logger.Error("app#authenticateUserHandler-user-retrieval", "error", err.Error())
		if !errors.Is(err, data.ErrRecordNotFound) {
			return err
		}

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
						ErrMsg:       "User with such email or password doesn't exist",
					},
					Fields: nil,
				},
			),
		)
	}

	match, err := u.Password.Match(input.Password)
	if err != nil {
		app.logger.Error("app#authenticateUserHandler-password-matching", "error", err.Error())
		return err
	}

	if !match {
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
						ErrMsg:       "email or password are not right",
					},
					Fields: nil,
				},
			),
		)
	}

	if err := app.sessionManager.RenewToken(c.Request().Context()); err != nil {
		app.logger.Error("app#authenticateUserHandler-token-renewal", "error", err.Error())
		return err
	}
	app.sessionManager.Put(c.Request().Context(), "userID", u.ID)

	app.sessionManager.Put(
		c.Request().Context(),
		"alert",
		components.AlertProps{
			Title: "Logged in",
			Text:  "You have been successfully logged in!",
			Icon:  components.Success,
		},
	)
	c.Response().Header().Set("HX-Push-Url", "/")
	c.Response().Header().Set("HX-Retarget", "#app-main-container")
	c.Response().Header().Set("HX-Reswap", "innerHTML")
	return views.Render(
		c,
		http.StatusOK,
		pages.AfterLoginPage(u.ID),
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

	var resPath string
	file, err := c.FormFile("avatar")
	if file != nil {
		if err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer func() {
			closeErr := src.Close()
			app.logger.Error("closing file", "error", closeErr.Error())
		}()
		fileBytes, err := io.ReadAll(src)
		if err != nil {
			return err
		}
		webpImgBytes, err := bimg.NewImage(fileBytes).Convert(bimg.WEBP)
		resPath = fmt.Sprintf("/public/avatars/%s.webp", uuid.NewString())
		filepath := fmt.Sprintf("cmd/web%s", resPath)
		f, err := os.Create(filepath)
		if err != nil {
			return err
		}
		defer func() {
			_, err = f.Write(webpImgBytes)
		}()
	}

	app.logger.Info("avatar!!!", "respath", resPath)

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
		AvatarSrc: resPath,
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

	t, err := app.models.Tokens.New(
		u.ID,
		24*time.Hour,
		data.TokenType(data.TokenTypeActivation),
	)
	if err != nil {
		app.logger.Error("TokenGeneration", "error", err.Error())
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
						ErrMsg:       "Could not generate token. We're sorry. Try Again.",
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
			mailer.PlainBody(u.ID, t.PlainText),
			mailer.HtmlBody(u.ID, t.PlainText),
		); err != nil {
			app.logger.Error(
				"user#create while sending email",
				"Err", err.Error(),
			)
		}
	})

	c.Response().Header().Set("HX-Push-Url", "/")
	c.Response().Header().Set("HX-Retarget", "#app-main-container")
	c.Response().Header().Set("HX-Reswap", "innerHTML")
	c.Set("activationSuccess", struct{}{})
	return views.Render(
		c,
		http.StatusOK,
		pages.HomeBody(),
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

func (app *application) getUserActivationSectionHandler(c echo.Context) error {
	var input struct {
		Id    string `param:"id" validate:"required,number"`
		Token string `query:"token" validate:"required,len=32,base32"`
	}

	if err := c.Bind(&input); err != nil {
		app.logger.Error(
			"app#getUserActivationSectionHandler - Values could no be binded",
			"Err",
			err.Error(),
		)
		return c.String(http.StatusOK, "Problem handling the request")
	}
	if err := c.Validate(&input); err != nil {
		app.logger.Error(
			"app#getUserActivationSectionHandler - Values could no be validated",
			"Err",
			err.Error(),
		)
		return c.String(
			http.StatusOK,
			"There was a problem with validating the parameters, please visit link from the email again.",
		)
	}

	return views.Render(
		c,
		http.StatusOK,
		pages.ActivationPage(pages.NewActivationPageProps(input.Id, input.Token)),
	)
}

func (app *application) updateUserActivationStatusHandler(c echo.Context) error {
	var input struct {
		Id    string `param:"id" validate:"required,number"`
		Token string `form:"token" validate:"required,len=32,base32"`
	}

	if err := c.Bind(&input); err != nil {
		app.logger.Error(
			"app#updateUserActivationSectionHandler - Values could no be binded",
			"Err",
			err.Error(),
		)
		return views.Render(
			c,
			http.StatusOK,
			pages.ActivationPageError("Values could not be binded to the request!"),
		)
	}
	if err := c.Validate(&input); err != nil {
		app.logger.Error(
			"app#updateUserActivationSectionHandler - Values could not be validated",
			"Err",
			err.Error(),
			"Input",
			input,
		)
		return views.Render(
			c,
			http.StatusOK,
			pages.ActivationPageError("Values could not be validated."),
		)
	}

	u, err := app.models.Users.GetForToken(data.TokenTypeActivation, strings.Trim(input.Token, "="))
	if err != nil || strconv.Itoa(u.ID) != input.Id {
		app.logger.Error(
			"app#updateUserActivationSectionHandler - While getting user by token",
			"Err",
			err.Error(),
		)
		return views.Render(
			c,
			http.StatusOK,
			pages.ActivationPageError("Account could not be activated sorry!"),
		)
	}

	u.Activated = true
	if err := app.models.Users.Update(u); err != nil {
		app.logger.Error(
			"app#updateUserActivationSectionHandler - While updating the user",
			"Err",
			err.Error(),
		)
		return views.Render(
			c,
			http.StatusOK,
			pages.ActivationPageError("Account could not be activated sorry!"),
		)
	}

	if err := app.models.Tokens.DeleteAllForUser(data.TokenTypeActivation, u.ID); err != nil {
		app.logger.Error(
			"app#updateUserActivationSectionHandler - while deleting token for the user",
			"Err",
			err.Error(),
		)
		return views.Render(
			c,
			http.StatusOK,
			pages.ActivationPageError("Account could not be activated sorry!"),
		)
	}

	c.Response().Header().Set("HX-Reswap", "outerHTML")
	return views.Render(
		c,
		http.StatusOK,
		pages.ActivationPageSuccess("Account is activated! Enjoy our service."),
	)
}

func (app *application) getUserAvatarHandler(c echo.Context) error {
	var input struct {
		ID string `param:"id" validate:"required,number"`
	}
	if err := c.Bind(&input); err != nil {
		return err
	}
	if err := c.Validate(&input); err != nil {
		return err
	}
	uID, err := strconv.Atoi(input.ID)
	if err != nil {
		return err
	}
	src, err := app.models.Users.AvatarSrcByID(uID)
	if err != nil {
		return err
	}
	var t templ.Component
	if src != "" {
		t = components.AvatarImg(src)
	} else {
		t = components.AvatarPlaceHolder()
	}
	return views.Render(c, http.StatusOK, t)
}

func (app *application) getUserHandler(c echo.Context) error {
	var input struct {
		ID string `param:"id" validate:"required,number"`
	}
	if err := c.Bind(&input); err != nil {
		return err
	}
	if err := c.Validate(&input); err != nil {
		tag := err.(validator.ValidationErrors)[0].Tag()
		var errMsg string
		switch tag {
		case "required":
			errMsg = "user id is required"
		case "number":
			errMsg = "user id must be a number"
		}
		vm := pages.UserPageError(errMsg)
		if c.Get("HTMX").(bool) {
			return views.Render(c, http.StatusNotFound, pages.UserPageBody(vm))
		}
		return views.Render(c, http.StatusNotFound, pages.UserPage(vm))
	}
	uID, err := strconv.Atoi(input.ID)
	if err != nil {
		return err
	}
	email, err := app.models.Users.GetEmail(uID)
	if err != nil {
		return err
	}
	u, err := app.models.Users.GetByEmail(email)
	if err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return views.Render(c, http.StatusNotFound, pages.Page404())
		}
		return err
	}
	vm := pages.UserPageViewModel{
		Id:          u.ID,
		Name:        u.Name,
		AvatarSrc:   u.AvatarSrc,
		Description: u.Description,
		Activated:   u.Activated,
	}
	if c.Get("HTMX").(bool) && !c.Get("Boosted").(bool) {
		return views.Render(c, http.StatusOK, pages.UserPageBody(vm))
	}
	return views.Render(c, http.StatusOK, pages.UserPage(vm))
}

func (app *application) editUserHandler(c echo.Context) error {
	var input struct {
		ID string `param:"id" validate:"required,number"`
	}
	if err := c.Bind(&input); err != nil {
		return err
	}
	if err := c.Validate(&input); err != nil {
		return err
	}
	uID, err := strconv.Atoi(input.ID)
	if err != nil {
		return err
	}
	description, err := app.models.Users.GetDescription(uID)
	if err != nil {
		return err
	}
	return views.Render(
		c,
		http.StatusOK,
		components.EditUserForm(
			components.EditUserFormViewModel{
				Id:          uID,
				Description: description,
				ErrMsg:      "",
			},
		),
	)
}

func (app *application) updateUserHandler(c echo.Context) error {
	var input struct {
		ID          string `param:"id" validate:"required,number"`
		Description string `form:"description" validate:"required,max=255"`
	}
	if err := c.Bind(&input); err != nil {
		return err
	}
	if err := c.Validate(&input); err != nil {
		uID, err := strconv.Atoi(input.ID)
		if err != nil {
			return err
		}
		description, err := app.models.Users.GetDescription(uID)
		if err != nil {
			return err
		}
		return views.Render(
			c,
			http.StatusBadRequest,
			components.EditUserForm(
				components.EditUserFormViewModel{
					Id:          uID,
					Description: description,
					ErrMsg:      "Validation problem",
				},
			),
		)
	}
	uID, err := strconv.Atoi(input.ID)
	if err != nil {
		return err
	}
	email, err := app.models.Users.GetEmail(uID)
	if err != nil {
		return err
	}
	u, err := app.models.Users.GetByEmail(email)
	if err != nil {
		return err
	}
	if u.ID != c.Get("userID").(int) {
		return c.NoContent(http.StatusUnauthorized)
	}
	if has, err := app.models.Users.HasRole(uID, "admin"); err != nil || !has {
		return views.Render(
			c,
			http.StatusOK,
			components.EditUserForm(
				components.EditUserFormViewModel{
					Id:          uID,
					Description: u.Description,
					ErrMsg:      "Not authorized",
				},
			),
		)
	}
	u.Description = input.Description
	if err := app.models.Users.Update(u); err != nil {
		return err
	}
	return views.Render(
		c,
		http.StatusOK,
		pages.AfterUserEdit(
			u.Description,
			u.ID,
		),
	)
}

func (app *application) getReportUserFormHandler(c echo.Context) error {
	var input struct {
		ID string `param:"id" validate:"required,number"`
	}
	if err := c.Bind(&input); err != nil {
		return err
	}
	if err := c.Validate(&input); err != nil {
		return err
	}
	userId, err := strconv.Atoi(input.ID)
	if err != nil {
		return err
	}
	discussionId, err := strconv.Atoi(c.QueryParam("discussionId"))
	if err != nil {
		app.logger.Error("in app#getReportUserFormHandler", "discussionId", discussionId, "err", err.Error())
	}
	commentId, err := strconv.Atoi(c.QueryParam("commentId"))
	if err != nil {
		app.logger.Error("in app#getReportUserFormHandler", "commentId", commentId, "err", err.Error())
	}
	if discussionId == 0 && commentId == 0 ||
		discussionId != 0 && commentId != 0 {
		return errors.New("can only report for discussion or category")
	}
	return views.Render(
		c,
		http.StatusOK,
		components.ReportUserForm(
			components.ReportUserFormViewModel{
				UserId:       userId,
				DiscussionId: discussionId,
				CommentId:    commentId,
			},
		),
	)
}

func (app *application) reportUserHandler(c echo.Context) error {
	var input struct {
		ID     string `param:"id" validate:"required,number"`
		Reason string `form:"reason" validate:"required,max=255"`
	}
	if err := c.Bind(&input); err != nil {
		return err
	}
	if err := c.Validate(&input); err != nil {
		return err
	}
	userId, err := strconv.Atoi(input.ID)
	if err != nil {
		return err
	}
	if c.Get("userID").(int) == userId {
		return errors.New("user cannot report himself")
	}
	if admin, err := app.models.Users.HasRole(
		userId,
		"admin",
	); err != nil || admin {
		if err != nil {
			return err
		}
		if admin {
			return c.String(
				http.StatusBadRequest,
				"admin cannot be reported",
			)
		}
	}
	discussionId, err := strconv.Atoi(c.QueryParam("discussionId"))
	if err != nil {
		app.logger.Error(
			"in app#getReportUserFormHandler",
			"discussionId", discussionId,
			"err", err.Error(),
		)
	}
	commentId, err := strconv.Atoi(c.QueryParam("commentId"))
	if err != nil {
		app.logger.Error(
			"in app#getReportUserFormHandler",
			"commentId", commentId,
			"err", err.Error(),
		)
	}
	if discussionId == 0 && commentId == 0 ||
		discussionId != 0 && commentId != 0 {
		return errors.New("can only report for discussion or category")
	}
	r := new(data.Report)
	r.ReportedUserID = userId
	r.UserID = c.Get("userID").(int)
	r.DiscussionID = discussionId
	r.CommentID = commentId
	r.Reason = input.Reason
	if err := app.models.Reports.Insert(r); err != nil {
		if errors.Is(err, data.ErrUniquenessViolation) {
			return c.String(
				http.StatusBadRequest,
				"You can't report same user twice for the same thing",
			)
		}
		return err
	}
	return c.NoContent(http.StatusOK)
}

func (app *application) banUserHandler(c echo.Context) error {
	var input struct {
		ID string `param:"id" validate:"required,number"`
	}
	if err := c.Bind(&input); err != nil {
		return fmt.Errorf("in app#banUserHandler while binding: %w", err)
	}
	if err := c.Validate(&input); err != nil {
		return fmt.Errorf("in app#banUserHandler while validating: %w", err)
	}
	uId, err := strconv.Atoi(input.ID)
	if err != nil {
		return fmt.Errorf("in app#banUserHandler while converting input: %w", err)
	}
	if err := app.models.Users.Ban(uId); err != nil {
		return fmt.Errorf("in app#banUserHandler while banning user: %w", err)
	}
	return c.NoContent(http.StatusOK)
}

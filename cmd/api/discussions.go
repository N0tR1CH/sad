package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/N0tR1CH/sad/internal/data"
	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/components"
	"github.com/N0tR1CH/sad/views/pages"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/yuin/goldmark"
)

func (app *application) newDiscussionHandler(c echo.Context) error {
	categories, err := app.models.Categories.GetAll()
	if err != nil {
		return err
	}
	cps := make(components.CategoriesProps, len(categories))
	for i := range cps {
		cps[i].ID = categories[i].ID
		cps[i].Name = categories[i].Name
	}

	dfp := components.DiscussionFormProps{
		ResourceUrl: c.QueryParam("url"),
		Categories:  cps,
	}

	if _, HTMX := c.Request().Header[http.CanonicalHeaderKey("HX-Request")]; HTMX {
		return views.Render(c, http.StatusOK, components.DiscussionForm(dfp))
	}
	return views.Render(c, http.StatusOK, pages.NewDiscussionPage(dfp))
}

func (app *application) getDiscussionsHandler(c echo.Context) error {
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil {
		page = 1
	}
	app.logger.Info("app#getDiscussionsHandler", "page", page)
	category := c.QueryParam("category")
	discussions, err := app.models.Discussions.GetAll(category, page)
	if err != nil {
		app.logger.Error("app#getDiscussionsHandler", "err", err.Error())
		return c.String(
			http.StatusInternalServerError,
			"Couldn't retrieve discussions from the server!",
		)
	}

	if activeCategoryId := c.QueryParam(
		"activeCategoryId",
	); activeCategoryId != "" && c.Get("HTMX").(bool) {
		activeCategoryId, err := strconv.Atoi(activeCategoryId)
		if err != nil {
			return err
		}
		categories, err := app.models.Categories.GetAll()
		if err != nil {
			return err
		}
		cps := make(components.CategoriesProps, len(categories))
		for i := range cps {
			cps[i].ID = categories[i].ID
			cps[i].Name = categories[i].Name
			if cps[i].ID == activeCategoryId {
				cps[i].Active = true
			}
		}
		views.Render(c, http.StatusOK, components.CategoriesOob(cps))
	}

	dcvms := make([]components.DiscussionCardViewModel, len(discussions))
	for i := 0; i < len(discussions); i++ {
		dcvms[i] = components.DiscussionCardViewModel{
			ImgSrc:    discussions[i].PreviewSrc,
			CardTitle: discussions[i].Title,
			Id:        discussions[i].ID,
		}
	}

	if c.Get("HTMX").(bool) {
		return views.Render(
			c,
			http.StatusOK,
			components.DiscussionCards(dcvms, page+1, category),
		)
	}
	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

func (app *application) getDiscussionHandler(c echo.Context) error {
	var input struct {
		Id string `param:"id" validate:"number,required"`
	}
	if err := c.Bind(&input); err != nil {
		return err
	}
	if err := c.Validate(&input); err != nil {
		return err
	}
	discussionId, err := strconv.Atoi(input.Id)
	if err != nil {
		return err
	}

	d, err := app.models.Discussions.Get(int64(discussionId))
	if err != nil {
		app.logger.Error("in app#getDiscussionHandler", "error", err.Error())
		app.sessionManager.Put(
			c.Request().Context(),
			"alert",
			components.AlertProps{
				Title: "No discussion!",
				Text:  "Sorry but this discussion does not exist",
				Icon:  components.Warning,
			},
		)
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}

	imgSrc, err := app.models.Users.AvatarSrcByID(d.UserId)
	if err != nil {
		return err
	}
	username, err := app.models.Users.GetUsername(d.UserId)
	if err != nil {
		return err
	}

	dvm := components.DiscussionViewModel{
		Id:          d.ID,
		Title:       d.Title,
		ResourceUrl: d.Url,
		Dtvm: components.DiscussionTopViewModel{
			Date:     d.CreatedAt.Format(time.ANSIC),
			ImgSrc:   imgSrc,
			Username: username,
		},
	}
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(d.Description), &buf); err != nil {
		return err
	}
	dvm.Description = views.Unsafe(buf.String())

	if c.Get("HTMX").(bool) {
		return views.Render(
			c,
			http.StatusOK,
			pages.DiscussionPageBody(pages.DiscussionPageProps{Dvm: dvm}),
		)
	}
	return views.Render(
		c,
		http.StatusOK,
		pages.DiscussionPage(
			pages.DiscussionPageProps{Dvm: dvm},
		),
	)
}

func (app *application) createDiscussionHandler(c echo.Context) error {
	var input struct {
		Title       string `form:"title" validate:"required,max=130"`
		Description string `form:"description" validate:"required,max=4000"`
		Url         string `form:"url" validate:"required,url"`
		CategoryId  string `form:"categories" validate:"required,number"`
	}

	if err := c.Bind(&input); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}

	if err := c.Validate(&input); err != nil {
		errs := make([]string, 0, len(err.(validator.ValidationErrors)))
		for _, ve := range err.(validator.ValidationErrors) {
			switch ve.Tag() {
			case "required":
				errs = append(
					errs,
					fmt.Sprintf("Field '%s' can't be blank.", ve.Field()),
				)
			case "max":
				errs = append(
					errs,
					fmt.Sprintf(
						"Field '%s' maximum length is %s characters.",
						ve.Field(),
						ve.Param(),
					),
				)
			case "number":
				errs = append(
					errs,
					fmt.Sprintf(
						"Field '%s' must be a number",
						ve.Field(),
					),
				)
			case "url":
				errs = append(
					errs,
					fmt.Sprintf("Invalid URL format."),
				)
			default:
				errs = append(errs, ve.Error())
			}
		}
		return views.Render(
			c,
			http.StatusBadRequest,
			components.DiscussionFormErrors(errs),
		)
	}

	previewSrc, err := app.services.ChromeDp.GenScreenshot(input.Url)
	if err != nil {
		return views.Render(
			c,
			http.StatusBadRequest,
			components.DiscussionFormErrors([]string{"Problem with the server"}),
		)
	}
	d := &data.Discussion{
		Title:       input.Title,
		Url:         input.Url,
		Description: input.Description,
		PreviewSrc:  previewSrc,
		UserId:      c.Get("userID").(int),
	}

	cID, err := strconv.Atoi(input.CategoryId)
	if err != nil {
		app.logger.Error("app#createDiscussionHandler", "err", err.Error())
	}
	d.CategoryID = cID

	if err := app.models.Discussions.Insert(d); err != nil {
		app.logger.Error("app#createDiscussionHandler", "err", err.Error())
		return views.Render(
			c,
			http.StatusBadRequest,
			components.DiscussionFormErrors([]string{"Problem with the server"}),
		)
	}

	c.Response().Header().Set("HX-Location", "/")
	app.sessionManager.Put(
		c.Request().Context(),
		"alert",
		components.AlertProps{
			Title: "Discussion created!",
			Text:  "You have successfully created a discussion!",
			Icon:  components.Success,
		},
	)
	return c.NoContent(http.StatusOK)
}

func (app *application) validateDiscussionTitleHandler(c echo.Context) error {
	var input struct {
		Title string `query:"title" validate:"required,max=130"`
	}
	if err := c.Bind(&input); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}
	if err := c.Validate(&input); err != nil {
		msg := ""
		for _, ve := range err.(validator.ValidationErrors) {
			switch ve.Tag() {
			case "required":
				msg = "Field is required"
			case "max":
				msg = "Too many characters"
			}
		}
		return views.Render(c, http.StatusOK, components.DiscussionFormErrorField(msg))
	}
	return c.String(http.StatusOK, "")
}

func (app *application) validateDiscussionDescriptionHandler(c echo.Context) error {
	var input struct {
		Description string `query:"description" validate:"required,max=4000"`
	}
	if err := c.Bind(&input); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}
	if err := c.Validate(&input); err != nil {
		msg := ""
		for _, ve := range err.(validator.ValidationErrors) {
			switch ve.Tag() {
			case "required":
				msg = "Field is required"
			case "max":
				msg = "Too many characters"
			}
		}
		return views.Render(c, http.StatusOK, components.DiscussionFormErrorField(msg))
	}
	return c.String(http.StatusOK, "")
}

func (app *application) validateDiscussionUrlHandler(c echo.Context) error {
	var input struct {
		Description string `query:"url" validate:"required,url"`
	}
	if err := c.Bind(&input); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}
	if err := c.Validate(&input); err != nil {
		msg := ""
		for _, ve := range err.(validator.ValidationErrors) {
			switch ve.Tag() {
			case "required":
				msg = "Field is required"
			case "url":
				msg = "Field must be url"
			}
		}
		return views.Render(c, http.StatusOK, components.DiscussionFormErrorField(msg))
	}
	return c.String(http.StatusOK, "")
}

func (app *application) genDiscussionPreview(c echo.Context) error {
	var input struct {
		Title       string `query:"title" validate:"required,max=130"`
		Description string `query:"description" validate:"required,max=4000"`
		Url         string `query:"url" validate:"required,url"`
	}

	if err := c.Bind(&input); err != nil {
		return c.String(http.StatusBadRequest, "bad request")
	}

	if err := c.Validate(&input); err != nil {
		errs := make([]string, 0, len(err.(validator.ValidationErrors)))
		for _, ve := range err.(validator.ValidationErrors) {
			switch ve.Tag() {
			case "required":
				errs = append(
					errs,
					fmt.Sprintf("Field '%s' can't be blank.", ve.Field()),
				)
			case "max":
				errs = append(
					errs,
					fmt.Sprintf(
						"Field '%s' maximum length is %s characters.",
						ve.Field(),
						ve.Param(),
					),
				)
			default:
				errs = append(errs, ve.Error())
			}
		}
		return c.String(http.StatusBadRequest, "Wrong params!")
	}

	resPath, err := app.services.ChromeDp.GenScreenshot(input.Url)
	if err != nil {
		return c.String(http.StatusBadRequest, "Couldn't generate preview")
	}

	return views.Render(
		c,
		http.StatusOK,
		components.DiscussionCard(
			components.DiscussionCardViewModel{
				CardTitle: input.Title,
				ImgSrc:    resPath,
			},
			true,
		),
	)
}

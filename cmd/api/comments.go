package main

import (
	"net/http"
	"strconv"

	"github.com/N0tR1CH/sad/internal/data"
	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/pages"
	"github.com/labstack/echo/v4"
)

func (app *application) createCommentHandler(c echo.Context) error {
	var input struct {
		DiscussionId string `param:"discussionId" validate:"required,number"`
		Content      string `form:"content" validate:"required"`
	}
	if err := c.Bind(&input); err != nil {
		app.logger.Error(
			"in app#createCommentHandler: values could not be binded",
			"input",
			input,
			"err",
			err.Error(),
		)
		return err
	}
	if err := c.Validate(&input); err != nil {
		app.logger.Error(
			"in app#createCommentHandler: values could not be validated",
			"input",
			input,
			"err",
			err.Error(),
		)
		return err
	}

	discussionId, err := strconv.Atoi(input.DiscussionId)
	if err != nil {
		app.logger.Error(
			"in app#createCommentHandler: id could not be converted to integer",
			"discussionId",
			input.DiscussionId,
			"err",
			err.Error(),
		)
	}
	comment := new(data.Comment)
	comment.UserId = app.sessionManager.GetInt(c.Request().Context(), "userID")
	comment.Content = input.Content
	comment.DiscussionId = discussionId
	if err := app.models.Comments.Insert(comment); err != nil {
		app.logger.Error("in app#createCommentHandler", "err", err.Error())
		return err
	}
	imgSrc, err := app.models.Users.AvatarSrcByID(comment.UserId)
	if err != nil {
		app.logger.Error("in app#createCommentHandler", "err", err.Error())
		return err
	}
	username, err := app.models.Users.GetUsername(comment.UserId)
	if err != nil {
		app.logger.Error("in app#createCommentHandler", "err", err.Error())
		return err
	}
	return views.Render(
		c,
		http.StatusOK,
		pages.Comment(
			pages.NewCommentViewModel(
				username,
				imgSrc,
				comment.Content,
				comment.CreatedAt,
			),
		),
	)
}

func (app *application) getCommentsHandler(c echo.Context) error {
	var input struct {
		DiscussionId string `param:"discussionId" validate:"required,number"`
	}

	if err := c.Bind(&input); err != nil {
		app.logger.Error(
			"in app#getCommentsHandler: values could not be binded",
			"input",
			input,
			"err",
			err.Error(),
		)
		return err
	}
	if err := c.Validate(&input); err != nil {
		app.logger.Error(
			"in app#getCommentsHandler: values could not be validated",
			"input",
			input,
			"err",
			err.Error(),
		)
		return err
	}

	discussionId, err := strconv.Atoi(input.DiscussionId)
	if err != nil {
		app.logger.Error(
			"in app#getCommentHandler: id could not be converted to integer",
			"discussionId",
			input.DiscussionId,
			"err",
			err.Error(),
		)
		return err
	}
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil {
		page = 1
	}

	comments, currCommCount, err := app.models.Comments.GetAllWithUser(discussionId, page)
	if err != nil {
		app.logger.Error(
			"in app#getCommentHandler: while getting all discussions",
			"discussionId",
			input.DiscussionId,
			"err",
			err.Error(),
			"comments",
			comments,
		)
		return err
	}
	cvms := make([]pages.CommentViewModel, len(comments))
	for i := range cvms {
		vm := pages.NewCommentViewModel(
			comments[i].U.Name,
			comments[i].U.AvatarSrc,
			comments[i].Content,
			comments[i].CreatedAt,
		)
		cvms[i] = vm
	}
	return views.Render(
		c,
		http.StatusOK,
		pages.Comments(cvms, discussionId, page+1, currCommCount),
	)
}

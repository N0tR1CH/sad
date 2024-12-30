package main

import (
	"errors"
	"fmt"
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
		return err
	}
	if err := c.Validate(&input); err != nil {
		return err
	}

	reply, err := strconv.ParseBool(c.FormValue("isReply"))
	if err != nil {
		app.logger.Error("in app#createCommentHandler", "err", err.Error())
	}
	app.logger.Info("in app#createCommentHandler", "reply", reply)

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

	if reply {
		parentId, err := strconv.Atoi(c.FormValue("parentId"))
		if err != nil {
			return err
		}
		comment.ParentId = parentId
	}
	if err := app.models.Comments.Insert(comment); err != nil {
		return err
	}
	imgSrc, err := app.models.Users.AvatarSrcByID(comment.UserId)
	if err != nil {
		return err
	}
	username, err := app.models.Users.GetUsername(comment.UserId)
	if err != nil {
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
				comment.NumUpvotes,
				comment.DiscussionId,
				comment.ID,
				comment.UserId,
			),
		),
	)
}

func (app *application) getCommentsHandler(c echo.Context) error {
	var input struct {
		DiscussionId string `param:"discussionId" validate:"required,number"`
	}

	if err := c.Bind(&input); err != nil {
		return fmt.Errorf("in app#getCommentHandler: %w", err)
	}
	if err := c.Validate(&input); err != nil {
		return fmt.Errorf("in app#getCommentHandler: %w", err)
	}
	if !c.Get("HTMX").(bool) {
		return c.Redirect(
			http.StatusTemporaryRedirect,
			fmt.Sprintf("/discussions/%s", input.DiscussionId),
		)
	}
	discussionId, err := strconv.Atoi(input.DiscussionId)
	if err != nil {
		return fmt.Errorf("in app#getCommentHandler: %w", err)
	}
	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil {
		page = 1
	}

	comments, currCommCount, err := app.models.Comments.GetAllWithUser(discussionId, page)
	if err != nil {
		return fmt.Errorf("in app#getCommentHandler: %w", err)
	}
	cvms := make([]pages.CommentViewModel, len(comments))
	for i := range cvms {
		vm := pages.NewCommentViewModel(
			comments[i].U.Name,
			comments[i].U.AvatarSrc,
			comments[i].Content,
			comments[i].CreatedAt,
			comments[i].NumUpvotes,
			comments[i].DiscussionId,
			comments[i].ID,
			comments[i].UserId,
		)
		cvms[i] = vm
	}
	return views.Render(
		c,
		http.StatusOK,
		pages.Comments(cvms, discussionId, page+1, currCommCount, 0),
	)
}

func (app *application) upvoteCommentHandler(c echo.Context) error {
	var input struct {
		CommentId string `param:"id" validate:"required,number"`
	}
	if err := c.Bind(&input); err != nil {
		return fmt.Errorf("in app#updateCommentHandler: %w", err)
	}
	if err := c.Validate(&input); err != nil {
		return fmt.Errorf("in app#updateCommentHandler: %w", err)
	}
	cId, err := strconv.Atoi(input.CommentId)
	if err != nil {
		return fmt.Errorf("in app#updateCommentHandler: %w", err)
	}
	var userId int
	if uId, ok := c.Get("userID").(int); !ok || uId == 0 {
		return errors.New("userID should be in the request context")
	} else {
		userId = uId
	}
	if err := app.models.Comments.Upvote(userId, cId); err != nil {
		if errors.Is(err, data.ErrUniquenessViolation) {
			return c.NoContent(http.StatusConflict)
		}
		return fmt.Errorf("in app#updateCommentHandler: %w", err)
	}
	return c.NoContent(http.StatusOK)
}

func (app *application) getCommentRepliesHandler(c echo.Context) error {
	var input struct {
		DiscussionId string `param:"discussionId" validate:"required,number"`
		CommentId    string `param:"id" validate:"required,number"`
		Page         string `query:"page" validate:"required,number"`
	}
	if err := c.Bind(&input); err != nil {
		return fmt.Errorf("in app#getCommentRepliesHandler: %w", err)
	}
	if err := c.Validate(&input); err != nil {
		return fmt.Errorf("in app#getCommentRepliesHandler: %w", err)
	}
	if !c.Get("HTMX").(bool) {
		return c.Redirect(
			http.StatusTemporaryRedirect,
			fmt.Sprintf("/discussions/%s", input.DiscussionId),
		)
	}
	discussionId, err := strconv.Atoi(input.DiscussionId)
	if err != nil {
		return fmt.Errorf("in app#getCommentRepliesHandler: %w", err)
	}
	page, err := strconv.Atoi(input.Page)
	if err != nil {
		page = 1
	}
	commentParentId, err := strconv.Atoi(input.CommentId)
	if err != nil {
		return fmt.Errorf("in app#getCommentRepliesHandler: %w", err)
	}

	comments, currCommCount, err := app.models.Comments.GetAllChildren(
		commentParentId,
		page,
	)
	if err != nil {
		return fmt.Errorf("in app#getCommentRepliesHandler: %w", err)
	}
	cvms := make([]pages.CommentViewModel, len(comments))
	for i := range cvms {
		vm := pages.NewCommentViewModel(
			comments[i].U.Name,
			comments[i].U.AvatarSrc,
			comments[i].Content,
			comments[i].CreatedAt,
			comments[i].NumUpvotes,
			comments[i].DiscussionId,
			comments[i].ID,
			comments[i].UserId,
		)
		cvms[i] = vm
	}
	return views.Render(
		c,
		http.StatusOK,
		pages.Comments(
			cvms,
			discussionId,
			page+1,
			currCommCount,
			commentParentId,
		),
	)
}

func (app *application) reportCommentHandler(c echo.Context) error {
	return nil
}

package main

import (
	"net/http"
	"strconv"

	"github.com/N0tR1CH/sad/views"
	"github.com/N0tR1CH/sad/views/pages"
	"github.com/labstack/echo/v4"
)

func (app *application) getReportsHandler(c echo.Context) error {
	lastSeenId, err := strconv.Atoi(c.QueryParam("lastSeenId"))
	if err != nil {
		app.logger.Error("in app#getReportsHandler", "err", err.Error())
	}
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		limit = 10
		app.logger.Error("in app#getReportsHandler", "err", err.Error())
	}
	if !c.Get("HTMX").(bool) {
		return views.Render(
			c,
			http.StatusOK,
			pages.ReportsPage(
				pages.ReportsPageProps{
					BodyProps: pages.ReportsPageBodyProps{
						LastSeenId: lastSeenId,
						Limit:      limit,
					},
				},
			),
		)
	}
	reports, err := app.models.Reports.GetAll(lastSeenId, limit)
	if err != nil {
		return err
	}
	rowsProps := make([]pages.ReportTableRowProps, len(reports))
	for i := range rowsProps {
		rowsProps[i].Id = reports[i].ID
		rowsProps[i].Reason = reports[i].Reason
		rowsProps[i].Username = reports[i].ReportedUser.Name
		rowsProps[i].UserId = reports[i].ReportedUser.ID
		rowsProps[i].DiscussionId = reports[i].DiscussionID
		rowsProps[i].CommentId = reports[i].CommentID
		rowsProps[i].ReportedAt = reports[i].CreatedAt
		rowsProps[i].UserAvatarSrc = reports[i].ReportedUser.AvatarSrc
	}
	return views.Render(
		c,
		http.StatusOK,
		pages.ReportTableRows(rowsProps, limit),
	)
}

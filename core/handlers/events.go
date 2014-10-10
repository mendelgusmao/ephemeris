package handlers

import (
	"database/sql"
	"ephemeris/core"
	"ephemeris/core/middleware"
	"ephemeris/core/models"
	"ephemeris/core/representers"
	"ephemeris/core/representers/transcoders"
	"ephemeris/core/routes"
	"fmt"
	"log/syslog"
	"net/http"
	"strconv"
	"time"

	"github.com/MendelGusmao/gorm"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
)

func init() {
	routes.Register(func(r martini.Router) {
		r.Get("/events", events)
		r.Post("/events", middleware.Authorize(),
			binding.Bind(representers.EventRequest{}), createEvent)

		r.Get("/events/:id", event)
		r.Put("/events/:id", middleware.Authorize(),
			binding.Bind(representers.EventRequest{}), updateEvent)
		r.Delete("/events/:id", deleteEvent)
	})
}

func createEvent(
	database *gorm.DB,
	eventRequest representers.EventRequest,
	logger core.Logger,
	renderer render.Render,
	user *models.User,
) {
	event := models.Event{User: *user}
	transcoders.EventFromRequest(&eventRequest, &event)

	if query := database.Save(&event); query.Error != nil {
		logger.Log(syslog.LOG_INFO, query.Error)
		renderer.Status(http.StatusInternalServerError)
		return
	}

	renderer.Header().Add("Location", fmt.Sprintf("/events/%d", event.Id))
	renderer.Status(http.StatusCreated)
}

func events(
	database *gorm.DB,
	logger core.Logger,
	renderer render.Render,
) {
	events := make([]models.Event, 0)
	lastModified := time.Time{}

	if query := database.Find(&events); query.Error != nil {
		// TODO gorm doesn't return gorm.RecordNotFound when using testdb as driver
		if query.Error == gorm.RecordNotFound || query.Error == sql.ErrNoRows {
			renderer.Status(http.StatusNoContent)
			return
		}

		logger.Log(syslog.LOG_INFO, query.Error)
		renderer.Status(http.StatusInternalServerError)
		return
	}

	representedEvents := make([]representers.EventResponse, len(events))

	for index, event := range events {
		if event.UpdatedAt.Unix() > lastModified.Unix() {
			lastModified = event.UpdatedAt
		}

		if query := database.Model(event).Related(&event.User); query.Error != nil {
			logger.Log(syslog.LOG_INFO, query.Error)
			renderer.Status(http.StatusInternalServerError)
			return
		}

		representedEvents[index] = transcoders.EventToResponse(&event)
	}

	renderer.Header().Add("Last-Modified", lastModified.UTC().Format(time.RFC1123))
	renderer.JSON(http.StatusOK, representedEvents)
}

func event(
	database *gorm.DB,
	logger core.Logger,
	params martini.Params,
	renderer render.Render,
) {
	id, _ := strconv.Atoi(params["id"])
	event := models.Event{Id: id}

	if query := database.Find(&event); query.Error != nil {
		// TODO gorm doesn't return gorm.RecordNotFound when using testdb as driver
		if query.Error == gorm.RecordNotFound || query.Error == sql.ErrNoRows {
			renderer.Status(http.StatusNotFound)
			return
		}

		logger.Log(syslog.LOG_INFO, query.Error)
		renderer.Status(http.StatusInternalServerError)
		return
	}

	if query := database.Model(event).Related(&event.User); query.Error != nil {
		logger.Log(syslog.LOG_INFO, query.Error)
		renderer.Status(http.StatusInternalServerError)
		return
	}

	renderer.Header().Add("Last-Modified", event.CreatedAt.UTC().Format(time.RFC1123))
	renderer.JSON(http.StatusOK, transcoders.EventToResponse(&event))
}

func updateEvent(
	database *gorm.DB,
	eventRequest representers.EventRequest,
	logger core.Logger,
	params martini.Params,
	renderer render.Render,
) {
	id, _ := strconv.Atoi(params["id"])
	event := models.Event{Id: id}

	if query := database.Find(&event); query.Error != nil {
		if query.Error == gorm.RecordNotFound || query.Error == sql.ErrNoRows {
			renderer.Status(http.StatusNotFound)
			return
		}

		logger.Log(syslog.LOG_INFO, query.Error)
		renderer.Status(http.StatusInternalServerError)
		return
	}

	transcoders.EventFromRequest(&eventRequest, &event)

	if query := database.Save(event); query.Error != nil {
		logger.Log(syslog.LOG_INFO, query.Error)
		renderer.Status(http.StatusInternalServerError)
		return
	}

	renderer.Header().Add("Location", fmt.Sprintf("/events/%d", event.Id))
	renderer.Status(http.StatusOK)
}

func deleteEvent(
	database *gorm.DB,
	logger core.Logger,
	params martini.Params,
	renderer render.Render,
) {
	id, _ := strconv.Atoi(params["id"])
	event := models.Event{Id: id}

	if query := database.Find(&event); query.Error != nil {
		if query.Error == gorm.RecordNotFound || query.Error == sql.ErrNoRows {
			renderer.Status(http.StatusNotFound)
			return
		}

		logger.Log(syslog.LOG_INFO, query.Error)
		renderer.Status(http.StatusInternalServerError)
		return
	}

	if query := database.Delete(&event); query.Error != nil {
		logger.Log(syslog.LOG_INFO, query.Error)
		renderer.Status(http.StatusInternalServerError)
		return
	}

	renderer.Status(http.StatusNoContent)
}

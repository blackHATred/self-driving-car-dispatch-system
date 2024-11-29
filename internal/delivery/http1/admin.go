package http1

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"self-driving-car-dispatch-system/internal/entity"
	"self-driving-car-dispatch-system/internal/usecase"
	"strconv"
)

type AdminDelivery struct {
	adminUsecase usecase.AdminUsecase
	logger       *logrus.Logger
}

func NewAdminDelivery(logger *logrus.Logger, adminUsecase usecase.AdminUsecase) *AdminDelivery {
	return &AdminDelivery{
		adminUsecase: adminUsecase,
		logger:       logger,
	}
}

func (a AdminDelivery) Configure(handler *gin.RouterGroup) {
	// Маршруты для работы с диспетчерами
	handler.GET("/dispatcher/:id", a.GetDispatcher)
	handler.POST("/dispatcher", a.AddDispatcher)
	handler.PUT("/dispatcher", a.EditDispatcherGrants)
	handler.DELETE("/dispatcher/:id", a.DeleteDispatcher)
	// Маршруты для работы с ТС
	handler.GET("/vehicle/:id", a.GetVehicle)
	handler.POST("/vehicle", a.AddVehicle)
	handler.DELETE("/vehicle/:id", a.DeleteVehicle)
}

// Dispatcher

func (a AdminDelivery) GetDispatcher(c *gin.Context) {
	var id int
	var err error
	if id, err = strconv.Atoi(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	secret := c.GetHeader("X-Secret")
	dispatcher, err := a.adminUsecase.GetDispatcher(secret, id)
	switch {
	case errors.Is(err, usecase.ErrAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, usecase.ErrDispatcherNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "dispatcher not found"})
	case err == nil:
		c.JSON(http.StatusOK, dispatcher)
	default:
		a.logger.Errorf("failed to get dispatcher: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (a AdminDelivery) AddDispatcher(c *gin.Context) {
	dispatcherRequest := entity.AddDispatcherRequest{}
	if err := c.BindJSON(&dispatcherRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	secret := c.GetHeader("X-Secret")
	dispatcher, err := a.adminUsecase.AddDispatcher(secret, &dispatcherRequest)
	switch {
	case errors.Is(err, usecase.ErrAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, usecase.ErrDispatcherAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "dispatcher already exists"})
	case errors.Is(err, usecase.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	case err == nil:
		c.JSON(http.StatusCreated, gin.H{"id": dispatcher.ID})
	default:
		a.logger.Errorf("failed to add dispatcher: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (a AdminDelivery) EditDispatcherGrants(c *gin.Context) {
	dispatcher := entity.EditDispatcherRequest{}
	if err := c.BindJSON(&dispatcher); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	secret := c.GetHeader("X-Secret")
	err := a.adminUsecase.EditDispatcher(secret, &dispatcher)
	switch {
	case errors.Is(err, usecase.ErrAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, usecase.ErrDispatcherNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "dispatcher not found"})
	case errors.Is(err, usecase.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	case err == nil:
		c.JSON(http.StatusNoContent, nil)
	default:
		a.logger.Errorf("failed to edit dispatcher: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (a AdminDelivery) DeleteDispatcher(c *gin.Context) {
	var id int
	var err error
	if id, err = strconv.Atoi(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}
	secret := c.GetHeader("X-Secret")
	err = a.adminUsecase.DeleteDispatcher(secret, id)
	switch {
	case errors.Is(err, usecase.ErrAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, usecase.ErrDispatcherNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "dispatcher not found"})
	case err == nil:
		c.JSON(http.StatusOK, nil)
	default:
		a.logger.Errorf("failed to delete dispatcher: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

// Vehicle

func (a AdminDelivery) GetVehicle(c *gin.Context) {
	var id int
	var err error
	if id, err = strconv.Atoi(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	secret := c.GetHeader("X-Secret")
	vehicle, err := a.adminUsecase.GetVehicle(secret, id)
	switch {
	case errors.Is(err, usecase.ErrAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, usecase.ErrVehicleNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
	case err == nil:
		c.JSON(http.StatusOK, vehicle)
	default:
		a.logger.Errorf("failed to get vehicle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (a AdminDelivery) AddVehicle(c *gin.Context) {
	vehicleRequest := entity.AddVehicleRequest{}
	if err := c.BindJSON(&vehicleRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	secret := c.GetHeader("X-Secret")
	vehicle, err := a.adminUsecase.AddVehicle(secret, &vehicleRequest)
	switch {
	case errors.Is(err, usecase.ErrAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, usecase.ErrVehicleAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "vehicle already exists"})
	case errors.Is(err, usecase.ErrBadRequest):
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	case err == nil:
		c.JSON(http.StatusCreated, gin.H{"id": vehicle.ID})
	default:
		a.logger.Errorf("failed to add vehicle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (a AdminDelivery) DeleteVehicle(c *gin.Context) {
	var id int
	var err error
	if id, err = strconv.Atoi(c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}
	secret := c.GetHeader("X-Secret")
	err = a.adminUsecase.DeleteVehicle(secret, id)
	switch {
	case errors.Is(err, usecase.ErrAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
	case errors.Is(err, usecase.ErrVehicleNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
	case err == nil:
		c.JSON(http.StatusNoContent, nil)
	default:
		a.logger.Errorf("failed to delete vehicle: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/juju/errors"
	"github.com/fleetdm/fleet/infrastructure/demo/JITProvisioner/lambda/models"
)

func NewDemo(_ *gin.Context, params *models.NewDemoParams) (models.NewDemo, error) {
	if params.Name != "bob" {
		return models.NewDemo{}, errors.NotFoundf("Pet %s", params.Name)
	}
	return models.NewDemo{
		Name:  "bob",
		Price: 100,
		Breed: "bengal",
	}, nil
}

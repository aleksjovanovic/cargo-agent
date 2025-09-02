package handlers

import "github.com/aleksjovanovic/cargo-agent/internal/services"

// Handler je tanak sloj – injektuje servise i poziva ih.
type Handler struct {
	Users             *services.UserService
	Countries         *services.CountryService
	CargoOffer        *services.CargoOfferService
	TruckAvailability *services.TruckAvailabilityService
}

// NewHandlers – prima servise i sklapa jedan objekat za rute.
func NewHandlers(users *services.UserService, countries *services.CountryService, cargoOffer *services.CargoOfferService, truckAvailability *services.TruckAvailabilityService) *Handler {
	return &Handler{
		Users:             users,
		Countries:         countries,
		CargoOffer:        cargoOffer,
		TruckAvailability: truckAvailability,
	}
}

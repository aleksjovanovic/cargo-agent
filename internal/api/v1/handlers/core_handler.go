// internal/api/v1/handlers/core_handler.go
package handlers

import (
	"github.com/aleksjovanovic/cargo-agent/internal/logger"
	"github.com/aleksjovanovic/cargo-agent/internal/services"
)

// Handler aggregates all domain services that HTTP handlers depend on.
// It is intentionally "thin": request parsing/validation happens in HTTP
// handler funcs, while business logic lives in the service layer.
//
// Note: This type is safe to reuse across requests; it is effectively
// immutable after construction (services are shared singletons).
type Handler struct {
	// Users encapsulates authentication, profile, and account management.
	Users *services.UserService

	// Countries provides country/city lookups with Redis-backed caching.
	Countries *services.CountryService

	// CargoOffer manages CRUD-ish operations for cargo offers.
	CargoOffer *services.CargoOfferService

	// TruckAvailability manages postings for available trucks/routes.
	TruckAvailability *services.TruckAvailabilityService
}

// NewHandlers wires a Handler with all required services.
// We keep the signature concrete (no interfaces here) to avoid breaking
// existing callers. For tests, you can create fake services and pass them in.
//
// This constructor logs a warning when any dependency is nil to help catch
// misconfiguration early, but it does not panic to remain non-breaking.
func NewHandlers(
	users *services.UserService,
	countries *services.CountryService,
	cargoOffer *services.CargoOfferService,
	truckAvailability *services.TruckAvailabilityService,
) *Handler {
	// Soft sanity checks (non-breaking): warn if any dependency is missing.
	if users == nil {
		logger.Warn("handlers.NewHandlers: Users service is nil")
	}
	if countries == nil {
		logger.Warn("handlers.NewHandlers: Countries service is nil")
	}
	if cargoOffer == nil {
		logger.Warn("handlers.NewHandlers: CargoOffer service is nil")
	}
	if truckAvailability == nil {
		logger.Warn("handlers.NewHandlers: TruckAvailability service is nil")
	}

	return &Handler{
		Users:             users,
		Countries:         countries,
		CargoOffer:        cargoOffer,
		TruckAvailability: truckAvailability,
	}
}

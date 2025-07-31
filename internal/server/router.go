package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-http-utils/headers"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/rs/cors"

	"github.com/platform-mesh/extension-manager-operator/internal/config"
	"github.com/platform-mesh/extension-manager-operator/pkg/validation"
)

func CreateRouter(
	appConfig config.ServerConfig,
	log *logger.Logger,
	validator validation.ExtensionConfiguration,
) *chi.Mux {
	router := chi.NewRouter()

	if appConfig.IsLocal {
		rl := requestLogger{
			log: log,
		}

		router.Use(cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowCredentials: true,
			AllowedHeaders:   []string{headers.Accept, headers.Authorization, headers.ContentType, headers.XCSRFToken},
			Debug:            false,
			AllowedMethods:   []string{http.MethodPost, http.MethodGet},
		}).Handler)
		router.Use(rl.Handler)
	}

	vh := NewHttpValidateHandler(log, validator)

	router.MethodFunc(http.MethodPost, "/validate", vh.HandlerValidate)
	router.MethodFunc(http.MethodGet, "/healthz", vh.HandlerHealthz)
	router.MethodFunc(http.MethodGet, "/readyz", vh.HandlerHealthz)

	return router
}

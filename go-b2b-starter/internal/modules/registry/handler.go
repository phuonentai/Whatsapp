package registry

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	registryServices "github.com/moasq/go-b2b-starter/internal/modules/registry/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/registry/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	"github.com/moasq/go-b2b-starter/pkg/response"
)

type Handler struct {
	moduleService registryServices.ModuleService
}

func NewHandler(moduleService registryServices.ModuleService) *Handler {
	return &Handler{moduleService: moduleService}
}

// GetCatalog returns the tenant-visible module catalog.
func (h *Handler) GetCatalog(c *gin.Context) {
	modules, err := h.moduleService.ListCatalog(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar módulos", err)
		return
	}
	response.Success(c, http.StatusOK, mapModules(modules))
}

// GetOrgModules returns the requesting org's enabled modules with configs.
func (h *Handler) GetOrgModules(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	modules, orgMods, err := h.moduleService.ListOrgModules(c.Request.Context(), ctx.OrganizationID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar módulos de la organización", err)
		return
	}
	result := make([]map[string]any, 0, len(modules))
	for i, m := range modules {
		config := map[string]any{}
		if i < len(orgMods) && orgMods[i].Config != nil {
			config = orgMods[i].Config
		}
		result = append(result, map[string]any{
			"key":        m.Key,
			"name":       m.Name,
			"description": m.Description,
			"features":   m.GrantedFeatures,
			"config":     config,
		})
	}
	response.Success(c, http.StatusOK, result)
}

// GetMyModules returns module state from the entitlement (single read).
func (h *Handler) GetMyModules(c *gin.Context) {
	entitlement := features.GetEntitlement(c)
	if entitlement == nil {
		response.Error(c, http.StatusOK, "", nil)
		return
	}
	response.Success(c, http.StatusOK, entitlement.Modules)
}

// SaveModuleConfig validates and persists the org's module config.
func (h *Handler) SaveModuleConfig(c *gin.Context) {
	ctx := auth.GetRequestContext(c)
	moduleKey := c.Param("key")
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Solicitud inválida", err)
		return
	}
	saved, err := h.moduleService.SaveOrgModuleConfig(c.Request.Context(), ctx.OrganizationID, moduleKey, req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrModuleNotFound):
			response.Error(c, http.StatusNotFound, domain.ErrModuleNotFound.Error(), err)
		case errors.Is(err, domain.ErrModuleDisabled):
			response.Error(c, http.StatusForbidden, domain.ErrModuleDisabled.Error(), err)
		case errors.Is(err, domain.ErrInvalidModuleConfig):
			response.Error(c, http.StatusBadRequest, domain.ErrInvalidModuleConfig.Error(), err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al guardar configuración", err)
		}
		return
	}
	response.Success(c, http.StatusOK, map[string]any{
		"module": saved.ModuleKey,
		"config": saved.Config,
	})
}

func mapModules(modules []*domain.Module) []map[string]any {
	result := make([]map[string]any, 0, len(modules))
	for _, m := range modules {
		result = append(result, map[string]any{
			"key":         m.Key,
			"name":        m.Name,
			"description": m.Description,
			"features":    m.GrantedFeatures,
			"requires":    m.Requires,
		})
	}
	return result
}

package playbooks

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	playbooksServices "github.com/moasq/go-b2b-starter/internal/modules/playbooks/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/playbooks/domain"
	"github.com/moasq/go-b2b-starter/pkg/response"
)

func orgIDFromContext(c *gin.Context) int32 {
	rc := auth.GetRequestContext(c)
	if rc == nil {
		return 0
	}
	return rc.OrganizationID
}

type Handler struct {
	playbookService playbooksServices.PlaybookService
}

func NewHandler(playbookService playbooksServices.PlaybookService) *Handler {
	return &Handler{playbookService: playbookService}
}

// GetCatalog returns the tenant-visible playbook catalog with applied state
// and guiones for applied playbooks.
func (h *Handler) GetCatalog(c *gin.Context) {
	ctx := c.Request.Context()
	orgID := orgIDFromContext(c)
	entries, err := h.playbookService.ListCatalog(ctx, orgID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Error al listar los playbooks", err)
		return
	}
	result := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		item := map[string]any{
			"key":              entry.Playbook.Key,
			"name":             entry.Playbook.Name,
			"vertical":         entry.Playbook.Vertical,
			"description":      entry.Playbook.Description,
			"requires_modules": entry.Playbook.RequiresModules,
			"applied":          entry.Applied != nil,
		}
		if entry.Applied != nil {
			item["applied_at"] = entry.Applied.AppliedAt
			guiones := make([]map[string]any, 0, len(entry.Guiones))
			for _, guion := range entry.Guiones {
				item := map[string]any{
					"id":      guion.ID,
					"titulo":  guion.Titulo,
					"mensaje": guion.Mensaje,
				}
				if len(guion.Pasos) > 0 {
					pasos := make([]map[string]any, 0, len(guion.Pasos))
					for _, paso := range guion.Pasos {
						pasos = append(pasos, map[string]any{
							"id":      paso.ID,
							"titulo":  paso.Titulo,
							"mensaje": paso.Mensaje,
						})
					}
					item["pasos"] = pasos
				}
				guiones = append(guiones, item)
			}
			item["guiones"] = guiones
		}
		result = append(result, item)
	}
	response.Success(c, http.StatusOK, result)
}

// Apply seeds the playbook for the requesting org (one-way, idempotent).
func (h *Handler) Apply(c *gin.Context) {
	key := c.Param("key")
	state, err := h.playbookService.Apply(c.Request.Context(), orgIDFromContext(c), key)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPlaybookNotFound):
			response.Error(c, http.StatusNotFound, "Playbook no encontrado", err)
		case errors.Is(err, domain.ErrInvalidPlaybookPayload),
			errors.Is(err, domain.ErrPlaybookRequiresModules),
			errors.Is(err, domain.ErrInvalidModuleConfigPreset):
			response.Error(c, http.StatusBadRequest, "No se pudo aplicar el playbook", err)
		default:
			response.Error(c, http.StatusInternalServerError, "Error al aplicar el playbook", err)
		}
		return
	}
	response.Success(c, http.StatusOK, map[string]any{
		"key":        state.PlaybookKey,
		"applied_at": state.AppliedAt,
	})
}

// Reset removes the playbook's seeded data only.
func (h *Handler) Reset(c *gin.Context) {
	key := c.Param("key")
	if err := h.playbookService.Reset(c.Request.Context(), orgIDFromContext(c), key); err != nil {
		if errors.Is(err, domain.ErrPlaybookNotFound) {
			response.Error(c, http.StatusNotFound, "Playbook no encontrado", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Error al reiniciar el playbook", err)
		return
	}
	response.Success(c, http.StatusOK, map[string]any{"key": key, "reset": true})
}

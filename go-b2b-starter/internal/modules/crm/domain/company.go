package domain

import (
	"time"
)

var validTipoEmpresa = map[string]bool{
	"microempresa": true,
	"pequena":      true,
	"mediana":      true,
	"grande":       true,
}

var validTipoDocumento = map[string]bool{
	"CC":  true,
	"NIT": true,
	"CE":  true,
	"TI":  true,
	"PP":  true,
}

type Company struct {
	ID             int32                  `json:"id"`
	OrganizationID int32                  `json:"organization_id"`
	Name           string                 `json:"name"`
	Nit            string                 `json:"nit,omitempty"`
	TipoEmpresa    string                 `json:"tipo_empresa,omitempty"`
	Sector         string                 `json:"sector,omitempty"`
	Ciudad         string                 `json:"ciudad,omitempty"`
	Departamento   string                 `json:"departamento,omitempty"`
	Website        string                 `json:"website,omitempty"`
	Phone          string                 `json:"phone,omitempty"`
	Address        string                 `json:"address,omitempty"`
	Notes          string                 `json:"notes,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	OwnerAccountID *int32                 `json:"owner_account_id,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

func (c *Company) Validate() error {
	if c.Name == "" {
		return ErrCompanyNameRequired
	}
	if c.TipoEmpresa != "" && !validTipoEmpresa[c.TipoEmpresa] {
		return ErrTipoEmpresaInvalido
	}
	return nil
}

type CompanyWithCounts struct {
	Company
	TotalContactos int32 `json:"total_contactos"`
	TotalNegocios  int32 `json:"total_negocios"`
}

package domain

import "errors"

var (
	ErrContactNotFound             = errors.New("contacto no encontrado")
	ErrContactOrganizationRequired = errors.New("el ID de la organización es requerido")
	ErrContactPhoneRequired        = errors.New("el número de teléfono es requerido")
	ErrContactDuplicateEmail       = errors.New("Ya existe un contacto con este correo electrónico.")
	ErrConversationNotFound        = errors.New("conversación no encontrada")
	ErrMessageNotFound             = errors.New("mensaje no encontrado")
	ErrInvalidPhoneNumber          = errors.New("número de teléfono inválido")
	ErrInvalidDirection            = errors.New("dirección de mensaje inválida")
	ErrInvalidMessageType          = errors.New("tipo de mensaje inválido")

	ErrCompanyNotFound        = errors.New("empresa no encontrada")
	ErrCompanyNameRequired    = errors.New("el nombre de la empresa es requerido")
	ErrCompanyDuplicateName   = errors.New("Ya existe una empresa con este nombre.")
	ErrTipoEmpresaInvalido    = errors.New("tipo de empresa inválido. Valores: microempresa, pequena, mediana, grande")
	ErrTipoDocumentoInvalido  = errors.New("tipo de documento inválido. Valores: CC, NIT, CE, TI, PP")

	ErrDealNotFound            = errors.New("negocio no encontrado")
	ErrDealNameRequired        = errors.New("el nombre del negocio es requerido")
	ErrInvalidStatusTransition = errors.New("transición de estado inválida")
	ErrStageNotInPipeline      = errors.New("la etapa no pertenece al pipeline del negocio")

	ErrPipelineNotFound = errors.New("pipeline no encontrado")
	ErrStageNotFound    = errors.New("etapa no encontrada")

	ErrActivityNotFound = errors.New("actividad no encontrada")

	ErrTagNotFound      = errors.New("etiqueta no encontrada")
	ErrTagDuplicateName = errors.New("Ya existe una etiqueta con este nombre.")

	ErrFuncionalidadNoDisponible = errors.New("funcionalidad no disponible en tu plan actual")
)

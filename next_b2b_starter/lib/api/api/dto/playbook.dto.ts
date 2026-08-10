export interface PlaybookGuionPasoDto {
  id: string;
  titulo: string;
  mensaje: string;
}

export interface PlaybookGuionDto {
  id: string;
  titulo: string;
  mensaje?: string;
  pasos?: PlaybookGuionPasoDto[];
}

export interface PlaybookDto {
  key: string;
  name: string;
  vertical: string;
  description?: string;
  requires_modules: string[];
  applied: boolean;
  applied_at?: string;
  guiones?: PlaybookGuionDto[];
}

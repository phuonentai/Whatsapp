export interface ParsedApiError {
  status?: number;
  message?: string;
}

const API_ERROR_PREFIX = /^API Error (\d{3}): (.*)$/;

export function parseApiError(error: unknown): ParsedApiError {
  if (!(error instanceof Error)) return {};
  const match = API_ERROR_PREFIX.exec(error.message);
  if (!match) return { message: error.message };
  return { status: Number(match[1]), message: match[2] };
}

export function toSpanishMutationError(error: unknown): string {
  const { status, message } = parseApiError(error);

  if (status === 409) {
    return message && message.length > 0 ? message : "Ya existe un registro con los mismos datos.";
  }
  if (status && status >= 400 && status < 500) {
    return "Solicitud inválida";
  }
  return "Error de conexión";
}

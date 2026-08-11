import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { siigoRepository } from "@/lib/api/api/repositories/siigo-repository";
import { queryKeys } from "@/lib/hooks/queries/query-keys";
import type {
  SiigoConnectInput,
} from "@/lib/models/siigo-connection.model";

function useInvalidateSiigo() {
  const queryClient = useQueryClient();
  return () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.siigo.all });
  };
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Operación de Siigo fallida";
}

export function useSiigoConnect() {
  const invalidate = useInvalidateSiigo();
  return useMutation({
    mutationFn: (input: SiigoConnectInput) => siigoRepository.connect(input),
    onSuccess: () => {
      invalidate();
      toast.success("Siigo conectado correctamente");
    },
    onError: (error: Error) => {
      toast.error(errorMessage(error));
    },
  });
}

export function useRequestAssistedSetup() {
  const invalidate = useInvalidateSiigo();
  return useMutation({
    mutationFn: () => siigoRepository.requestAssisted(),
    onSuccess: () => {
      invalidate();
      toast.success("Solicitud enviada — tu equipo configurará tu facturación");
    },
    onError: (error: Error) => {
      toast.error(errorMessage(error));
    },
  });
}

export function useConfirmNumeration() {
  const invalidate = useInvalidateSiigo();
  return useMutation({
    mutationFn: () => siigoRepository.confirmNumeration(),
    onSuccess: () => {
      invalidate();
      toast.success("Numeración confirmada");
    },
    onError: (error: Error) => {
      toast.error(errorMessage(error));
    },
  });
}

export function useImportConfirm() {
  const invalidate = useInvalidateSiigo();
  return useMutation({
    mutationFn: () => siigoRepository.importConfirm(),
    onSuccess: () => {
      invalidate();
      toast.success("Clientes importados correctamente");
    },
    onError: (error: Error) => {
      toast.error(errorMessage(error));
    },
  });
}

export function useSiigoSync() {
  const invalidate = useInvalidateSiigo();
  return useMutation({
    mutationFn: () => siigoRepository.sync(),
    onSuccess: () => {
      invalidate();
      toast.success("Sincronización completada");
    },
    onError: (error: Error) => {
      toast.error(errorMessage(error));
    },
  });
}

export function useTestInvoice() {
  const invalidate = useInvalidateSiigo();
  return useMutation({
    mutationFn: () => siigoRepository.testInvoice(),
    onSuccess: () => {
      invalidate();
      toast.success("Factura de prueba creada en sandbox");
    },
    onError: (error: Error) => {
      toast.error(errorMessage(error));
    },
  });
}

export function usePauseInvoicing() {
  const invalidate = useInvalidateSiigo();
  return useMutation({
    mutationFn: () => siigoRepository.pause(),
    onSuccess: () => {
      invalidate();
      toast.success("Facturación pausada");
    },
    onError: (error: Error) => {
      toast.error(errorMessage(error));
    },
  });
}

export function useResumeInvoicing() {
  const invalidate = useInvalidateSiigo();
  return useMutation({
    mutationFn: () => siigoRepository.resume(),
    onSuccess: () => {
      invalidate();
      toast.success("Facturación reanudada");
    },
    onError: (error: Error) => {
      toast.error(errorMessage(error));
    },
  });
}

export function useActivateInvoicing() {
  const invalidate = useInvalidateSiigo();
  return useMutation({
    mutationFn: () => siigoRepository.activate(),
    onSuccess: () => {
      invalidate();
      toast.success("Facturación activada");
    },
    onError: (error: Error) => {
      toast.error(errorMessage(error));
    },
  });
}

export function useAdminProvision() {
  const invalidate = useInvalidateSiigo();
  return useMutation({
    mutationFn: (input: {
      organization_id: number;
      client_id: string;
      client_secret: string;
      nit: string;
    }) => siigoRepository.adminProvision(input),
    onSuccess: () => {
      invalidate();
      toast.success("Credenciales provisionadas");
    },
    onError: (error: Error) => {
      toast.error(errorMessage(error));
    },
  });
}

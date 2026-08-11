import type { UseQueryResult } from "@tanstack/react-query";

interface ErrorRecovery {
  isLoading: boolean;
  isError: boolean;
  retry: () => void;
  isRetrying: boolean;
}

export function useErrorRecovery<T>(query: UseQueryResult<T>): ErrorRecovery {
  return {
    isLoading: query.isLoading,
    isError: query.isError,
    retry: () => {
      void query.refetch();
    },
    isRetrying: query.isRefetching,
  };
}

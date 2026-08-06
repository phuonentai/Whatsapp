export enum LoadableState {
  NotInitiated = "notInitiated",
  Loading = "loading",
  Success = "success",
  Failure = "failure",
  Empty = "empty"
}

export type Loadable<T> = 
  | { state: LoadableState.NotInitiated; existing?: undefined; error?: undefined }
  | { state: LoadableState.Loading; existing?: T; error?: undefined }
  | { state: LoadableState.Success; data: T; existing?: undefined; error?: undefined }
  | { state: LoadableState.Failure; error: Error; existing?: undefined }
  | { state: LoadableState.Empty; existing?: undefined; error?: undefined };

export const LoadableHelpers = {
  notInitiated: <T>(): Loadable<T> => ({ state: LoadableState.NotInitiated }),
  
  loading: <T>(existing?: T): Loadable<T> => ({ 
    state: LoadableState.Loading, 
    existing 
  }),
  
  success: <T>(data: T): Loadable<T> => ({ 
    state: LoadableState.Success, 
    data 
  }),
  
  failure: <T>(error: Error): Loadable<T> => ({ 
    state: LoadableState.Failure, 
    error 
  }),
  
  empty: <T>(): Loadable<T> => ({ state: LoadableState.Empty }),
  
  getValue: <T>(loadable: Loadable<T>): T | undefined => {
    switch (loadable.state) {
      case LoadableState.Success:
        return loadable.data;
      case LoadableState.Loading:
        return loadable.existing;
      default:
        return undefined;
    }
  },
  
  isLoading: <T>(loadable: Loadable<T>): boolean => 
    loadable.state === LoadableState.Loading,
  
  isSuccess: <T>(loadable: Loadable<T>): boolean =>
    loadable.state === LoadableState.Success,
  
  isFailure: <T>(loadable: Loadable<T>): boolean =>
    loadable.state === LoadableState.Failure,
  
  isEmpty: <T>(loadable: Loadable<T>): boolean =>
    loadable.state === LoadableState.Empty,
  
  isNotInitiated: <T>(loadable: Loadable<T>): boolean =>
    loadable.state === LoadableState.NotInitiated,
  
  getError: <T>(loadable: Loadable<T>): Error | undefined =>
    loadable.state === LoadableState.Failure ? loadable.error : undefined
};
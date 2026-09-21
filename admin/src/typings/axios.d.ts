import 'axios';

declare module 'axios' {
  interface AxiosRequestConfig {
    /** Suppress global error UI and authentication side effects for optional requests. */
    silent?: boolean;
  }
}

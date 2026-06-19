import { useState, useEffect, useCallback } from 'react';
import { AxiosResponse } from 'axios';

interface UseApiResult<T> {
  data: T | null;
  error: string | null;
  loading: boolean;
  refetch: () => void;
}

export function useApi<T>(
  fetcher: () => Promise<AxiosResponse<T>>,
  deps: unknown[] = []
): UseApiResult<T> {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const execute = useCallback(() => {
    setLoading(true);
    setError(null);
    fetcher()
      .then((res) => setData(res.data))
      .catch((err) => setError(err.response?.data?.message || err.message))
      .finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  useEffect(() => {
    execute();
  }, [execute]);

  return { data, error, loading, refetch: execute };
}

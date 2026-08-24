import { useCallback, useEffect, useRef, useState } from "react";

import { getClient } from "src/core/StashService";
import { SearchDocument, SearchQuery } from "src/core/generated-graphql";

const DEBOUNCE_MS = 250;
// the loading indicator is only shown if the query takes longer than this
const LOADING_INDICATOR_DELAY_MS = 150;

export interface ISearchState {
  results: SearchQuery["search"] | undefined;
  // true while a query is in flight
  loading: boolean;
  // true while a query is in flight and has taken longer than
  // LOADING_INDICATOR_DELAY_MS - used to show skeletons
  showLoading: boolean;
  error: string | undefined;
  retry: () => void;
}

export function useSearch(term: string, active: boolean): ISearchState {
  const [results, setResults] = useState<SearchQuery["search"]>();
  const [loading, setLoading] = useState(false);
  const [showLoading, setShowLoading] = useState(false);
  const [error, setError] = useState<string>();

  const seq = useRef(0);
  const debounceTimer = useRef<number | undefined>(undefined);
  const loadingTimer = useRef<number | undefined>(undefined);
  const lastTerm = useRef("");

  const run = useCallback(async (searchTerm: string) => {
    const id = ++seq.current;
    lastTerm.current = searchTerm;

    setLoading(true);
    setShowLoading(false);

    window.clearTimeout(loadingTimer.current);
    loadingTimer.current = window.setTimeout(() => {
      if (seq.current === id) {
        setShowLoading(true);
      }
    }, LOADING_INDICATOR_DELAY_MS);

    try {
      const res = await getClient().query<SearchQuery>({
        query: SearchDocument,
        variables: { input: { term: searchTerm } },
        fetchPolicy: "no-cache",
      });

      if (id !== seq.current) return;

      setResults(res.data.search);
      setError(undefined);
    } catch (e) {
      if (id !== seq.current) return;
      setResults(undefined);
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      window.clearTimeout(loadingTimer.current);
      if (id === seq.current) {
        setLoading(false);
        setShowLoading(false);
      }
    }
  }, []);

  const retry = useCallback(() => {
    if (lastTerm.current) {
      run(lastTerm.current);
    }
  }, [run]);

  useEffect(() => {
    window.clearTimeout(debounceTimer.current);
    window.clearTimeout(loadingTimer.current);

    if (!active || !term.trim()) {
      // invalidate any in-flight request and clear state
      seq.current++;
      setLoading(false);
      setShowLoading(false);
      setError(undefined);
      if (!term.trim()) {
        setResults(undefined);
      }
      return;
    }

    debounceTimer.current = window.setTimeout(() => {
      run(term.trim());
    }, DEBOUNCE_MS);

    return () => {
      window.clearTimeout(debounceTimer.current);
    };
  }, [term, active, run]);

  useEffect(() => {
    return () => {
      window.clearTimeout(debounceTimer.current);
      window.clearTimeout(loadingTimer.current);
    };
  }, []);

  return { results, loading, showLoading, error, retry };
}

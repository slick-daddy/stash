import { useCallback, useEffect, useRef, useState } from "react";

import { getClient } from "src/core/StashService";
import { SearchDocument, SearchQuery } from "src/core/generated-graphql";

const DEBOUNCE_MS = 250;

export interface ISearchState {
  // results are undefined while waiting for the query for the current
  // term - either within the debounce window or while in flight
  results: SearchQuery["search"] | undefined;
  error: string | undefined;
  retry: () => void;
}

export function useSearch(term: string, active: boolean): ISearchState {
  const [results, setResults] = useState<SearchQuery["search"]>();
  const [error, setError] = useState<string>();

  const seq = useRef(0);
  const debounceTimer = useRef<number | undefined>(undefined);
  const lastTerm = useRef("");
  // the term the current results belong to
  const resultsTerm = useRef("");

  const run = useCallback(async (searchTerm: string) => {
    const id = ++seq.current;
    lastTerm.current = searchTerm;

    try {
      const res = await getClient().query<SearchQuery>({
        query: SearchDocument,
        variables: { input: { term: searchTerm } },
        fetchPolicy: "no-cache",
      });

      if (id !== seq.current) return;

      resultsTerm.current = searchTerm;
      setResults(res.data.search);
      setError(undefined);
    } catch (e) {
      if (id !== seq.current) return;
      setResults(undefined);
      setError(e instanceof Error ? e.message : String(e));
    }
  }, []);

  const retry = useCallback(() => {
    if (lastTerm.current) {
      run(lastTerm.current);
    }
  }, [run]);

  useEffect(() => {
    window.clearTimeout(debounceTimer.current);

    const trimmed = term.trim();

    if (!active || !trimmed) {
      // invalidate any in-flight request and clear state
      seq.current++;
      setError(undefined);
      if (!trimmed) {
        resultsTerm.current = "";
        setResults(undefined);
      }
      return;
    }

    if (resultsTerm.current !== trimmed) {
      // scheduling a different term - invalidate any in-flight request for
      // the previous term and hide its results so they cannot be selected
      // under the new term while waiting for the debounced query
      seq.current++;
      resultsTerm.current = "";
      setResults(undefined);
      setError(undefined);
    }

    debounceTimer.current = window.setTimeout(() => {
      run(trimmed);
    }, DEBOUNCE_MS);

    return () => {
      window.clearTimeout(debounceTimer.current);
    };
  }, [term, active, run]);

  useEffect(() => {
    return () => {
      window.clearTimeout(debounceTimer.current);
    };
  }, []);

  return { results, error, retry };
}

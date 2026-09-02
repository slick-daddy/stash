import { useCallback, useState } from "react";

const STORAGE_KEY = "headerSearch.recentSearches";
const MAX_RECENT_SEARCHES = 8;

function loadRecentSearches(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((v): v is string => typeof v === "string");
  } catch {
    return [];
  }
}

function saveRecentSearches(searches: string[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(searches));
  } catch {
    // ignore - non-fatal if persistence is unavailable
  }
}

// useRecentSearches persists recent search terms to localStorage,
// most recent first.
export function useRecentSearches() {
  const [recentSearches, setRecentSearches] =
    useState<string[]>(loadRecentSearches);

  const addRecentSearch = useCallback((term: string) => {
    setRecentSearches((current) => {
      const trimmed = term.trim();
      if (!trimmed) return current;

      const ret = [
        trimmed,
        ...current.filter((v) => v.toLowerCase() !== trimmed.toLowerCase()),
      ].slice(0, MAX_RECENT_SEARCHES);

      saveRecentSearches(ret);
      return ret;
    });
  }, []);

  return { recentSearches, addRecentSearch };
}

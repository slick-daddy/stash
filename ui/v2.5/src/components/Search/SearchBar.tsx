import React, { useCallback, useMemo, useRef, useState } from "react";
import { useIntl } from "react-intl";
import { Form } from "react-bootstrap";
import { useHistory } from "react-router-dom";
import {
  faSearch,
  faTimes,
  faArrowLeft,
} from "@fortawesome/free-solid-svg-icons";

import { Icon } from "src/components/Shared/Icon";
import { useOnOutsideClick } from "src/hooks/OutsideClick";
import { useRecentSearches } from "src/hooks/useRecentSearches";
import { useSearch } from "./useSearch";
import {
  SearchOverlay,
  buildSearchGroups,
  hasSearchResults,
  selectableItemCount,
  selectableItemAction,
  OverlayMode,
} from "./SearchOverlay";

export const SearchBar: React.FC = () => {
  const intl = useIntl();
  const history = useHistory();

  const [term, setTerm] = useState("");
  const [open, setOpen] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  // -1 means nothing is keyboard-selected; recents open unselected
  const [selectedIndex, setSelectedIndex] = useState(-1);

  const containerRef = useRef<HTMLDivElement>(null);
  const desktopInputRef = useRef<HTMLInputElement>(null);
  const mobileInputRef = useRef<HTMLInputElement>(null);

  const active = open || mobileOpen;
  const { results, error, retry } = useSearch(term, active);
  const { recentSearches, addRecentSearch } = useRecentSearches();

  const navigate = useCallback(
    (href: string) => {
      history.push(href);
      setOpen(false);
      setMobileOpen(false);
      setTerm("");
      if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    },
    [history]
  );

  const trimmedTerm = term.trim();
  const anyResults = hasSearchResults(results);

  const handleNavigate = useCallback(
    (href: string) => {
      if (trimmedTerm) {
        addRecentSearch(trimmedTerm);
      }
      navigate(href);
      setSelectedIndex(-1);
    },
    [addRecentSearch, navigate, trimmedTerm]
  );

  const groups = useMemo(
    () =>
      trimmedTerm && results && anyResults
        ? buildSearchGroups(results, trimmedTerm, handleNavigate)
        : [],
    [results, anyResults, trimmedTerm, handleNavigate]
  );

  // recents are keyboard selectable when the query is empty
  const itemCount = !trimmedTerm
    ? recentSearches.length
    : selectableItemCount(groups);

  const mode: OverlayMode = useMemo(() => {
    if (!trimmedTerm) {
      return recentSearches.length > 0 ? "recents" : "idle";
    }
    if (error) return "error";
    // no results for the current term: either waiting for a response
    // (within the debounce window after clearing the previous term's
    // results, or in flight without skeletons yet) or genuinely empty
    if (!results || !anyResults) return results ? "noResults" : "loading";
    return "results";
  }, [trimmedTerm, recentSearches.length, error, results, anyResults]);

  const closeOverlay = useCallback(() => {
    setOpen(false);
    setMobileOpen(false);
    setSelectedIndex(-1);
  }, []);

  useOnOutsideClick(containerRef as React.RefObject<HTMLElement>, () =>
    setOpen(false)
  );

  const selectRecent = useCallback(
    (recent: string) => {
      addRecentSearch(recent);
      setTerm(recent);
      setSelectedIndex(0);

      // focus the input so the user can keep typing
      const input = mobileOpen
        ? mobileInputRef.current
        : desktopInputRef.current;
      input?.focus();
    },
    [addRecentSearch, mobileOpen]
  );

  const onKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      switch (e.key) {
        case "ArrowDown":
          if (itemCount > 0) {
            e.preventDefault();
            setSelectedIndex((i) => (i + 1) % itemCount);
          }
          break;
        case "ArrowUp":
          if (itemCount > 0) {
            e.preventDefault();
            setSelectedIndex((i) => (i - 1 + itemCount) % itemCount);
          }
          break;
        case "Enter": {
          e.preventDefault();

        if (!trimmedTerm) {
          if (
            recentSearches.length > 0 &&
            selectedIndex !== -1 &&
            selectedIndex < recentSearches.length
          ) {
            selectRecent(recentSearches[selectedIndex]);
          }
          break;
        }

          // navigate to the selected item in the flattened list of
          // result rows and "see all" actions
          selectableItemAction(groups, selectedIndex)?.();
          break;
        }
        case "Escape":
          e.preventDefault();
          closeOverlay();
          break;
      }
    },
    [
      itemCount,
      trimmedTerm,
      recentSearches,
      selectedIndex,
      selectRecent,
      groups,
      closeOverlay,
    ]
  );

  function renderDesktop() {
    return (
      <Form className="header-search-form d-none d-xl-flex align-items-center">
        <Icon icon={faSearch} className="header-search-icon" />
        <Form.Control
          ref={desktopInputRef}
          type="text"
          className="header-search-input"
          placeholder={intl.formatMessage({
            id: "search.placeholder",
            defaultMessage: "Search…",
          })}
          value={term}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
            setTerm(e.target.value);
            setSelectedIndex(e.target.value ? 0 : -1);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={onKeyDown}
          autoComplete="off"
          spellCheck={false}
          aria-label={intl.formatMessage({
            id: "search.placeholder",
            defaultMessage: "Search…",
          })}
        />
        {term && (
          <button
            type="button"
            className="btn minimal header-search-clear"
            onClick={() => {
              setTerm("");
              setSelectedIndex(-1);
              desktopInputRef.current?.focus();
            }}
          >
            <Icon icon={faTimes} />
          </button>
        )}
        {open && (
          <div className="header-search-dropdown">
            <SearchOverlay
              mode={mode}
              term={trimmedTerm}
              error={error}
              onRetry={retry}
              recentSearches={recentSearches}
              onRecentSelected={selectRecent}
              groups={groups}
              selectedIndex={selectedIndex}
            />
          </div>
        )}
      </Form>
    );
  }

  function renderMobile() {
    if (!mobileOpen) {
      return (
        <button
          type="button"
          className="btn minimal d-xl-none"
          title={intl.formatMessage({
            id: "search.placeholder",
            defaultMessage: "Search…",
          })}
          onClick={() => {
            setMobileOpen(true);
            setSelectedIndex(-1);
          }}
        >
          <Icon icon={faSearch} />
        </button>
      );
    }

    return (
      <div className="header-search-fullscreen d-xl-none">
        <div className="header-search-fullscreen-bar d-flex align-items-center">
          <button
            type="button"
            className="btn minimal"
            onClick={() => {
              setMobileOpen(false);
              setTerm("");
            }}
          >
            <Icon icon={faArrowLeft} />
          </button>
          <Form.Control
            ref={mobileInputRef}
            type="text"
            className="header-search-input"
            placeholder={intl.formatMessage({
              id: "search.placeholder",
              defaultMessage: "Search…",
            })}
            value={term}
            autoFocus
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
              setTerm(e.target.value);
              setSelectedIndex(e.target.value ? 0 : -1);
            }}
            onKeyDown={onKeyDown}
            autoComplete="off"
            spellCheck={false}
            aria-label={intl.formatMessage({
              id: "search.placeholder",
              defaultMessage: "Search…",
            })}
          />
        </div>
        <div className="header-search-fullscreen-results">
          <SearchOverlay
            mode={mode}
            term={trimmedTerm}
            error={error}
            onRetry={retry}
            recentSearches={recentSearches}
            onRecentSelected={selectRecent}
            groups={groups}
            selectedIndex={selectedIndex}
          />
        </div>
      </div>
    );
  }

  return (
    <div ref={containerRef} className="header-search-container">
      {renderDesktop()}
      {renderMobile()}
    </div>
  );
};

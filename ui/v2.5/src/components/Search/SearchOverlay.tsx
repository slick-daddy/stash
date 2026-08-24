import React, { useEffect, useRef } from "react";
import { FormattedMessage, useIntl } from "react-intl";
import {
  faFilm,
  faHistory,
  faImages,
  faMapMarkerAlt,
  faPlayCircle,
  faTag,
  faUser,
  faVideo,
} from "@fortawesome/free-solid-svg-icons";
import { IconDefinition } from "@fortawesome/fontawesome-svg-core";

import { SearchQuery } from "src/core/generated-graphql";
import TextUtils from "src/utils/text";
import {
  SearchResultGroup,
  SearchRecentHeader,
  SearchSkeletonGroup,
} from "./SearchResultGroup";
import { SearchResultRow } from "./SearchResultRow";

type SearchResults = SearchQuery["search"];

export interface ISearchDisplayItem {
  key: string;
  node: (selected: boolean) => React.ReactNode;
  onSelect: () => void;
}

interface ISearchGroupData {
  typeLabel: string;
  icon: IconDefinition;
  total: number;
  seeAllMessage: React.ReactNode;
  href: string;
  items: ISearchDisplayItem[];
  onSeeAll: () => void;
}

export function hasSearchResults(results: SearchResults | undefined) {
  if (!results) return false;

  const counts = results.totalCounts;
  return (
    counts.scenes > 0 ||
    counts.performers > 0 ||
    counts.tags > 0 ||
    counts.studios > 0 ||
    counts.galleries > 0 ||
    counts.groups > 0 ||
    counts.markers > 0
  );
}

// joinSubtitle combines the given parts into a single subtitle string,
// omitting missing parts.
function joinSubtitle(parts: (string | null | undefined)[]) {
  const filtered = parts.filter((part) => !!part);
  return filtered.length > 0 ? filtered.join(" · ") : undefined;
}

function sceneSubtitle(
  studioName: string | null | undefined,
  duration: number | null | undefined
) {
  return joinSubtitle([
    studioName,
    duration != null ? TextUtils.secondsToTimestamp(duration) : null,
  ]);
}

// Builds the grouped search results for the given response. Each group
// contains its result rows followed by its "see all N" action item.
export function buildSearchGroups(
  results: SearchResults,
  term: string,
  onNavigate: (href: string) => void
): ISearchGroupData[] {
  const q = encodeURIComponent(term);

  const defs = [
    {
      typeLabel: "scenes",
      icon: faPlayCircle,
      total: results.totalCounts.scenes,
      href: `/scenes?q=${q}`,
      items: results.scenes.map<ISearchDisplayItem>((scene) => ({
        key: `scene-${scene.id}`,
        node: (selected: boolean) => (
          <SearchResultRow
            key={scene.id}
            icon={faPlayCircle}
            imagePath={scene.thumbnailPath}
            title={scene.title}
            subtitle={sceneSubtitle(scene.studioName, scene.duration)}
            selected={selected}
            onSelect={() => onNavigate(`/scenes/${scene.id}`)}
          />
        ),
        onSelect: () => onNavigate(`/scenes/${scene.id}`),
      })),
    },
    {
      typeLabel: "performers",
      icon: faUser,
      total: results.totalCounts.performers,
      href: `/performers?q=${q}`,
      items: results.performers.map<ISearchDisplayItem>((performer) => ({
        key: `performer-${performer.id}`,
        node: (selected: boolean) => (
          <SearchResultRow
            key={performer.id}
            icon={faUser}
            imagePath={performer.imagePath}
            title={performer.name}
            selected={selected}
            onSelect={() => onNavigate(`/performers/${performer.id}`)}
          />
        ),
        onSelect: () => onNavigate(`/performers/${performer.id}`),
      })),
    },
    {
      typeLabel: "tags",
      icon: faTag,
      total: results.totalCounts.tags,
      href: `/tags?q=${q}`,
      items: results.tags.map<ISearchDisplayItem>((tag) => ({
        key: `tag-${tag.id}`,
        node: (selected: boolean) => (
          <SearchResultRow
            key={tag.id}
            icon={faTag}
            title={tag.name}
            subtitle={
              tag.sceneCount != null ? (
                <FormattedMessage
                  id="search.scene_count"
                  defaultMessage="{count, plural, one {# scene} other {# scenes}}"
                  values={{ count: tag.sceneCount }}
                />
              ) : undefined
            }
            selected={selected}
            onSelect={() => onNavigate(`/tags/${tag.id}`)}
          />
        ),
        onSelect: () => onNavigate(`/tags/${tag.id}`),
      })),
    },
    {
      typeLabel: "studios",
      icon: faVideo,
      total: results.totalCounts.studios,
      href: `/studios?q=${q}`,
      items: results.studios.map<ISearchDisplayItem>((studio) => ({
        key: `studio-${studio.id}`,
        node: (selected: boolean) => (
          <SearchResultRow
            key={studio.id}
            icon={faVideo}
            imagePath={studio.imagePath}
            title={studio.name}
            selected={selected}
            onSelect={() => onNavigate(`/studios/${studio.id}`)}
          />
        ),
        onSelect: () => onNavigate(`/studios/${studio.id}`),
      })),
    },
    {
      typeLabel: "galleries",
      icon: faImages,
      total: results.totalCounts.galleries,
      href: `/galleries?q=${q}`,
      items: results.galleries.map<ISearchDisplayItem>((gallery) => ({
        key: `gallery-${gallery.id}`,
        node: (selected: boolean) => (
          <SearchResultRow
            key={gallery.id}
            icon={faImages}
            imagePath={gallery.coverPath}
            title={gallery.title}
            selected={selected}
            onSelect={() => onNavigate(`/galleries/${gallery.id}`)}
          />
        ),
        onSelect: () => onNavigate(`/galleries/${gallery.id}`),
      })),
    },
    {
      typeLabel: "groups",
      icon: faFilm,
      total: results.totalCounts.groups,
      href: `/groups?q=${q}`,
      items: results.groups.map<ISearchDisplayItem>((group) => ({
        key: `group-${group.id}`,
        node: (selected: boolean) => (
          <SearchResultRow
            key={group.id}
            icon={faFilm}
            imagePath={group.thumbnailPath}
            title={group.name}
            selected={selected}
            onSelect={() => onNavigate(`/groups/${group.id}`)}
          />
        ),
        onSelect: () => onNavigate(`/groups/${group.id}`),
      })),
    },
    {
      typeLabel: "markers",
      icon: faMapMarkerAlt,
      total: results.totalCounts.markers,
      href: `/scenes/markers?q=${q}`,
      items: results.markers.map<ISearchDisplayItem>((marker) => ({
        key: `marker-${marker.id}`,
        node: (selected: boolean) => (
          <SearchResultRow
            key={marker.id}
            icon={faMapMarkerAlt}
            title={marker.title}
            subtitle={joinSubtitle([
              marker.sceneName,
              marker.seconds != null
                ? TextUtils.secondsToTimestamp(marker.seconds)
                : null,
            ])}
            selected={selected}
            onSelect={() => onNavigate(`/scenes/${marker.sceneId}?tab=markers`)}
          />
        ),
        onSelect: () => onNavigate(`/scenes/${marker.sceneId}?tab=markers`),
      })),
    },
  ];

  return defs
    .map((def) => ({
      ...def,
      seeAllMessage: (
        <FormattedMessage
          id={`search.see_all_${def.typeLabel}`}
          defaultMessage={`See all {count, plural, one {# ${def.typeLabel.slice(0, -1)}} other {# ${def.typeLabel}}}`}
          values={{ count: def.total }}
        />
      ),
      onSeeAll: () => onNavigate(def.href),
    }))
    .filter((group) => group.items.length > 0);
}

// Returns the number of keyboard-selectable items for the given groups:
// each group contributes its rows plus its "see all" action.
export function selectableItemCount(groups: ISearchGroupData[]) {
  return groups.reduce((sum, group) => sum + group.items.length + 1, 0);
}

// selectableItemAction returns the action for the item at the given index
// within the flattened list of groups' items and "see all" actions.
export function selectableItemAction(
  groups: ISearchGroupData[],
  index: number
): (() => void) | undefined {
  let flat = 0;
  for (const group of groups) {
    for (const item of group.items) {
      if (flat === index) return item.onSelect;
      flat++;
    }

    if (flat === index) return group.onSeeAll;
    flat++;
  }

  return undefined;
}

export type OverlayMode =
  | "idle"
  | "recents"
  | "error"
  | "loading"
  | "noResults"
  | "results";

interface ISearchOverlayProps {
  mode: OverlayMode;
  term: string;
  error: string | undefined;
  onRetry: () => void;
  recentSearches: string[];
  onRecentSelected: (term: string) => void;
  groups: ISearchGroupData[];
  selectedIndex: number;
}

export const SearchOverlay: React.FC<ISearchOverlayProps> = ({
  mode,
  term,
  error,
  onRetry,
  recentSearches,
  onRecentSelected,
  groups,
  selectedIndex,
}) => {
  const intl = useIntl();
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (selectedIndex === -1) return;
    listRef.current
      ?.querySelector(".selected")
      ?.scrollIntoView({ block: "nearest" });
  }, [selectedIndex]);

  if (mode === "idle") {
    return (
      <div className="search-overlay-empty">
        <FormattedMessage
          id="search.type_to_search"
          defaultMessage="Type to search"
        />
      </div>
    );
  }

  if (mode === "recents") {
    return (
      <div ref={listRef}>
        <div className="search-result-group">
          <SearchRecentHeader />
          {recentSearches.map((recent, i) => (
            <SearchResultRow
              key={recent}
              icon={faHistory}
              title={recent}
              selected={i === selectedIndex}
              onSelect={() => onRecentSelected(recent)}
            />
          ))}
        </div>
      </div>
    );
  }

  if (mode === "error") {
    return (
      <div className="search-overlay-error">
        <FormattedMessage
          id="search.error"
          defaultMessage="An error occurred while searching"
        />
        <div className="search-overlay-error-detail">{error}</div>
        <button
          type="button"
          className="btn btn-secondary btn-sm mt-2"
          onClick={onRetry}
        >
          <FormattedMessage id="search.retry" defaultMessage="Retry" />
        </button>
      </div>
    );
  }

  if (mode === "loading") {
    return (
      <div ref={listRef}>
        {[0, 1].map((i) => (
          <SearchSkeletonGroup key={i} />
        ))}
      </div>
    );
  }

  if (mode === "noResults") {
    return (
      <div className="search-overlay-empty">
        <FormattedMessage
          id="search.no_results"
          defaultMessage='No results found for "{term}"'
          values={{ term }}
        />
      </div>
    );
  }

  let flatIndex = 0;
  return (
    <div ref={listRef} className="search-overlay-list">
      {groups.map((group) => {
        const startIndex = flatIndex;
        flatIndex += group.items.length;
        const seeAllIndex = flatIndex++;

        return (
          <SearchResultGroup
            key={group.href}
            typeLabel={intl.formatMessage({
              id: group.typeLabel,
              defaultMessage: group.typeLabel,
            })}
            count={group.total}
            icon={group.icon}
            seeAllMessage={group.seeAllMessage}
            seeAllSelected={selectedIndex === seeAllIndex}
            onSeeAll={group.onSeeAll}
          >
            {group.items.map((item, i) =>
              item.node(selectedIndex === startIndex + i)
            )}
          </SearchResultGroup>
        );
      })}
    </div>
  );
};

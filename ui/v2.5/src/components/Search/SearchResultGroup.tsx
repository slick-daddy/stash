import React from "react";
import { FormattedMessage } from "react-intl";
import { IconDefinition } from "@fortawesome/fontawesome-svg-core";

import { Icon } from "src/components/Shared/Icon";

interface ISearchResultGroupProps {
  typeLabel: string;
  icon: IconDefinition;
  // message for the "See all N results" link
  seeAllMessage: React.ReactNode;
  onSeeAll: () => void;
  // true when the see all action is highlighted by keyboard navigation
  seeAllSelected: boolean;
  children: React.ReactNode;
}

export const SearchResultGroup: React.FC<ISearchResultGroupProps> = ({
  typeLabel,
  icon,
  seeAllMessage,
  onSeeAll,
  seeAllSelected,
  children,
}) => {
  return (
    <div className="search-result-group">
      <div className="search-result-group-header d-flex align-items-center">
        <Icon icon={icon} className="search-result-group-icon" />
        <span className="search-result-group-type">{typeLabel}</span>
      </div>
      {children}
      <div className="search-result-see-all-row d-flex justify-content-end">
        <button
          type="button"
          className={`search-result-see-all btn minimal${
            seeAllSelected ? " selected" : ""
          }`}
          onClick={onSeeAll}
        >
          <span>{seeAllMessage}</span>
        </button>
      </div>
    </div>
  );
};

export const SearchSkeletonGroup: React.FC<{ rows?: number }> = ({
  rows = 3,
}) => (
  <div className="search-result-group">
    <div className="search-skeleton-header" />
    {Array.from({ length: rows }).map((_, i) => (
      <div key={i} className="search-skeleton-row d-flex align-items-center">
        <div className="search-skeleton-thumb" />
        <div className="search-skeleton-lines">
          <div className="search-skeleton-line" />
          <div className="search-skeleton-line short" />
        </div>
      </div>
    ))}
  </div>
);

export const SearchRecentHeader: React.FC = () => (
  <div className="search-result-group-header search-recent-header">
    <FormattedMessage
      id="search.recent_searches"
      defaultMessage="Recent searches"
    />
  </div>
);

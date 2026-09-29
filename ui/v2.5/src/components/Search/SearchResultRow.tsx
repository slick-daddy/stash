import React from "react";
import { IconDefinition } from "@fortawesome/fontawesome-svg-core";
import { Icon } from "src/components/Shared/Icon";

interface ISearchResultRowProps {
  icon: IconDefinition;
  imagePath?: string | null;
  title: string;
  subtitle?: React.ReactNode;
  selected: boolean;
  onSelect: () => void;
}

export const SearchResultRow: React.FC<ISearchResultRowProps> = ({
  icon,
  imagePath,
  title,
  subtitle,
  selected,
  onSelect,
}) => {
  return (
    <button
      type="button"
      className={`search-result-row btn minimal d-flex align-items-center${
        selected ? " selected" : ""
      }`}
      onClick={onSelect}
    >
      <span className="search-result-thumb d-flex align-items-center justify-content-center">
        {imagePath ? (
          <img src={imagePath} alt="" loading="lazy" />
        ) : (
          <Icon icon={icon} className="search-result-thumb-icon" />
        )}
      </span>
      <span className="search-result-text">
        <span className="search-result-title" title={title}>
          {title}
        </span>
        {subtitle ? (
          <span className="search-result-subtitle">{subtitle}</span>
        ) : null}
      </span>
    </button>
  );
};

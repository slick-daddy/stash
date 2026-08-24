import React, { useState } from "react";
import { Button, ButtonGroup, OverlayTrigger, Tooltip } from "react-bootstrap";
import { useIntl } from "react-intl";
import { ScrapeListMergeMode } from "src/core/config";
import { useConfigureUISetting } from "src/core/StashService";
import { useConfigurationContextOptional } from "src/hooks/Config";
import { useToast } from "src/hooks/Toast";
import { IHasStoredID, sortStoredIdObjects } from "src/utils/data";
import { ObjectListScrapeResult, ScrapeResult } from "./scrapeResult";

function isValidMergeMode(mode: unknown): mode is ScrapeListMergeMode {
  return mode === "merge" || mode === "overwrite";
}

// isEqualList returns true if v1 and v2 have the same length and each value
// in v1 is equal (per isEqual) to the value at the same position in v2.
function isEqualList<T>(
  isEqual: (v1: T, v2: T) => boolean
): (v1: T[], v2: T[]) => boolean {
  return (v1, v2) =>
    v1.length === v2.length && v1.every((v, i) => isEqual(v, v2[i]));
}

// usePersistedMergeMode returns the persisted merge mode for the given field,
// falling back to the given default, and a function to persist a new mode in
// the UI configuration.
//
// The backend splits configuration keys on "." and writes single leaves, so
// each field is persisted under its own key (eg
// scrapeDialogMergeModes.scene_urls). Field names must therefore not contain
// periods; callers use underscore-separated names.
function usePersistedMergeMode(
  field: string,
  defaultMode: ScrapeListMergeMode
) {
  const Toast = useToast();
  const context = useConfigurationContextOptional();
  const [saveUISetting] = useConfigureUISetting();

  const mergeModes = context?.configuration?.ui?.scrapeDialogMergeModes;
  const persisted = mergeModes?.[field];
  const initialMode = isValidMergeMode(persisted) ? persisted : defaultMode;

  async function persistMode(mode: ScrapeListMergeMode) {
    try {
      await saveUISetting({
        variables: {
          key: `scrapeDialogMergeModes.${field}`,
          value: mode,
        },
      });
    } catch (e) {
      Toast.error(e);
    }
  }

  return { initialMode, persistMode };
}

export interface IMergeModeScrapeResult<R> {
  result: R;
  setResult: React.Dispatch<React.SetStateAction<R>>;
  mergeMode: ScrapeListMergeMode;
  onSetMergeMode: (mode: ScrapeListMergeMode) => void;
}

// useMergeModeResultState manages a scrape result with a persisted merge
// mode. init must return the initial (unmerged) result, plus functions to
// apply and remove the merged-in existing values.
function useMergeModeResultState<V, R extends ScrapeResult<V>>(
  field: string,
  defaultMode: ScrapeListMergeMode,
  init: () => {
    base: R;
    merge: (r: R) => R;
    overwrite: (r: R) => R;
  }
): IMergeModeScrapeResult<R> {
  const { initialMode, persistMode } = usePersistedMergeMode(
    field,
    defaultMode
  );
  const [mode, setMode] = useState(initialMode);
  const [fns] = useState(init);

  const [result, setResult] = useState<R>(() =>
    initialMode === "merge" && fns.base.scraped ? fns.merge(fns.base) : fns.base
  );

  function onSetMergeMode(m: ScrapeListMergeMode) {
    if (m === mode) {
      return;
    }

    setMode(m);
    persistMode(m);
    setResult(m === "merge" ? fns.merge(result) : fns.overwrite(result));
  }

  return { result, setResult, mergeMode: mode, onSetMergeMode };
}

interface IMergeModeListOptions<T, R extends ScrapeResult<T[]>> {
  field: string;
  defaultMode?: ScrapeListMergeMode;
  // creates the initial scrape result, without any merging applied
  createResult: () => R;
  isEqual: (v1: T, v2: T) => boolean;
  // optional sort applied to merged values
  sortValues?: (values: T[]) => T[];
}

// useMergeModeList manages a list scrape result with a persisted merge mode.
// In merge mode, existing values missing from the scraped result are added to
// it. Toggling back to overwrite mode removes them again, leaving any other
// changes intact.
export function useMergeModeList<T, R extends ScrapeResult<T[]>>(
  options: IMergeModeListOptions<T, R>
): IMergeModeScrapeResult<R> {
  const {
    field,
    defaultMode = "overwrite",
    createResult,
    isEqual,
    sortValues,
  } = options;

  return useMergeModeResultState<T[], R>(field, defaultMode, () => {
    const base = createResult();
    const scrapedValues = base.newValue ?? [];

    // capture the existing values missing from the scraped result, so that
    // the merge can be applied and removed after the fact
    const missingExisting = (base.originalValue ?? []).filter(
      (v) => !scrapedValues.some((vv) => isEqual(v, vv))
    );

    function merge(r: R) {
      const current = r.newValue ?? [];
      const original = base.originalValue ?? [];
      // preserve the order of the existing values: urls[0] is treated as the
      // primary URL, so merged-in values must not be moved to the end
      const merged = [
        ...original.filter(
          (v) =>
            missingExisting.some((vv) => isEqual(v, vv)) ||
            current.some((vv) => isEqual(v, vv))
        ),
        ...current.filter((v) => !original.some((vv) => isEqual(v, vv))),
      ];
      if (isEqualList(isEqual)(merged, current)) return r;
      return r.cloneWithValue(sortValues ? sortValues(merged) : merged);
    }

    function overwrite(r: R) {
      const current = r.newValue ?? [];
      const filtered = current.filter(
        (v) => !missingExisting.some((vv) => isEqual(v, vv))
      );
      if (filtered.length === current.length) {
        return r;
      }

      return r.cloneWithValue(filtered);
    }

    return { base, merge, overwrite };
  });
}

// useMergeModeStringList manages a string list scrape result with a persisted
// merge mode. Defaults to merge mode to preserve the existing URL behaviour.
export function useMergeModeStringList(
  field: string,
  originalValue: string[] | undefined | null,
  scrapedValue: string[] | undefined | null,
  defaultMode: ScrapeListMergeMode = "merge"
): IMergeModeScrapeResult<ScrapeResult<string[]>> {
  return useMergeModeList({
    field,
    defaultMode,
    createResult: () => new ScrapeResult<string[]>(originalValue, scrapedValue),
    isEqual: (v1: string, v2: string) => v1 === v2,
  });
}

// useMergeModeObjectList manages an object list scrape result with a
// persisted merge mode. Values are compared by stored_id.
export function useMergeModeObjectList<T extends IHasStoredID>(
  field: string,
  originalValue: T[] | undefined | null,
  scrapedValue: T[] | undefined | null,
  defaultMode: ScrapeListMergeMode = "overwrite"
): IMergeModeScrapeResult<ObjectListScrapeResult<T>> {
  return useMergeModeList({
    field,
    defaultMode,
    createResult: () =>
      new ObjectListScrapeResult<T>(
        sortStoredIdObjects(originalValue ?? undefined),
        sortStoredIdObjects(scrapedValue ?? undefined)
      ),
    isEqual: (v1: T, v2: T) => v1.stored_id === v2.stored_id,
    sortValues: (values) => sortStoredIdObjects(values) ?? [],
  });
}

function splitValues(v: string | undefined): string[] {
  return v
    ? v
        .split(",")
        .map((vv) => vv.trim())
        .filter((vv) => vv.length > 0)
    : [];
}

// useMergeModeDelimitedString manages a comma-delimited string scrape result
// (such as performer aliases) with a persisted merge mode. Values are
// compared case-insensitively.
export function useMergeModeDelimitedString(
  field: string,
  originalValue: string | undefined | null,
  scrapedValue: string | undefined | null,
  defaultMode: ScrapeListMergeMode = "overwrite"
): IMergeModeScrapeResult<ScrapeResult<string>> {
  return useMergeModeResultState<string, ScrapeResult<string>>(
    field,
    defaultMode,
    () => {
      const base = new ScrapeResult<string>(originalValue, scrapedValue);

      const isEqual = (v1: string, v2: string) =>
        v1.toLowerCase() === v2.toLowerCase();

      const scrapedValues = splitValues(base.newValue);
      const missingExisting = splitValues(base.originalValue).filter(
        (v) => !scrapedValues.some((vv) => isEqual(v, vv))
      );

      function merge(r: ScrapeResult<string>) {
        const current = splitValues(r.newValue);
        const missing = missingExisting.filter(
          (v) => !current.some((vv) => isEqual(v, vv))
        );
        if (missing.length === 0) {
          return r;
        }

        return r.cloneWithValue([...missing, ...current].join(", "));
      }

      function overwrite(r: ScrapeResult<string>) {
        const current = splitValues(r.newValue);
        const filtered = current.filter(
          (v) => !missingExisting.some((vv) => isEqual(v, vv))
        );
        if (filtered.length === current.length) {
          return r;
        }

        return r.cloneWithValue(filtered.join(", "));
      }

      return { base, merge, overwrite };
    }
  );
}

const mergeModes: ScrapeListMergeMode[] = ["merge", "overwrite"];

const modeMessageIDs: Record<ScrapeListMergeMode, string> = {
  merge: "actions.merge",
  overwrite: "actions.overwrite",
};

const modeTooltipIDs: Record<ScrapeListMergeMode, string> = {
  merge: "dialogs.scrape_results_merge_tooltip",
  overwrite: "dialogs.scrape_results_overwrite_tooltip",
};

export const MergeModeButtons: React.FC<{
  mode: ScrapeListMergeMode;
  onSetMode: (mode: ScrapeListMergeMode) => void;
}> = ({ mode, onSetMode }) => {
  const intl = useIntl();

  return (
    <ButtonGroup className="merge-mode-buttons">
      {mergeModes.map((m) => (
        <OverlayTrigger
          key={m}
          placement="top"
          overlay={
            <Tooltip id={`merge-mode-${m}-tooltip`}>
              {intl.formatMessage({ id: modeTooltipIDs[m] })}
            </Tooltip>
          }
        >
          <Button
            variant="primary"
            active={mode === m}
            size="sm"
            onClick={() => onSetMode(m)}
          >
            {intl.formatMessage({ id: modeMessageIDs[m] })}
          </Button>
        </OverlayTrigger>
      ))}
    </ButtonGroup>
  );
};

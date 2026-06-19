import { LibraryItemBadgeState } from "./LibraryItemAccordion";

export interface IDeriveAccordionRowStateInput {
  /** Version string of this row (one entry from `fleet_maintained_versions[]`,
   * or a cached `software_installers.version` row). */
  rowVersion: string;
  /** Version string of the currently active installer on the title. Rows that
   * match are considered active. `null`/`undefined` collapses every row into
   * the inactive state. */
  activeVersion: string | null | undefined;
  /** The title's pin value:
   * - `null`/`undefined` → no pin, active row gets `badgeState: "latest"`
   * - exact string ("149.0.7827.54") → active row gets `badgeState: "pinned"`
   * - caret-prefixed string ("^149") → active row gets `badgeState: "majorVersion"`
   *
   * Only the active row carries a badge; inactive rows always return
   * `badgeState: undefined`. */
  pinnedVersion: string | null | undefined;
}

/** Pure derivation of the accordion's per-row state from the title-level data
 * that #47623 will read out of the API. Centralized here (and unit tested) so
 * the page integration doesn't open-code the pin-vs-latest-vs-major branching
 * — keeps the parent story's "row badge matches the pin kind" rule in one
 * place that's easy to grep. */
export const deriveAccordionRowState = ({
  rowVersion,
  activeVersion,
  pinnedVersion,
}: IDeriveAccordionRowStateInput): {
  isActive: boolean;
  badgeState?: LibraryItemBadgeState;
} => {
  const isActive = !!activeVersion && rowVersion === activeVersion;
  if (!isActive) return { isActive: false };
  if (!pinnedVersion) return { isActive: true, badgeState: "latest" };
  if (pinnedVersion.startsWith("^")) {
    return { isActive: true, badgeState: "majorVersion" };
  }
  return { isActive: true, badgeState: "pinned" };
};

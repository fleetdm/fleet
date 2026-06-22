import React, { useEffect, useRef, useState } from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import TruncatedTextList from "components/TruncatedTextList";
import { ILabelSoftwareTitle } from "interfaces/label";
import { InstallerType } from "interfaces/software";
import InstallerDetailsWidget from "pages/SoftwarePage/SoftwareTitleDetailsPage/SoftwareInstallerCard/InstallerDetailsWidget";
import { stringToClipboard } from "utilities/copy_text";

const baseClass = "library-item-accordion";

export type LibraryItemLabelKind = "includeAny" | "includeAll" | "excludeAny";

/** Which status badge the active row renders, if any. `undefined` (the
 * default) renders no badge. The three states are mutually exclusive by
 * construction — the type system, not prop comments, enforces this. */
export type LibraryItemBadgeState = "latest" | "pinned" | "majorVersion";

const LABEL_KIND_HEADING: Record<LibraryItemLabelKind, string> = {
  includeAny: "Include any",
  includeAll: "Include all",
  excludeAny: "Exclude any",
};

export interface ILibraryItemAccordionProps {
  /** Software title display name (or package filename for custom packages). */
  filename: string;
  version?: string | null;
  /** ISO timestamp. Rendered as "Added X ago". */
  addedAt: string;

  /** Drives the file/store icon and the version-row treatment.
   * - "package" (default): file-pkg graphic, plain version text
   * - "app-store" without `androidPlayStoreId`: Apple App Store icon, version + "Updated every hour." tooltip
   * - "app-store" with `androidPlayStoreId`: Play Store icon, "Latest" + Play Store link tooltip (web apps hide the version entirely) */
  installerType?: InstallerType;
  /** Play Store package id (e.g. `com.android.chrome`). Presence implies an Android app. */
  androidPlayStoreId?: string;
  /** Fleet-maintained app — switches the version tooltip to the "Actions > Edit" hint. */
  isFma?: boolean;
  isLatestFmaVersion?: boolean;
  /** Hide the version entirely (script-only packages). */
  isScriptPackage?: boolean;

  /** When false, the row is dimmed and the expand affordance is hidden. */
  isActive: boolean;

  /** Mirrors backend WRITE on the `SoftwareInstaller` entity — admin or
   * maintainer. Compute with `permissions.canWriteSoftware(user, teamId)`.
   * Gates every edit/delete affordance on the row: the label-count badge
   * (button → static span), the expanded-panel labels-click handler, the
   * inactive-row "Select Actions > Versions and pin this version to rollback"
   * hover tooltip, and the trash button (hidden entirely when false). */
  canEditSoftware: boolean;

  /** Which status badge the active row renders. `"latest"` → "Latest" with a
   * refresh icon. `"pinned"` → "Pinned" with a pin icon. `"majorVersion"` →
   * "Major version" with the same pin icon, distinct label. `undefined` →
   * no badge. Inactive rows never render a badge regardless. */
  badgeState?: LibraryItemBadgeState;

  /** Labels assigned to this version (drives the label-count badge and the expanded Labels row). */
  labels?: ILabelSoftwareTitle[] | null;
  /** How `labels` are scoped — matches backend label fields. Defaults to "includeAny". */
  labelKind?: LibraryItemLabelKind;

  installed: number;
  pending: number;
  failed: number;

  /** Link targets for the install-status counts. Every count renders as a
   * link to the corresponding hosts filter — there is no plain-text fallback,
   * since the production page always builds these from the title id. */
  installedPath: string;
  pendingPath: string;
  failedPath: string;

  hashSha256?: string | null;
  downloadUrl?: string;

  /** Click handler for whichever badge is rendered per `badgeState`. The
   * consumer can branch on `badgeState` inside the callback if it needs to
   * differentiate (e.g. exact vs major-version pin); the row itself fires the
   * same callback for all three. */
  onBadgeClick?: () => void;
  onLabelCountClick?: () => void;
  /** Click on the labels list in the expanded panel — opens the edit software
   * modal. Wired as a CustomLink-style underline button via TruncatedTextList. */
  onLabelsClick?: () => void;
  onDownloadClick?: () => void;
  onTrashClick?: () => void;
}

const ALL_HOSTS_LABEL = "All hosts";

const LibraryItemAccordion = ({
  filename,
  version,
  addedAt,
  installerType = "package",
  androidPlayStoreId,
  isFma = false,
  isLatestFmaVersion,
  isScriptPackage = false,
  isActive,
  canEditSoftware,
  badgeState,
  labels,
  labelKind = "includeAny",
  installed,
  pending,
  failed,
  installedPath,
  pendingPath,
  failedPath,
  hashSha256,
  downloadUrl,
  onBadgeClick,
  onLabelCountClick,
  onLabelsClick,
  onDownloadClick,
  onTrashClick,
}: ILibraryItemAccordionProps) => {
  const [expanded, setExpanded] = useState(false);
  const [copyMessage, setCopyMessage] = useState("");

  const labelCount = labels?.length ?? 0;
  const hasLabelScope = labelCount > 0;
  const showAllHostsBadge =
    isActive && !hasLabelScope && badgeState !== undefined;

  const canExpand = isActive;
  const isExpanded = canExpand && expanded;

  const toggleExpanded = () => {
    if (!canExpand) return;
    setExpanded((prev) => !prev);
  };

  // Holds the active "clear copy message" timer so a re-click resets it
  // instead of stacking timers, and so unmount can cancel it cleanly.
  const copyMessageTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (copyMessageTimer.current) clearTimeout(copyMessageTimer.current);
    };
  }, []);

  const handleCopyHash = () => {
    if (!hashSha256) return;
    const resolve = (msg: string) => {
      setCopyMessage(msg);
      if (copyMessageTimer.current) clearTimeout(copyMessageTimer.current);
      copyMessageTimer.current = setTimeout(() => setCopyMessage(""), 1000);
    };
    stringToClipboard(hashSha256)
      .then(() => resolve("Copied!"))
      .catch(() => resolve("Copy failed"));
  };

  const inactiveTooltip = (
    <>
      Select <strong>Actions &gt; Versions</strong> and pin this version to
      rollback.
    </>
  );

  const sortedLabelNames = (labels ?? [])
    .map((l) => l.name)
    .sort((a, b) => a.localeCompare(b));

  const renderLabelCountTooltip = () => (
    <div style={{ textAlign: "center" }}>
      <strong>{LABEL_KIND_HEADING[labelKind]}:</strong>
      <br />
      {sortedLabelNames.map((name, i) => (
        <React.Fragment key={name}>
          {name}
          {i < sortedLabelNames.length - 1 && <br />}
        </React.Fragment>
      ))}
    </div>
  );

  const handleBadgeClick = (handler?: () => void) => (
    e: React.MouseEvent | React.KeyboardEvent
  ) => {
    e.stopPropagation();
    handler?.();
  };

  const renderHeaderBadges = () => {
    if (!isActive) return null;

    return (
      <div className={`${baseClass}__badges`}>
        {badgeState === "latest" && (
          <Button
            variant="text-icon"
            size="small"
            onClick={handleBadgeClick(onBadgeClick)}
            className={`${baseClass}__badge-button`}
          >
            <Icon name="refresh" color="ui-fleet-black-75" />
            <span>Latest</span>
          </Button>
        )}
        {badgeState === "pinned" && (
          <Button
            variant="text-icon"
            size="small"
            onClick={handleBadgeClick(onBadgeClick)}
            className={`${baseClass}__badge-button`}
          >
            <Icon name="pin" color="ui-fleet-black-75" />
            <span>Pinned</span>
          </Button>
        )}
        {badgeState === "majorVersion" && (
          <Button
            variant="text-icon"
            size="small"
            onClick={handleBadgeClick(onBadgeClick)}
            className={`${baseClass}__badge-button`}
          >
            <Icon name="pin" color="ui-fleet-black-75" />
            <span>Major version</span>
          </Button>
        )}
        {hasLabelScope && (
          <TooltipWrapper
            tipContent={renderLabelCountTooltip()}
            showArrow
            underline={false}
            position="top"
            tipOffset={8}
          >
            {canEditSoftware ? (
              <Button
                variant="text-icon"
                size="small"
                onClick={handleBadgeClick(onLabelCountClick)}
                className={`${baseClass}__badge-button`}
              >
                <Icon name="tag" color="ui-fleet-black-75" />
                <span>{labelCount}</span>
              </Button>
            ) : (
              <span
                className={`${baseClass}__badge-button ${baseClass}__badge-button--static`}
              >
                <Icon name="tag" color="ui-fleet-black-75" />
                <span>{labelCount}</span>
              </span>
            )}
          </TooltipWrapper>
        )}
        {showAllHostsBadge && (
          <span
            className={`${baseClass}__badge-button ${baseClass}__badge-button--static`}
          >
            <Icon name="tag" color="ui-fleet-black-75" />
            <span>{ALL_HOSTS_LABEL}</span>
          </span>
        )}
      </div>
    );
  };

  const renderStatusCount = (
    iconName: "success" | "pending-outline" | "error",
    count: number,
    label: string,
    iconTooltip: React.ReactNode,
    path: string,
    trailing?: React.ReactNode
  ) => (
    <div className={`${baseClass}__status-count`}>
      <TooltipWrapper
        tipContent={iconTooltip}
        showArrow
        underline={false}
        position="top"
        tipOffset={8}
        clickable={false}
      >
        <Icon name={iconName} />
      </TooltipWrapper>
      <CustomLink
        url={path}
        text={`${count} ${label}`}
        className={`${baseClass}__status-count-link`}
      />
      {trailing}
    </div>
  );

  const statusCountsTooltip = (
    <>
      Latest status from policy automation,
      <br />
      setup experience, or manual install.
    </>
  );

  const installedIconTooltip = (
    <>
      Software is installed on these hosts
      <br />
      (install script finished with exit code 0).
      <br />
      Currently, if the software is uninstalled,
      <br />
      the &quot;Installed&quot; status won&apos;t be updated.
    </>
  );

  const pendingIconTooltip = (
    <>
      Fleet is installing/uninstalling or will
      <br />
      do so when the host comes online.
    </>
  );

  const failedIconTooltip = (
    <>
      These hosts failed to install/uninstall
      <br />
      software. Click on a host to view error(s).
    </>
  );

  const renderLabelsBlock = () => {
    if (!hasLabelScope) return null;

    return (
      <div className={`${baseClass}__data-row`}>
        <p className={`${baseClass}__data-heading`}>
          {LABEL_KIND_HEADING[labelKind]}
        </p>
        <TruncatedTextList
          className={`${baseClass}__data-value`}
          items={sortedLabelNames}
          onClick={canEditSoftware ? onLabelsClick : undefined}
        />
      </div>
    );
  };

  const renderHashBlock = () => {
    if (!hashSha256) return null;

    return (
      <div className={`${baseClass}__data-row`}>
        <p className={`${baseClass}__data-heading`}>Hash</p>
        <div className={`${baseClass}__hash-row`}>
          <TooltipTruncatedText
            className={`${baseClass}__hash`}
            value={hashSha256}
          />

          <Button
            variant="icon"
            iconStroke
            onClick={handleCopyHash}
            ariaLabel="Copy hash to clipboard"
            className={`${baseClass}__copy-button`}
          >
            <Icon name="copy" />
          </Button>
          {copyMessage && (
            <span className={`${baseClass}__copy-message`}>{copyMessage}</span>
          )}
        </div>
      </div>
    );
  };

  const renderTrashButtonBody = (disabled: boolean) => (
    <Button
      variant="icon"
      disabled={disabled}
      onClick={onTrashClick}
      ariaLabel="Delete this version"
      className={`${baseClass}__trash-button`}
    >
      <Icon name="trash" />
    </Button>
  );

  // Mirrors `SoftwareInstallerCard.SoftwareActionButtons` and complements the
  // `permissions.canWriteSoftware` role gate: only FMA and App Store / Play
  // Store rows are GitOps-locked (those installer types can't be managed via
  // YAML). Custom packages stay deletable. The "software" entity exception is
  // respected via `GitOpsModeTooltipWrapper`'s `entityType`.
  const isAppStore = installerType === "app-store";
  const lockedByGitOpsMode = isFma || isAppStore;

  const renderTrashButton = () =>
    lockedByGitOpsMode ? (
      <GitOpsModeTooltipWrapper
        position="top"
        tipOffset={8}
        entityType="software"
        renderChildren={(gitOpsDisabled) =>
          renderTrashButtonBody(!!gitOpsDisabled)
        }
      />
    ) : (
      renderTrashButtonBody(false)
    );

  const headerButton = (
    <button
      type="button"
      className={`${baseClass}__header`}
      onClick={toggleExpanded}
      aria-expanded={isExpanded}
      aria-disabled={!canExpand}
      tabIndex={canExpand ? 0 : -1}
    >
      <span
        className={classnames(`${baseClass}__chevron`, {
          [`${baseClass}__chevron--open`]: isExpanded,
        })}
      >
        <Icon name="chevron-right" color="ui-fleet-black-75" />
      </span>
      <InstallerDetailsWidget
        className={`${baseClass}__installer-details`}
        softwareName={filename}
        installerType={installerType}
        version={version}
        addedTimestamp={addedAt}
        isFma={isFma}
        isLatestFmaVersion={isLatestFmaVersion}
        isScriptPackage={isScriptPackage}
        androidPlayStoreId={androidPlayStoreId}
        hideInstallerType
        // Inactive rows wrap the entire header button in the
        // `inactiveTooltip` ("Select Actions > Versions and pin this version
        // to rollback."), which is the only hover affordance the row should
        // surface. Suppressing the widget's own tooltips (title truncation,
        // version chip, addedAt, Play Store) avoids stacking a second tooltip
        // on top of that one — Fleet UI avoids rendering two tooltips on the
        // same hover target across the app.
        disableTooltips={!isActive}
      />
      <div className={`${baseClass}__header-right`}>{renderHeaderBadges()}</div>
    </button>
  );

  return (
    <div
      className={classnames(baseClass, {
        [`${baseClass}--inactive`]: !isActive,
        [`${baseClass}--expanded`]: isExpanded,
      })}
    >
      {isActive ? (
        headerButton
      ) : (
        <TooltipWrapper
          className={`${baseClass}__inactive-tooltip`}
          tipContent={inactiveTooltip}
          showArrow
          underline={false}
          position="top"
          tipOffset={8}
          disableTooltip={!canEditSoftware}
        >
          {headerButton}
        </TooltipWrapper>
      )}

      {isExpanded && (
        <div className={`${baseClass}__panel`}>
          <div className={`${baseClass}__status-column`}>
            <div className={`${baseClass}__status-counts`}>
              {renderStatusCount(
                "success",
                installed,
                "installed",
                installedIconTooltip,
                installedPath,
                <TooltipWrapper
                  className={`${baseClass}__status-counts-info`}
                  tipContent={statusCountsTooltip}
                  showArrow
                  underline={false}
                  position="top"
                  tipOffset={8}
                >
                  <Icon name="info-outline" color="ui-fleet-black-50" />
                </TooltipWrapper>
              )}
              {renderStatusCount(
                "pending-outline",
                pending,
                "pending",
                pendingIconTooltip,
                pendingPath
              )}
              {renderStatusCount(
                "error",
                failed,
                "failed",
                failedIconTooltip,
                failedPath
              )}
            </div>
          </div>

          <div className={`${baseClass}__details-column`}>
            {renderLabelsBlock()}
            {renderHashBlock()}
          </div>

          <div className={`${baseClass}__actions-column`}>
            {downloadUrl && (
              <Button
                variant="icon"
                onClick={onDownloadClick}
                ariaLabel="Download installer"
                className={`${baseClass}__download-button`}
              >
                <Icon name="download" />
              </Button>
            )}
            {canEditSoftware && renderTrashButton()}
          </div>
        </div>
      )}
    </div>
  );
};

export default LibraryItemAccordion;

import React, { useState } from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
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

  /** Gates the inactive-row hover tooltip ("Select Actions > Versions and pin
   * this version to rollback."). Mirrors `canEditSoftware` elsewhere in this
   * area — users without edit access can't reach Actions > Versions, so the
   * hint would point at a menu they can't use. Defaults to true. */
  canEditSoftware?: boolean;

  /** Show the "Latest" badge-button. Mutually exclusive with isPinned. */
  isLatest?: boolean;
  /** Show the "Pinned" badge-button. Mutually exclusive with isLatest. */
  isPinned?: boolean;

  /** Labels assigned to this version (drives the label-count badge and the expanded Labels row). */
  labels?: ILabelSoftwareTitle[] | null;
  /** How `labels` are scoped — matches backend label fields. Defaults to "includeAny". */
  labelKind?: LibraryItemLabelKind;

  installed: number;
  pending: number;
  failed: number;

  /** Link targets for the install-status counts. When provided, the count renders as a link. */
  installedPath?: string;
  pendingPath?: string;
  failedPath?: string;

  hashSha256?: string | null;
  downloadUrl?: string;

  trashDisabled?: boolean;
  trashDisabledTooltip?: React.ReactNode;

  onLatestClick?: () => void;
  onPinnedClick?: () => void;
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
  canEditSoftware = true,
  isLatest,
  isPinned,
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
  trashDisabled,
  trashDisabledTooltip,
  onLatestClick,
  onPinnedClick,
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
    isActive && !hasLabelScope && (isLatest || isPinned);

  const canExpand = isActive;
  const isExpanded = canExpand && expanded;

  const toggleExpanded = () => {
    if (!canExpand) return;
    setExpanded((prev) => !prev);
  };

  const handleCopyHash = () => {
    if (!hashSha256) return;
    stringToClipboard(hashSha256)
      .then(() => setCopyMessage("Copied!"))
      .catch(() => setCopyMessage("Copy failed"));
    setTimeout(() => setCopyMessage(""), 1000);
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
        {isLatest && (
          <Button
            variant="text-icon"
            size="small"
            onClick={handleBadgeClick(onLatestClick)}
            className={`${baseClass}__badge-button`}
          >
            <Icon name="refresh" color="ui-fleet-black-75" />
            <span>Latest</span>
          </Button>
        )}
        {isPinned && (
          <Button
            variant="text-icon"
            size="small"
            onClick={handleBadgeClick(onPinnedClick)}
            className={`${baseClass}__badge-button`}
          >
            <Icon name="pin" color="ui-fleet-black-75" />
            <span>Pinned</span>
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
    path?: string,
    trailing?: React.ReactNode
  ) => {
    const text = `${count} ${label}`;

    return (
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
        {path ? (
          <CustomLink
            url={path}
            text={text}
            className={`${baseClass}__status-count-link`}
          />
        ) : (
          <span>{text}</span>
        )}
        {trailing}
      </div>
    );
  };

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

  const renderTrashButton = () => {
    const trashButton = (
      <Button
        variant="icon"
        disabled={trashDisabled}
        onClick={onTrashClick}
        ariaLabel="Delete this version"
        className={`${baseClass}__trash-button`}
      >
        <Icon name="trash" />
      </Button>
    );

    if (trashDisabled && trashDisabledTooltip) {
      return (
        <TooltipWrapper
          tipContent={trashDisabledTooltip}
          showArrow
          underline={false}
          position="top"
          tipOffset={8}
        >
          {trashButton}
        </TooltipWrapper>
      );
    }
    return trashButton;
  };

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
        // Hash is intentionally shown in the expanded panel only.
        sha256={null}
        hideInstallerType
        disableTitleTooltip={!isActive}
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
            {renderTrashButton()}
          </div>
        </div>
      )}
    </div>
  );
};

export default LibraryItemAccordion;

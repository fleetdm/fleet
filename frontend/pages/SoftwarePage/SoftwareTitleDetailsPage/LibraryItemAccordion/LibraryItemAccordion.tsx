import React, { useState } from "react";
import classnames from "classnames";
import { formatDistanceToNow } from "date-fns";

import Button from "components/buttons/Button";
import Graphic from "components/Graphic";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";
import { ILabelSoftwareTitle } from "interfaces/label";
import { stringToClipboard } from "utilities/copy_text";

const baseClass = "library-item-accordion";

export type LibraryItemLabelKind = "include" | "exclude";

export interface ILibraryItemAccordionProps {
  filename: string;
  version: string;
  /** ISO timestamp. Rendered as "Added X ago". */
  addedAt: string;

  /** When false, the row is dimmed and the expand affordance is hidden. */
  isActive: boolean;

  /** Show the "Latest" badge-button. Mutually exclusive with isPinned. */
  isLatest?: boolean;
  /** Show the "Pinned" badge-button. Mutually exclusive with isLatest. */
  isPinned?: boolean;

  /** Labels assigned to this version (drives the label-count badge and the expanded Labels row). */
  labels?: ILabelSoftwareTitle[] | null;
  /** Whether `labels` are include or exclude scoped. Defaults to "include". */
  labelKind?: LibraryItemLabelKind;

  installed: number;
  pending: number;
  failed: number;

  hashSha256?: string | null;
  downloadUrl?: string;

  trashDisabled?: boolean;
  trashDisabledTooltip?: React.ReactNode;

  onLatestClick?: () => void;
  onPinnedClick?: () => void;
  onLabelCountClick?: () => void;
  onInstalledClick?: () => void;
  onPendingClick?: () => void;
  onFailedClick?: () => void;
  onDownloadClick?: () => void;
  onTrashClick?: () => void;
}

const ALL_HOSTS_LABEL = "All hosts";

const formatAddedAt = (addedAt: string) => {
  const parsed = new Date(addedAt);
  if (Number.isNaN(parsed.getTime())) {
    return "";
  }
  return `Added ${formatDistanceToNow(parsed, { addSuffix: true })}`;
};

const LibraryItemAccordion = ({
  filename,
  version,
  addedAt,
  isActive,
  isLatest,
  isPinned,
  labels,
  labelKind = "include",
  installed,
  pending,
  failed,
  hashSha256,
  downloadUrl,
  trashDisabled,
  trashDisabledTooltip,
  onLatestClick,
  onPinnedClick,
  onLabelCountClick,
  onInstalledClick,
  onPendingClick,
  onFailedClick,
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

  const addedAtLabel = formatAddedAt(addedAt);

  const renderHeaderBadges = () => {
    if (!isActive) return null;

    return (
      <div className={`${baseClass}__badges`}>
        {isLatest && (
          <Button
            variant="text-icon"
            size="small"
            onClick={onLatestClick}
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
            onClick={onPinnedClick}
            className={`${baseClass}__badge-button`}
          >
            <Icon name="pin" color="ui-fleet-black-75" />
            <span>Pinned</span>
          </Button>
        )}
        {hasLabelScope && (
          <Button
            variant="text-icon"
            size="small"
            onClick={onLabelCountClick}
            className={`${baseClass}__badge-button`}
          >
            <Icon name="tag" color="ui-fleet-black-75" />
            <span>{labelCount}</span>
          </Button>
        )}
        {showAllHostsBadge && (
          <span className={`${baseClass}__all-hosts`}>{ALL_HOSTS_LABEL}</span>
        )}
      </div>
    );
  };

  const renderStatusCount = (
    iconName: "success" | "pending-outline" | "error",
    count: number,
    label: string,
    onClick?: () => void
  ) => {
    const content = (
      <>
        <Icon name={iconName} />
        <span>
          {count} {label}
        </span>
      </>
    );

    if (onClick) {
      return (
        <Button
          variant="text-icon"
          onClick={onClick}
          className={`${baseClass}__status-count`}
        >
          {content}
        </Button>
      );
    }
    return <div className={`${baseClass}__status-count`}>{content}</div>;
  };

  const statusCountsTooltip = <>Counts show installs for this version only.</>;

  const renderLabelsBlock = () => {
    if (!hasLabelScope) return null;

    const heading = labelKind === "exclude" ? "Excluded labels" : "Labels";
    const names = (labels ?? []).map((l) => l.name).join(", ");

    return (
      <div className={`${baseClass}__data-row`}>
        <p className={`${baseClass}__data-heading`}>{heading}</p>
        <p className={`${baseClass}__data-value`}>{names}</p>
      </div>
    );
  };

  const renderHashBlock = () => {
    if (!hashSha256) return null;

    return (
      <div className={`${baseClass}__data-row`}>
        <p className={`${baseClass}__data-heading`}>Hash</p>
        <div className={`${baseClass}__hash-row`}>
          <p className={`${baseClass}__data-value ${baseClass}__hash`}>
            {hashSha256}
          </p>
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
        iconStroke
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

  return (
    <div
      className={classnames(baseClass, {
        [`${baseClass}--inactive`]: !isActive,
        [`${baseClass}--expanded`]: isExpanded,
      })}
    >
      <button
        type="button"
        className={`${baseClass}__header`}
        onClick={toggleExpanded}
        aria-expanded={isExpanded}
        aria-disabled={!canExpand}
      >
        <span className={`${baseClass}__chevron`}>
          <Icon
            name={isExpanded ? "chevron-down" : "chevron-right"}
            color="ui-fleet-black-75"
          />
        </span>
        <Graphic name="file-pkg" />
        <div className={`${baseClass}__info`}>
          <span className={`${baseClass}__filename`}>{filename}</span>
          <span className={`${baseClass}__sub`}>
            {version}
            {addedAtLabel && <> &middot; {addedAtLabel}</>}
          </span>
        </div>
        <div className={`${baseClass}__header-right`}>
          {renderHeaderBadges()}
        </div>
      </button>

      {isExpanded && (
        <div className={`${baseClass}__panel`}>
          <div className={`${baseClass}__status-column`}>
            <div className={`${baseClass}__status-counts`}>
              {renderStatusCount(
                "success",
                installed,
                "installed",
                onInstalledClick
              )}
              {renderStatusCount(
                "pending-outline",
                pending,
                "pending",
                onPendingClick
              )}
              {renderStatusCount("error", failed, "failed", onFailedClick)}
            </div>
            <TooltipWrapper
              tipContent={statusCountsTooltip}
              showArrow
              underline={false}
              position="top"
              tipOffset={8}
            >
              <Icon name="info-outline" color="ui-fleet-black-50" />
            </TooltipWrapper>
          </div>

          <div className={`${baseClass}__details-column`}>
            {renderLabelsBlock()}
            {renderHashBlock()}
          </div>

          <div className={`${baseClass}__actions-column`}>
            {downloadUrl && (
              <Button
                variant="icon"
                iconStroke
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

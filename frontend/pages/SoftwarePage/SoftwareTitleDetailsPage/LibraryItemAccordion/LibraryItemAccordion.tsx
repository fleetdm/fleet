import React, { useState } from "react";
import classnames from "classnames";

import Button from "components/buttons/Button";
import CopyButton from "components/buttons/CopyButton";
import CustomLink from "components/CustomLink";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon";
import { IconNames } from "components/icons";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import TruncatedTextList from "components/TruncatedTextList";
import { ILabelSoftwareTitle } from "interfaces/label";
import { InstallerType } from "interfaces/software";
import { getSelfServiceTooltip } from "pages/SoftwarePage/helpers";
import InstallerDetailsWidget from "pages/SoftwarePage/SoftwareTitleDetailsPage/SoftwareInstallerCard/InstallerDetailsWidget";

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
  isTarballPackage?: boolean;
  /** Apple App Store app whose platform is iOS or iPadOS. Drops the
   * "policy automation" leg from the info-icon tooltip — `automatic_install`
   * is not supported for iOS or iPadOS. */
  isIosOrIpadosApp?: boolean;
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

  /** Single page-level flag mirroring `SoftwareTitleDetailsPage`'s
   * `canActivateMultiplePackages` — true for titles that can hold multiple
   * custom packages (#48400). Drives the row's self-service / auto-install
   * icons (and the absence of the Latest badge, which is FMA-only by gate). */
  canActivateMultiplePackages?: boolean;
  /** Drives the self-service icon's tooltip variant and visibility on the
   * custom-package row. Mirrors `software_package.self_service`. */
  isSelfService?: boolean;
  /** Drives the auto-install icon's visibility on the custom-package row.
   * Truthy when the package has ≥1 linked policy (auto-install OR patch). */
  hasAutoInstallPolicy?: boolean;
  /** When set, tooltips mention "Patch policy" instead of "Auto install" —
   * matches the SoftwareSummaryCard chip's tooltip split. */
  isPatchPolicyOnly?: boolean;
  /** Self-service tooltip copy varies for Android Play Store apps. Wired
   * through for completeness even though the current call sites only enable
   * the custom-package path for desktop titles. */
  isAndroidPlayStoreApp?: boolean;
  /** Click handler for the self-service icon — opens the per-package Edit
   * software modal (same target as the labels-count badge). */
  onSelfServiceClick?: () => void;
  /** Click handler for the auto-install icon — the page resolves whether to
   * navigate straight to the single linked policy or open the PoliciesModal,
   * scoped to THIS package's policies (#48400). */
  onAutoInstallClick?: () => void;
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
  isTarballPackage = false,
  isIosOrIpadosApp = false,
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
  canActivateMultiplePackages = false,
  isSelfService = false,
  hasAutoInstallPolicy = false,
  isPatchPolicyOnly = false,
  isAndroidPlayStoreApp = false,
  onSelfServiceClick,
  onAutoInstallClick,
}: ILibraryItemAccordionProps) => {
  const [expanded, setExpanded] = useState(false);

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

  // Per-row indicator icon — tooltipped, optionally wrapped in a Button when
  // a click handler is provided AND the caller's gate (e.g. permission) is
  // open. Falls back to a static Icon otherwise. Used for the self-service
  // and auto-install indicators on multi-package custom rows (#48400).
  const renderRowActionIcon = ({
    iconName,
    tooltipContent,
    ariaLabel,
    onClick,
    canClick = true,
  }: {
    iconName: IconNames;
    tooltipContent: React.ReactNode;
    ariaLabel: string;
    onClick?: () => void;
    /** When false, the icon stays static even if `onClick` is provided —
     * used to gate clickability on permission (e.g. self-service). */
    canClick?: boolean;
  }) => (
    <TooltipWrapper
      tipContent={tooltipContent}
      showArrow
      underline={false}
      position="top"
      tipOffset={8}
    >
      {onClick && canClick ? (
        <Button
          variant="inverse"
          onClick={handleBadgeClick(onClick)}
          className={`${baseClass}__icon-button`}
          ariaLabel={ariaLabel}
        >
          <Icon name={iconName} color="ui-fleet-black-75" />
        </Button>
      ) : (
        <Icon name={iconName} color="ui-fleet-black-75" />
      )}
    </TooltipWrapper>
  );

  // Self-service tooltip mirrors the SoftwareSummaryCard chip's copy so the
  // per-row indicator says the same thing the title-level chip would.
  const renderSelfServiceIcon = () =>
    renderRowActionIcon({
      iconName: "user",
      tooltipContent: getSelfServiceTooltip(
        !!isIosOrIpadosApp,
        !!isAndroidPlayStoreApp
      ),
      // Same modal opens regardless of which icon is clicked; the icon glyph
      // carries the contextual signal ("self-service is on for this package").
      ariaLabel: "Edit package",
      onClick: onSelfServiceClick,
      canClick: canEditSoftware,
    });

  // Auto-install icon navigates rather than edits — its label is verb-forward
  // ("View") so it doesn't read as a state toggle. Patch-only and dynamic
  // variants get their own labels for screen-reader clarity.
  const renderAutoInstallIcon = () =>
    renderRowActionIcon({
      iconName: isPatchPolicyOnly ? "policy" : "refresh",
      tooltipContent: isPatchPolicyOnly ? (
        <>Patch policy is linked to this package.</>
      ) : (
        <>Policy triggers install.</>
      ),
      ariaLabel: isPatchPolicyOnly
        ? "View patch policy"
        : "View auto-install policies",
      onClick: onAutoInstallClick,
    });

  const renderHeaderBadges = () => {
    if (!isActive) return null;

    return (
      <div className={`${baseClass}__badges`}>
        {/* The "Latest" badge is FMA-specific — only Fleet-maintained apps
            have a meaningful "latest available cached version" concept that
            drives the badge state. VPP / App Store / Play Store / iOS
            in-house and custom packages do not render this badge.
            Pinned / Major version variants below also stay FMA-only by
            construction (only FMA exposes version pinning). */}
        {isFma && badgeState === "latest" && (
          <Button
            variant="inverse"
            size="small"
            onClick={handleBadgeClick(onBadgeClick)}
            className={`${baseClass}__badge-button`}
          >
            <Icon name="refresh" color="ui-fleet-black-75" />
            <span>Latest</span>
          </Button>
        )}
        {canActivateMultiplePackages &&
          isSelfService &&
          renderSelfServiceIcon()}
        {canActivateMultiplePackages &&
          hasAutoInstallPolicy &&
          renderAutoInstallIcon()}
        {badgeState === "pinned" && (
          <Button
            variant="inverse"
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
            variant="inverse"
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
                variant="inverse"
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
        {showAllHostsBadge &&
          (canEditSoftware ? (
            <Button
              variant="inverse"
              size="small"
              onClick={handleBadgeClick(onLabelCountClick)}
              className={`${baseClass}__badge-button`}
            >
              <Icon name="tag" color="ui-fleet-black-75" />
              <span>{ALL_HOSTS_LABEL}</span>
            </Button>
          ) : (
            <span
              className={`${baseClass}__badge-button ${baseClass}__badge-button--static`}
            >
              <Icon name="tag" color="ui-fleet-black-75" />
              <span>{ALL_HOSTS_LABEL}</span>
            </span>
          ))}
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
      {iconTooltip ? (
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
      ) : (
        <Icon name={iconName} />
      )}
      <CustomLink
        url={path}
        text={`${count} ${label}`}
        className={`${baseClass}__status-count-link`}
      />
      {trailing}
    </div>
  );

  // Script-only packages swap the "installed" label/tooltip for "ran"
  // semantics; Android Play Store apps swap pending/failed tooltip wording
  // to match MDM check-in semantics.
  const isAndroidApp = !!androidPlayStoreId;
  const installedLabel = isScriptPackage ? "ran" : "installed";

  const getStatusCountTooltip = () => {
    if (isAndroidApp) {
      return <>Latest status from the Google Play Store</>;
    }

    if (isTarballPackage) {
      return <>Latest status from policy automation or manual install.</>;
    }

    // `automatic_install` is not supported for iOS or iPadOS, so drop the
    // policy-automation leg.
    if (isIosOrIpadosApp) {
      return <>Latest status from setup experience or manual install.</>;
    }

    return (
      <>
        Latest status from policy automation,
        <br />
        setup experience, or manual install.
      </>
    );
  };

  const getInstalledIconTooltip = (): React.ReactNode => {
    if (isScriptPackage) {
      return (
        <>
          The script successfully
          <br />
          ran on these hosts.
        </>
      );
    }
    if (isAndroidApp) {
      return null;
    }
    return (
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
  };

  const getPendingIconTooltip = (): React.ReactNode => {
    if (isScriptPackage) {
      return (
        <>
          Fleet is running the script or will do
          <br />
          so when the host comes online.
        </>
      );
    }
    if (isAndroidApp) {
      return (
        <>
          Software will be installed or configuration will
          <br />
          be applied the next time the host checks in.
        </>
      );
    }
    return (
      <>
        Fleet is installing/uninstalling or will
        <br />
        do so when the host comes online.
      </>
    );
  };

  const getFailedIconTooltip = (): React.ReactNode => {
    if (isScriptPackage) {
      return (
        <>
          These hosts failed to run the script.
          <br />
          Click on a host to view error(s).
        </>
      );
    }
    if (isAndroidApp) {
      return <>Software failed to install or configuration failed to apply.</>;
    }
    return (
      <>
        These hosts failed to install/uninstall
        <br />
        software. Click on a host to view error(s).
      </>
    );
  };

  const installedIconTooltip = getInstalledIconTooltip();
  const pendingIconTooltip = getPendingIconTooltip();
  const failedIconTooltip = getFailedIconTooltip();
  const statusCountsTooltip = getStatusCountTooltip();

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
          <CopyButton
            copyText={hashSha256}
            ariaLabel="Copy hash to clipboard"
          />
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

  // Only FMA and App Store / Play Store rows are GitOps-locked (those
  // installer types can't be managed via YAML); custom packages stay
  // deletable. The `software` entity exception is honored via the wrapper.
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

  // `<div role="button">` rather than `<button>` because the badges nested
  // inside are native `<button>`s — nesting them violates the HTML spec
  // (React fires `validateDOMNesting`). Keyboard handling mirrors
  // `DataTable.tsx`'s clickable-row pattern.
  const handleHeaderKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (!canExpand) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      toggleExpanded();
    }
  };

  const headerButton = (
    <div
      role="button"
      className={`${baseClass}__header`}
      onClick={toggleExpanded}
      onKeyDown={handleHeaderKeyDown}
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
        // Inactive rows surface a single hover tooltip (the rollback hint);
        // suppress the widget's tooltips to avoid stacking two on the same
        // target. See `InstallerDetailsWidget` for the full set silenced.
        disableTooltips={!isActive}
      />
      <div className={`${baseClass}__header-right`}>{renderHeaderBadges()}</div>
    </div>
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
                installedLabel,
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

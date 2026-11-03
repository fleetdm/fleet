/**
software/titles/:id > Top section
software/versions/:id > Top section
software/os/:id > Top section
*/

import classnames from "classnames";
import React from "react";

import ActionsDropdown from "components/ActionsDropdown";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import DataSet from "components/DataSet";
import LastUpdatedHostCount from "components/LastUpdatedHostCount";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipWrapperArchLinuxRolling from "components/TooltipWrapperArchLinuxRolling";
import { IDropdownOption } from "interfaces/dropdownOption";
import { formatVppTokenLabel, IVppTokenSummary } from "interfaces/mdm";
import {
  NO_VERSION_OR_HOST_DATA_SOURCES,
  ROLLING_ARCH_LINUX_VERSIONS,
} from "interfaces/software";
import {
  GITOPS_ID_TOOLTIP,
  isSafeImagePreviewUrl,
} from "pages/SoftwarePage/helpers";
import paths from "router/paths";
import { getPathWithQueryParams, QueryParams } from "utilities/url";

import OSIcon from "../../icons/OSIcon";
import SoftwareIcon from "../../icons/SoftwareIcon";

export const ACTION_EDIT_APPEARANCE = "edit_appearance";
export const ACTION_EDIT_SOFTWARE = "edit_software";
export const ACTION_EDIT_CONFIGURATION = "edit_configuration";
export const ACTION_DEPLOY = "deploy";
export const ACTION_VERSIONS = "versions";
export const ACTION_EDIT_AUTO_UPDATE_CONFIGURATION =
  "edit_auto_update_configuration";

export interface BuildActionOptionsArgs {
  canEditSoftware: boolean;
  canEditConfiguration: boolean;
  canDeploySoftware: boolean;
  canManageVersions: boolean;
  canConfigureAutoUpdate: boolean;
}

// Edit options stay clickable in gitops mode; each modal self-disables and
// gates Save.
export const buildActionOptions = ({
  canEditSoftware,
  canEditConfiguration,
  canDeploySoftware,
  canManageVersions,
  canConfigureAutoUpdate,
}: BuildActionOptionsArgs): IDropdownOption[] => {
  const options: IDropdownOption[] = [
    {
      label: "Edit appearance",
      value: ACTION_EDIT_APPEARANCE,
    },
  ];

  // Hides edit software option only for Android installers (Playstore and Web apps), as they are currently non-editable.
  // Stays clickable in gitops mode so admins can open the modal to view
  // applied settings; the modal disables its own fields and Save button.
  if (canEditSoftware) {
    options.push({
      label: "Edit software",
      value: ACTION_EDIT_SOFTWARE,
    });
  }

  // Show edit configuration option for Android installers (not web apps) and
  // iOS/iPadOS apps. Stays clickable in gitops mode — the modal itself
  // disables Save with the gitops tooltip, matching the Versions option.
  if (canEditConfiguration) {
    options.push({
      label: "Edit configuration",
      value: ACTION_EDIT_CONFIGURATION,
    });
  }

  // Show Deploy only for Fleet-maintained apps.
  if (canDeploySoftware) {
    options.push({
      label: "Deploy",
      value: ACTION_DEPLOY,
    });
  }

  // Show versions option only for Fleet-maintained apps on Premium. Stays
  // clickable in gitops mode — the modal itself disables Save with the gitops
  // tooltip, matching the in-row Latest/Pinned badge behavior.
  if (canManageVersions) {
    options.push({
      label: "Versions",
      value: ACTION_VERSIONS,
    });
  }

  if (canConfigureAutoUpdate) {
    options.push({
      label: "Schedule auto updates",
      value: ACTION_EDIT_AUTO_UPDATE_CONFIGURATION,
    });
  }

  return options;
};

const baseClass = "software-details-summary";

interface ISoftwareDetailsSummaryProps {
  /** Name displayed in UI */
  displayName: string;
  /** Name is keyed for fallback icon  */
  name?: string;
  type?: string;
  hostCount?: number;
  countsUpdatedAt?: string;
  /** The query param that will be added when user clicks on the host count
   * Optional as isPreview mode doesn't have host count/link
   */
  queryParams?: QueryParams;
  source?: string;
  versions?: number;
  iconUrl?: string | null;
  /** Displays OS icon instead of Software icon */
  isOperatingSystem?: boolean;
  /** Shows Actions dropdown allowing user to edit software */
  canManageSoftware?: boolean;
  /** Displays an edit CTA to edit the software's icon and display name
   * Should only be defined for team view of an installable software */
  onClickEditAppearance?: () => void;
  /** Displays an edit CTA to edit the software installer
   * Should only be defined for team view of an installable software */
  onClickEditSoftware?: () => void;
  /** Displays Deploy CTA for Fleet-maintained apps. */
  onClickDeploy?: () => void;
  /** Displays Versions CTA to open the versions / pin modal (Premium FMA only) */
  onClickVersions?: () => void;
  /** undefined unless previewing icon, in which case is string or null */
  /** Displays an edit CTA to edit the software's icon
   * Should only be defined for team view of an installable software */
  onClickEditConfiguration?: () => void;
  onClickEditAutoUpdateConfig?: () => void;
  iconPreviewUrl?: string | null;
  /** timestamp of when icon was last uploaded, used to force refresh of cached icon */
  iconUploadedAt?: string;
  /** Optional pill row rendered between the title and the Actions dropdown
   * (e.g. Fleet-maintained, Self-service, Auto install). */
  headerPills?: React.ReactNode;
  /** Custom non-FMA packages collapse the Actions dropdown into a single
   * pencil-icon "Edit" button that opens the Edit Appearance modal directly.
   * Per-installer Edit lives on the Library accordion row. */
  useSingleEditAppearanceButton?: boolean;
  fleetMaintainedAppSlug?: string;
  appStoreId?: string;
  /** Set when the title response carries a VPP token (fleet view of an Apple
   * App Store app). */
  vppToken?: IVppTokenSummary;
  googlePlayId?: string;
}

const SoftwareDetailsSummary = ({
  displayName,
  type,
  hostCount,
  countsUpdatedAt,
  queryParams,
  name,
  source,
  versions,
  iconUrl,
  isOperatingSystem,
  canManageSoftware = false,
  onClickEditAppearance,
  onClickEditSoftware,
  onClickDeploy,
  onClickVersions,
  onClickEditConfiguration,
  onClickEditAutoUpdateConfig,
  iconPreviewUrl,
  iconUploadedAt,
  headerPills,
  useSingleEditAppearanceButton = false,
  fleetMaintainedAppSlug,
  appStoreId,
  vppToken,
  googlePlayId,
}: ISoftwareDetailsSummaryProps) => {
  const hostCountPath = getPathWithQueryParams(paths.MANAGE_HOSTS, queryParams);

  const isRollingArch = ROLLING_ARCH_LINUX_VERSIONS.includes(displayName);

  const onSelectSoftwareAction = (value: string) => {
    switch (value) {
      case ACTION_EDIT_APPEARANCE:
        onClickEditAppearance && onClickEditAppearance();
        break;
      case ACTION_EDIT_SOFTWARE:
        onClickEditSoftware && onClickEditSoftware();
        break;
      case ACTION_DEPLOY:
        onClickDeploy && onClickDeploy();
        break;
      case ACTION_VERSIONS:
        onClickVersions && onClickVersions();
        break;
      case ACTION_EDIT_CONFIGURATION:
        onClickEditConfiguration && onClickEditConfiguration();
        break;
      case ACTION_EDIT_AUTO_UPDATE_CONFIGURATION:
        onClickEditAutoUpdateConfig && onClickEditAutoUpdateConfig();
        break;
      default:
    }
  };

  // Remove host count for sources without version/host data (tgz and script
  // packages) or if viewing details summary from edit icon preview modal
  const showHostCount =
    !!hostCount && !NO_VERSION_OR_HOST_DATA_SOURCES.includes(source || "");

  const renderSoftwareIcon = () => {
    if (
      typeof iconPreviewUrl === "string" &&
      isSafeImagePreviewUrl(iconPreviewUrl)
    ) {
      return (
        <img
          src={iconPreviewUrl}
          alt="Uploaded icon preview"
          style={{ width: 96, height: 96 }}
        />
      );
    }

    return (
      <SoftwareIcon
        name={name}
        source={source}
        url={iconUrl}
        uploadedAt={iconUploadedAt}
        size="xlarge"
      />
    );
  };

  const renderGitOpsIdDataSet = (title: string, value?: string, url?: string) =>
    !!value && (
      <DataSet
        title={
          <TooltipWrapper tipContent={GITOPS_ID_TOOLTIP}>
            {title}
          </TooltipWrapper>
        }
        value={url ? <CustomLink url={url} text={value} newTab /> : value}
      />
    );

  // Without a country, Apple redirects to the viewer's storefront.
  const appStoreUrl = `https://apps.apple.com/${
    vppToken?.country_code ? `${vppToken.country_code.toLowerCase()}/` : ""
  }app/id${appStoreId}`;

  const actionOptions = buildActionOptions({
    canEditSoftware: !!onClickEditSoftware,
    canEditConfiguration: !!onClickEditConfiguration,
    canDeploySoftware: !!onClickDeploy,
    canManageVersions: !!onClickVersions,
    canConfigureAutoUpdate: !!onClickEditAutoUpdateConfig,
  });

  return (
    <>
      <div
        className={classnames(baseClass, {
          [`${baseClass}--has-pills`]: !!headerPills,
        })}
      >
        <div className={`${baseClass}__icon-wrap`}>
          {isOperatingSystem ? (
            <OSIcon name={name} size="xlarge" />
          ) : (
            renderSoftwareIcon()
          )}
        </div>
        <div className={`${baseClass}__info`}>
          <h1
            aria-label="software display name"
            className={`${baseClass}__title`}
          >
            {isRollingArch ? (
              // wrap a tooltip around the "rolling" suffix
              <>
                {displayName.slice(0, -8)}
                <TooltipWrapperArchLinuxRolling />
              </>
            ) : (
              <TooltipTruncatedText value={displayName} />
            )}
          </h1>
          {canManageSoftware && (
            <div className={`${baseClass}__actions-wrapper`}>
              {useSingleEditAppearanceButton ? (
                // Clickable in gitops mode so admins can open the modal to
                // view the applied icon and display name; the modal disables
                // its own fields and Save button.
                <Button
                  variant="secondary"
                  size="small"
                  onClick={onClickEditAppearance}
                  disabled={!onClickEditAppearance}
                  icon="pencil"
                >
                  Edit
                </Button>
              ) : (
                <ActionsDropdown
                  className={`${baseClass}__actions-dropdown`}
                  onChange={onSelectSoftwareAction}
                  placeholder="Actions"
                  options={actionOptions}
                  variant="secondary"
                  menuAlign="right"
                />
              )}
            </div>
          )}
          <dl className={`${baseClass}__description-list`}>
            {!!type && <DataSet title="Type" value={type} />}

            {!!versions && <DataSet title="Versions" value={versions} />}
            {showHostCount && (
              <DataSet
                title="Hosts"
                value={
                  <LastUpdatedHostCount
                    hostCount={
                      <TooltipWrapper tipContent="View all hosts">
                        <CustomLink
                          url={hostCountPath}
                          text={hostCount.toString()}
                        />
                      </TooltipWrapper>
                    }
                    lastUpdatedAt={countsUpdatedAt}
                  />
                }
              />
            )}
            {renderGitOpsIdDataSet(
              "Fleet-maintained app slug",
              fleetMaintainedAppSlug
            )}
            {renderGitOpsIdDataSet("App Store ID", appStoreId, appStoreUrl)}
            {vppToken && (
              <DataSet
                title={
                  <TooltipWrapper tipContent="This is an organization unit in your Apple Business account, and licenses will be assigned from it.">
                    VPP token
                  </TooltipWrapper>
                }
                value={formatVppTokenLabel(vppToken)}
              />
            )}
            {renderGitOpsIdDataSet("Google Play ID", googlePlayId)}
          </dl>
          {headerPills && (
            <div className={`${baseClass}__header-pills`}>{headerPills}</div>
          )}
        </div>
      </div>
    </>
  );
};

export default SoftwareDetailsSummary;

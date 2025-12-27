/**
software/titles/:id > Top section
software/versions/:id > Top section
software/os/:id > Top section
*/

import React, { useContext } from "react";

import { SingleValue } from "react-select-5";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import { TooltipContent } from "interfaces/dropdownOption";

import { getPathWithQueryParams, QueryParams } from "utilities/url";
import { getGitOpsModeTipContent } from "utilities/helpers";
import paths from "router/paths";
import {
  NO_VERSION_OR_HOST_DATA_SOURCES,
  ROLLING_ARCH_LINUX_VERSIONS,
} from "interfaces/software";

import { AppContext } from "context/app";

import DataSet from "components/DataSet";
import LastUpdatedHostCount from "components/LastUpdatedHostCount";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import TooltipWrapper from "components/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import CustomLink from "components/CustomLink";
import { isSafeImagePreviewUrl } from "pages/SoftwarePage/helpers";
import TooltipWrapperArchLinuxRolling from "components/TooltipWrapperArchLinuxRolling";

import SoftwareIcon from "../../icons/SoftwareIcon";
import OSIcon from "../../icons/OSIcon";

const ACTION_EDIT_APPEARANCE = "edit_appearance";
const ACTION_EDIT_SOFTWARE = "edit_software";
const ACTION_EDIT_CONFIGURATION = "edit_configuration";
const ACTION_EDIT_AUTO_UPDATE_CONFIGURATION = "edit_auto_update_configuration";

const buildActionOptions = (
  gitOpsModeEnabled: boolean | undefined,
  repoURL: string | undefined,
  source: string | undefined,
  androidSoftwareAvailableForInstall: boolean,
  canConfigureAutoUpdate: boolean
): CustomOptionType[] => {
  let disableEditAppearanceTooltipContent: TooltipContent | undefined;
  let disableEditSoftwareTooltipContent: TooltipContent | undefined;
  let disabledEditConfigurationTooltipContent: TooltipContent | undefined;

  if (gitOpsModeEnabled) {
    const gitOpsModeTooltipContent =
      repoURL && getGitOpsModeTipContent(repoURL);

    disableEditAppearanceTooltipContent = gitOpsModeTooltipContent;
    disabledEditConfigurationTooltipContent = gitOpsModeTooltipContent;

    if (source === "vpp_apps") {
      disableEditSoftwareTooltipContent = gitOpsModeTooltipContent;
    }
  }

  const options: CustomOptionType[] = [
    {
      label: "Edit appearance",
      value: ACTION_EDIT_APPEARANCE,
      isDisabled: !!disableEditAppearanceTooltipContent,
      tooltipContent: disableEditAppearanceTooltipContent,
    },
  ];

  // Hides edit software option only for Android installers, as they are currently non-editable
  if (!androidSoftwareAvailableForInstall) {
    options.push({
      label: "Edit software",
      value: ACTION_EDIT_SOFTWARE,
      isDisabled: !!disableEditSoftwareTooltipContent,
      tooltipContent: disableEditSoftwareTooltipContent,
    });
  }

  // Show edit configuration option only for Android installers
  if (androidSoftwareAvailableForInstall) {
    options.push({
      label: "Edit configuration",
      value: ACTION_EDIT_CONFIGURATION,
      isDisabled: !!disabledEditConfigurationTooltipContent,
      tooltipContent: disabledEditConfigurationTooltipContent,
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
  /** undefined unless previewing icon, in which case is string or null */
  /** Displays an edit CTA to edit the software's icon
   * Should only be defined for team view of an installable software */
  onClickEditConfiguration?: () => void;
  onClickEditAutoUpdateConfig?: () => void;
  iconPreviewUrl?: string | null;
  /** timestamp of when icon was last uploaded, used to force refresh of cached icon */
  iconUploadedAt?: string;
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
  onClickEditConfiguration,
  onClickEditAutoUpdateConfig,
  iconPreviewUrl,
  iconUploadedAt,
}: ISoftwareDetailsSummaryProps) => {
  const hostCountPath = getPathWithQueryParams(paths.MANAGE_HOSTS, queryParams);

  const { config } = useContext(AppContext);

  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled;
  const repoURL = config?.gitops.repository_url;
  const isRollingArch = ROLLING_ARCH_LINUX_VERSIONS.includes(displayName);

  const onSelectSoftwareAction = (option: SingleValue<CustomOptionType>) => {
    switch (option?.value) {
      case ACTION_EDIT_APPEARANCE:
        onClickEditAppearance && onClickEditAppearance();
        break;
      case ACTION_EDIT_SOFTWARE:
        onClickEditSoftware && onClickEditSoftware();
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

  // Remove host count for tgz_packages, sh_packages, and ps1_packages only
  // or if viewing details summary from edit icon preview modal
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

  const actionOptions = buildActionOptions(
    gitOpsModeEnabled,
    repoURL,
    source,
    !!onClickEditConfiguration,
    !!onClickEditAutoUpdateConfig
  );

  return (
    <>
      <div className={baseClass}>
        {isOperatingSystem ? (
          <OSIcon name={name} size="xlarge" />
        ) : (
          renderSoftwareIcon()
        )}
        <dl className={`${baseClass}__info`}>
          <div className={`${baseClass}__title-actions`}>
            <h1>
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
                <DropdownWrapper
                  className={`${baseClass}__actions-dropdown`}
                  name="software-actions"
                  onChange={onSelectSoftwareAction}
                  placeholder="Actions"
                  options={actionOptions}
                  variant="button"
                  nowrapMenu
                />
              </div>
            )}
          </div>
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
          </dl>
        </dl>
      </div>
    </>
  );
};

export default SoftwareDetailsSummary;

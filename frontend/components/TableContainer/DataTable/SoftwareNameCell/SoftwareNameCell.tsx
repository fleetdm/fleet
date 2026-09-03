import React from "react";
import { InjectedRouter } from "react-router";

import {
  getSelfServiceTooltip,
  getDisplayedSoftwareName,
} from "pages/SoftwarePage/helpers";

import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";
import { IconNames } from "components/icons";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import LinkCell from "../LinkCell";
import TooltipTruncatedTextCell from "../TooltipTruncatedTextCell";

const baseClass = "software-name-cell";

type InstallType =
  | "manual"
  | "selfService"
  | "automatic"
  | "automaticSelfService";

export type PageContext = "deviceUser" | "hostDetails" | "hostDetailsLibrary";

interface InstallIconTooltip {
  automaticInstallPoliciesCount?: number;
  pageContext?: PageContext;
  isIosOrIpadosApp?: boolean;
  isAndroidPlayStoreApp?: boolean;
  autoUpdateEnabled?: boolean;
  autoUpdateWindowStart?: string;
  autoUpdateWindowEnd?: string;
}

interface InstallIconConfig {
  iconName: IconNames;
  tooltip: (args: InstallIconTooltip) => JSX.Element;
}

const getPolicyTooltip = (count = 0) =>
  count === 1
    ? "A policy triggers install."
    : `${count} policies trigger install.`;

const getAutoUpdateTooltip = (start: string, end: string) =>
  `Auto updates between ${start} and ${end}.`;

const installIconMap: Record<InstallType, InstallIconConfig> = {
  manual: {
    iconName: "install",
    tooltip: ({ pageContext }) => (
      <>
        Software can be installed on the{" "}
        {pageContext === "hostDetails" ? "Library tab" : "Host details page"}.
      </>
    ),
  },
  selfService: {
    iconName: "user",
    tooltip: ({ isIosOrIpadosApp = false, isAndroidPlayStoreApp = false }) =>
      getSelfServiceTooltip(isIosOrIpadosApp, isAndroidPlayStoreApp),
  },
  automatic: {
    iconName: "refresh",
    tooltip: ({
      automaticInstallPoliciesCount = 0,
      autoUpdateEnabled = false,
      autoUpdateWindowStart,
      autoUpdateWindowEnd,
    }) => {
      const showAutoUpdate =
        autoUpdateEnabled && !!autoUpdateWindowStart && !!autoUpdateWindowEnd;
      return (
        <>
          {automaticInstallPoliciesCount > 0 && (
            <>{getPolicyTooltip(automaticInstallPoliciesCount)}</>
          )}
          {automaticInstallPoliciesCount > 0 && showAutoUpdate && <br />}
          {showAutoUpdate && (
            <>
              {getAutoUpdateTooltip(autoUpdateWindowStart, autoUpdateWindowEnd)}
            </>
          )}
        </>
      );
    },
  },
  automaticSelfService: {
    iconName: "automatic-self-service",
    tooltip: ({
      automaticInstallPoliciesCount = 0,
      isIosOrIpadosApp = false,
      isAndroidPlayStoreApp = false,
      autoUpdateEnabled = false,
      autoUpdateWindowStart,
      autoUpdateWindowEnd,
    }) => {
      const showAutoUpdate =
        autoUpdateEnabled && !!autoUpdateWindowStart && !!autoUpdateWindowEnd;
      return (
        <>
          {automaticInstallPoliciesCount > 0 && (
            <>
              {getPolicyTooltip(automaticInstallPoliciesCount)}
              <br />
            </>
          )}
          {showAutoUpdate && (
            <>
              {getAutoUpdateTooltip(autoUpdateWindowStart, autoUpdateWindowEnd)}
              <br />
            </>
          )}
          {getSelfServiceTooltip(isIosOrIpadosApp, isAndroidPlayStoreApp)}
        </>
      );
    },
  },
};

interface IInstallIconWithTooltipProps {
  isSelfService: boolean;
  automaticInstallPoliciesCount?: number;
  pageContext?: PageContext;
  isIosOrIpadosApp: boolean;
  isAndroidPlayStoreApp: boolean;
  autoUpdateEnabled?: boolean;
  autoUpdateWindowStart?: string;
  autoUpdateWindowEnd?: string;
}

// A row can be "automatic" via a policy-triggered install or a VPP scheduled
// auto-update. Either signal promotes the icon to the automatic family so the
// visual language stays consistent with the details page chip.
const getInstallIconType = (
  isSelfService: boolean,
  automaticInstallPoliciesCount = 0,
  autoUpdateEnabled = false
): InstallType => {
  const isAutomatic = automaticInstallPoliciesCount > 0 || autoUpdateEnabled;
  if (isAutomatic) {
    return isSelfService ? "automaticSelfService" : "automatic";
  }
  return isSelfService ? "selfService" : "manual";
};

export const InstallIconWithTooltip = ({
  isSelfService,
  automaticInstallPoliciesCount,
  pageContext,
  isIosOrIpadosApp,
  isAndroidPlayStoreApp,
  autoUpdateEnabled,
  autoUpdateWindowStart,
  autoUpdateWindowEnd,
}: IInstallIconWithTooltipProps) => {
  const iconType = getInstallIconType(
    isSelfService,
    automaticInstallPoliciesCount,
    autoUpdateEnabled
  );

  // Don't show installer icon on host software library page
  if (iconType === "manual" && pageContext === "hostDetailsLibrary") {
    return null;
  }

  const { iconName, tooltip } = installIconMap[iconType];
  const tipContent = tooltip({
    automaticInstallPoliciesCount,
    pageContext,
    isIosOrIpadosApp,
    isAndroidPlayStoreApp,
    autoUpdateEnabled,
    autoUpdateWindowStart,
    autoUpdateWindowEnd,
  });

  return (
    <div className={`${baseClass}__install-icon-with-tooltip`}>
      <TooltipWrapper
        tipContent={tipContent}
        showArrow
        underline={false}
        position="top"
        tipOffset={12}
        fixedPositionStrategy
      >
        <Icon
          name={iconName}
          className={`${baseClass}__install-icon`}
          color="ui-fleet-black-50"
        />
      </TooltipWrapper>
    </div>
  );
};

interface ISoftwareNameCellProps {
  /** Used to key default software icon and name displayed if no display_name */
  name: string;
  /** Overrides name for display */
  display_name?: string;
  source?: string;
  /** pass in a `path` that this cell will link to */
  path?: string;
  router?: InjectedRouter;
  pageContext?: PageContext;
  hasInstaller?: boolean;
  isSelfService?: boolean;
  automaticInstallPoliciesCount?: number;
  /** e.g. custom icons & app_store_app's override default icons with URLs */
  iconUrl?: string | null;
  isIosOrIpadosApp?: boolean;
  isAndroidPlayStoreApp?: boolean;
  /** VPP auto-updates flag from the list response. Promotes the install icon
   * to the automatic (or automatic-self-service) variant and adds a window
   * line to the tooltip. */
  autoUpdateEnabled?: boolean;
  autoUpdateWindowStart?: string;
  autoUpdateWindowEnd?: string;
  /** Only used on Edit icon modal to render a preview of the chosen unsaved icon */
  previewIcon?: JSX.Element;
}

const SoftwareNameCell = ({
  name,
  display_name,
  source,
  path,
  router,
  pageContext,
  hasInstaller = false,
  isSelfService = false,
  automaticInstallPoliciesCount,
  iconUrl,
  isIosOrIpadosApp = false,
  isAndroidPlayStoreApp = false,
  autoUpdateEnabled = false,
  autoUpdateWindowStart,
  autoUpdateWindowEnd,
  previewIcon,
}: ISoftwareNameCellProps) => {
  const softwareDisplayName = getDisplayedSoftwareName(name, display_name);
  const icon = previewIcon || (
    <SoftwareIcon name={name} source={source} url={iconUrl} />
  );
  // My device page > Software fake link as entire row opens a modal
  if (pageContext === "deviceUser" && !isSelfService) {
    return (
      <LinkCell tooltipTruncate prefix={icon} value={softwareDisplayName} />
    );
  }

  // Non-clickable cell if no router/path (e.g. My device page > SelfService)
  if (!router || !path) {
    return (
      <div className={baseClass}>
        <TooltipTruncatedTextCell
          prefix={icon}
          value={softwareDisplayName}
          className="software-name"
        />
      </div>
    );
  }

  const onClickSoftware = (e: React.MouseEvent) => {
    // Allows for button to be clickable in a clickable row
    e.stopPropagation();
    router.push(path);
  };

  return (
    <LinkCell
      className={baseClass}
      path={path}
      tooltipTruncate
      customOnClick={onClickSoftware}
      prefix={icon}
      value={softwareDisplayName}
      suffix={
        hasInstaller ? (
          <InstallIconWithTooltip
            isSelfService={isSelfService}
            automaticInstallPoliciesCount={automaticInstallPoliciesCount}
            pageContext={pageContext}
            isIosOrIpadosApp={isIosOrIpadosApp}
            isAndroidPlayStoreApp={isAndroidPlayStoreApp}
            autoUpdateEnabled={autoUpdateEnabled}
            autoUpdateWindowStart={autoUpdateWindowStart}
            autoUpdateWindowEnd={autoUpdateWindowEnd}
          />
        ) : undefined
      }
    />
  );
};

export default SoftwareNameCell;

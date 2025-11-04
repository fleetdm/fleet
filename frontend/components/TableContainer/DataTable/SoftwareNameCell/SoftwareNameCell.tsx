import React from "react";
import { InjectedRouter } from "react-router";

import {
  SELF_SERVICE_TOOLTIP,
  SELF_SERVICE_ANDROID_PLAY_STORE_TOOLTIP,
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
  isAndroidPlayStoreApp?: boolean;
}

interface InstallIconConfig {
  iconName: IconNames;
  tooltip: ({
    automaticInstallPoliciesCount,
    pageContext,
    isAndroidPlayStoreApp,
  }: InstallIconTooltip) => JSX.Element;
}

const getPolicyTooltip = (count = 0) =>
  count === 1
    ? "A policy triggers install."
    : `${count} policies trigger install.`;

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
    tooltip: (isAndroidPlayStoreApp) =>
      isAndroidPlayStoreApp
        ? SELF_SERVICE_ANDROID_PLAY_STORE_TOOLTIP
        : SELF_SERVICE_TOOLTIP,
  },
  automatic: {
    iconName: "refresh",
    tooltip: ({ automaticInstallPoliciesCount = 0 }) => (
      <>{getPolicyTooltip(automaticInstallPoliciesCount)}</>
    ),
  },
  automaticSelfService: {
    iconName: "automatic-self-service",
    tooltip: ({ automaticInstallPoliciesCount = 0 }) => (
      <>
        {getPolicyTooltip(automaticInstallPoliciesCount)}
        <br />
        End users can reinstall from
        <br />
        <b>Fleet Desktop {">"} Self-service</b>.
      </>
    ),
  },
};

interface IInstallIconWithTooltipProps {
  isSelfService: boolean;
  automaticInstallPoliciesCount?: number;
  pageContext?: PageContext;
  isAndroidPlayStoreApp?: boolean;
}

const getInstallIconType = (
  isSelfService: boolean,
  automaticInstallPoliciesCount = 0
): InstallType => {
  if (automaticInstallPoliciesCount > 0) {
    return isSelfService ? "automaticSelfService" : "automatic";
  }
  return isSelfService ? "selfService" : "manual";
};

const InstallIconWithTooltip = ({
  isSelfService,
  automaticInstallPoliciesCount,
  pageContext,
  isAndroidPlayStoreApp,
}: IInstallIconWithTooltipProps) => {
  const iconType = getInstallIconType(
    isSelfService,
    automaticInstallPoliciesCount
  );

  // Don't show installer icon on host software library page
  if (iconType === "manual" && pageContext === "hostDetailsLibrary") {
    return null;
  }

  const { iconName, tooltip } = installIconMap[iconType];
  const tipContent = tooltip({
    automaticInstallPoliciesCount,
    pageContext,
    isAndroidPlayStoreApp,
  });

  return (
    <div className={`${baseClass}__install-icon-with-tooltip`}>
      <TooltipWrapper
        tipContent={tipContent}
        showArrow
        underline={false}
        position="top"
        tipOffset={12}
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
  name: string;
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
  isAndroidPlayStoreApp?: boolean;
}

const SoftwareNameCell = ({
  name,
  source,
  path,
  router,
  pageContext,
  hasInstaller = false,
  isSelfService = false,
  automaticInstallPoliciesCount,
  iconUrl,
  isAndroidPlayStoreApp = false,
}: ISoftwareNameCellProps) => {
  const icon = <SoftwareIcon name={name} source={source} url={iconUrl} />;
  // My device page > Software fake link as entire row opens a modal
  if (pageContext === "deviceUser" && !isSelfService) {
    return <LinkCell tooltipTruncate prefix={icon} value={name} />;
  }

  // Non-clickable cell if no router/path (e.g. My device page > SelfService)
  if (!router || !path) {
    return (
      <div className={baseClass}>
        <TooltipTruncatedTextCell
          prefix={icon}
          value={name}
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
      value={name}
      suffix={
        hasInstaller ? (
          <InstallIconWithTooltip
            isSelfService={isSelfService}
            automaticInstallPoliciesCount={automaticInstallPoliciesCount}
            pageContext={pageContext}
            isAndroidPlayStoreApp={isAndroidPlayStoreApp}
          />
        ) : undefined
      }
    />
  );
};

export default SoftwareNameCell;

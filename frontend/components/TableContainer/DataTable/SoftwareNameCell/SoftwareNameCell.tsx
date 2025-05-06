import React from "react";
import { InjectedRouter } from "react-router";

import { SELF_SERVICE_TOOLTIP } from "pages/SoftwarePage/helpers";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
import TooltipWrapper from "components/TooltipWrapper";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import LinkCell from "../LinkCell";

const baseClass = "software-name-cell";

type InstallType =
  | "manual"
  | "selfService"
  | "automatic"
  | "automaticSelfService";

interface installIconConfig {
  iconName: IconNames;
  tooltip: (automaticInstallPolicyCount?: number) => JSX.Element;
}

const installIconMap: Record<InstallType, installIconConfig> = {
  manual: {
    iconName: "install",
    tooltip: () => <>Software can be installed on Host details page.</>,
  },
  selfService: {
    iconName: "user",
    tooltip: () => SELF_SERVICE_TOOLTIP,
  },
  automatic: {
    iconName: "refresh",
    tooltip: (count = 0) => (
      <>
        {count === 1
          ? "A policy triggers install."
          : `${count} policies trigger install.`}
      </>
    ),
  },
  automaticSelfService: {
    iconName: "automatic-self-service",
    tooltip: (count = 0) => (
      <>
        {count === 1
          ? "A policy triggers install."
          : `${count} policies trigger install.`}{" "}
        <br /> End users can reinstall from
        <br /> <b>Fleet Desktop {">"} Self-service</b>.
      </>
    ),
  },
};

interface IInstallIconWithTooltipProps {
  isSelfService: boolean;
  installType?: "manual" | "automatic";
  automaticInstallPoliciesCount?: number;
}

const InstallIconWithTooltip = ({
  isSelfService,
  installType,
  automaticInstallPoliciesCount,
}: IInstallIconWithTooltipProps) => {
  let iconType: InstallType = "manual";
  if (installType === "automatic") {
    iconType = isSelfService ? "automaticSelfService" : "automatic";
  } else if (isSelfService) {
    iconType = "selfService";
  }

  return (
    <TooltipWrapper
      tipContent={installIconMap[iconType].tooltip(
        automaticInstallPoliciesCount
      )}
      showArrow
      position="top"
      underline={false}
      tipOffset={8}
    >
      <Icon
        name={installIconMap[iconType].iconName}
        className={`${baseClass}__install-icon`}
        color="ui-fleet-black-50"
      />
    </TooltipWrapper>
  );
};

interface ISoftwareNameCellProps {
  name: string;
  source?: string;
  /** pass in a `path` that this cell will link to */
  path?: string;
  router?: InjectedRouter;
  /** Open details modal onClick */
  myDevicePage?: boolean;
  hasPackage?: boolean;
  isSelfService?: boolean;
  installType?: "manual" | "automatic";
  /** e.g. app_store_app's override default icons with URLs */
  iconUrl?: string;
  automaticInstallPoliciesCount?: number;
}

const SoftwareNameCell = ({
  name,
  source,
  path,
  router,
  myDevicePage = false,
  hasPackage = false,
  isSelfService = false,
  installType,
  iconUrl,
  automaticInstallPoliciesCount,
}: ISoftwareNameCellProps) => {
  // My device page > Software
  if (myDevicePage && !isSelfService) {
    return (
      <LinkCell
        tooltipTruncate
        prefix={<SoftwareIcon name={name} source={source} url={iconUrl} />}
        value={name}
      />
    );
  }

  // NO path or router means it's not clickable. return
  // a non-clickable cell early
  // e.g. My device page > SelfService
  if (!router || !path) {
    return (
      <div className={baseClass}>
        <SoftwareIcon name={name} source={source} url={iconUrl} />
        <span className="software-name">{name}</span>
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
      prefix={<SoftwareIcon name={name} source={source} url={iconUrl} />}
      value={name}
      suffix={
        hasPackage ? (
          <InstallIconWithTooltip
            isSelfService={isSelfService}
            installType={installType}
            automaticInstallPoliciesCount={automaticInstallPoliciesCount}
          />
        ) : undefined
      }
    />
  );
};

export default SoftwareNameCell;

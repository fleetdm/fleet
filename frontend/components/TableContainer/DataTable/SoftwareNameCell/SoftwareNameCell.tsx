import React from "react";
import { InjectedRouter } from "react-router";
import ReactTooltip from "react-tooltip";
import { uniqueId } from "lodash";

import Icon from "components/Icon";
import { IconNames } from "components/icons";
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
  tooltip: JSX.Element;
}

const installIconMap: Record<InstallType, installIconConfig> = {
  manual: {
    iconName: "install",
    tooltip: <>Software can be installed on Host details page.</>,
  },
  selfService: {
    iconName: "user",
    tooltip: (
      <>
        End users can install from <b>Fleet Desktop {">"} Self-service</b>.
      </>
    ),
  },
  automatic: {
    iconName: "refresh",
    tooltip: <>Software will be automatically installed on each host.</>,
  },
  automaticSelfService: {
    iconName: "automatic-self-service",
    tooltip: (
      <>
        Software will be automatically installed on each host. End users can
        reinstall from <b>Fleet Desktop {">"} Self-service</b>.
      </>
    ),
  },
};

interface IInstallIconWithTooltipProps {
  isSelfService: boolean;
  installType?: "manual" | "automatic";
}

const InstallIconWithTooltip = ({
  isSelfService,
  installType,
}: IInstallIconWithTooltipProps) => {
  let iconType: InstallType = "manual";
  if (installType === "automatic") {
    iconType = isSelfService ? "automaticSelfService" : "automatic";
  } else if (isSelfService) {
    iconType = "selfService";
  }

  const tooltipId = uniqueId();
  return (
    <div className={`${baseClass}__install-icon-with-tooltip`}>
      <div
        className={`${baseClass}__install-icon-tooltip`}
        data-tip
        data-for={tooltipId}
      >
        <Icon
          name={installIconMap[iconType].iconName}
          className={`${baseClass}__install-icon`}
          color="ui-fleet-black-50"
        />
      </div>
      <ReactTooltip
        className={`${baseClass}__install-tooltip`}
        place="top"
        effect="solid"
        backgroundColor="#3e4771"
        id={tooltipId}
        data-html
      >
        <span className={`${baseClass}__install-tooltip-text`}>
          {installIconMap[iconType].tooltip}
        </span>
      </ReactTooltip>
    </div>
  );
};

interface ISoftwareNameCellProps {
  name?: string;
  source?: string;
  /** pass in a `path` that this cell will link to */
  path?: string;
  router?: InjectedRouter;
  hasPackage?: boolean;
  isSelfService?: boolean;
  installType?: "manual" | "automatic";
  iconUrl?: string;
}

const SoftwareNameCell = ({
  name,
  source,
  path,
  router,
  hasPackage = false,
  isSelfService = false,
  installType,
  iconUrl,
}: ISoftwareNameCellProps) => {
  // NO path or router means it's not clickable. return
  // a non-clickable cell early
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
      customOnClick={onClickSoftware}
      value={
        <>
          <SoftwareIcon name={name} source={source} url={iconUrl} />
          <span className="software-name">{name}</span>
          {hasPackage && (
            <InstallIconWithTooltip
              isSelfService={isSelfService}
              installType={installType}
            />
          )}
        </>
      }
    />
  );
};

export default SoftwareNameCell;

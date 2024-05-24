import React from "react";
import { InjectedRouter } from "react-router";
import ReactTooltip from "react-tooltip";

import { uniqueId } from "lodash";

import Icon from "components/Icon";

import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";

import LinkCell from "../LinkCell";

const baseClass = "software-name-cell";

const InstallIconWithTooltip = () => {
  const tooltipId = uniqueId();
  return (
    <div className={`${baseClass}__install-icon-with-tooltip`}>
      <div
        className={`${baseClass}__install-icon-tooltip`}
        data-tip
        data-for={tooltipId}
      >
        <Icon name="install" className={`${baseClass}__install-icon`} />
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
          Software can be installed on Host details page.
        </span>
      </ReactTooltip>
    </div>
  );
};

interface ISoftwareNameCellProps {
  name: string;
  source: string;
  path?: string;
  router?: InjectedRouter;
  hasPackage?: boolean;
}

const SoftwareNameCell = ({
  name,
  source,
  path,
  router,
  hasPackage = false,
}: ISoftwareNameCellProps) => {
  // NO path or router means it's not clickable. return
  // a non-clickable cell early
  if (!router || !path) {
    return (
      <div className={baseClass}>
        <SoftwareIcon name={name} source={source} />
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
          <SoftwareIcon name={name} source={source} />
          <span className="software-name">{name}</span>
          {hasPackage && <InstallIconWithTooltip />}
        </>
      }
    />
  );
};

export default SoftwareNameCell;

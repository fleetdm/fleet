import React from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";
import { IFleetMaintainedApp } from "interfaces/software";

import { softwareAlreadyAddedTipContent } from "pages/SoftwarePage/SoftwareAddPage/SoftwareFleetMaintained/FleetMaintainedAppDetailsPage/FleetAppDetailsForm/FleetAppDetailsForm";
import classnames from "classnames";
import Icon from "components/Icon";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";
import TextCell from "../TextCell";

const baseClass = "installer-action-cell";

interface IInstallerActionCellProps {
  value?: Omit<IFleetMaintainedApp, "name" | "version"> | null;
  router?: InjectedRouter;
  className?: string;
  teamId?: number;
}

const InstallerActionCell = ({
  value,
  router,
  className = "w250",
  teamId,
}: IInstallerActionCellProps) => {
  const cellClasses = classnames(baseClass, className);

  // Not all FMAs are supported for all platforms
  if (!value) {
    return (
      <TextCell
        className={cellClasses}
        emptyCellTooltipText="Currently unavailable for this platform."
      />
    );
  }

  const { id, software_title_id: softwareTitleId } = value;

  const onClick = () => {
    const path = getPathWithQueryParams(
      PATHS.SOFTWARE_FLEET_MAINTAINED_DETAILS(id),
      { team_id: teamId }
    );
    if (router && path) {
      router?.push(path);
    }
  };

  if (softwareTitleId) {
    return (
      <div className={cellClasses}>
        <TooltipWrapper
          tipContent={softwareAlreadyAddedTipContent(
            softwareTitleId,
            teamId?.toString()
          )}
          disableTooltip={!softwareTitleId}
          position="top"
          underline={false}
          showArrow
          clickable
          tipOffset={10}
        >
          <Icon name="success" />
        </TooltipWrapper>
      </div>
    );
  }
  return (
    <div className={cellClasses}>
      <Button variant="pill" onClick={onClick}>
        Add
      </Button>
    </div>
  );
};

export default InstallerActionCell;

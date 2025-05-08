import Button from "components/buttons/Button";
import Icon from "components/Icon";
import React from "react";
import { Link } from "react-router";

import PATHS from "router/paths";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import { buildQueryStringFromParams } from "utilities/url";

const baseClass = "config-profile-host-count-cell";

interface IConfigProfileHostCountCellProps {
  teamId: number;
  uuid: string;
  status: string;
  count: number;
  onClickResend: () => void;
}

const ConfigProfileHostCountCell = ({
  teamId,
  uuid,
  status,
  count,
  onClickResend,
}: IConfigProfileHostCountCellProps) => {
  const hostPath = `${PATHS.MANAGE_HOSTS}?${buildQueryStringFromParams({
    team_id: teamId,
    profile_uuid: uuid,
    profile_status: status,
  })}`;

  const renderContent = () => {
    if (count === 0) {
      return <div>{DEFAULT_EMPTY_CELL_VALUE}</div>;
    }
    return (
      <>
        <Link to={hostPath}>{count}</Link>
        <Button onClick={onClickResend} variant="text-icon">
          <Icon name="refresh" color="core-fleet-blue" size="small" />
          <span>Resend</span>
        </Button>
      </>
    );
  };

  return <div className={baseClass}>{renderContent()}</div>;
};

export default ConfigProfileHostCountCell;

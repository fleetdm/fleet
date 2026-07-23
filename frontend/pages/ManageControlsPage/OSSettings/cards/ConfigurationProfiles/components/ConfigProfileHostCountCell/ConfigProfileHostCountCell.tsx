import React from "react";

import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import ViewAllHostsLink from "components/ViewAllHostsLink";

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
  const renderResendButton = () => {
    // we check if the count is 0 or if the uuid starts with "d" which means it
    // is a DDM profile.
    if (count === 0 || uuid[0] === "d" || status !== "failed") {
      return null;
    }

    return (
      <Button
        className={`${baseClass}__resend-button`}
        onClick={onClickResend}
        variant="secondary"
        size="small"
      >
        <Icon name="refresh" color="ui-fleet-black-75" size="small" />
        <span>Resend</span>
      </Button>
    );
  };

  if (count === 0) {
    return <div className={baseClass}>{DEFAULT_EMPTY_CELL_VALUE}</div>;
  }

  return (
    <div className={baseClass}>
      <div>{count}</div>
      <div className={`${baseClass}__actions`}>
        {renderResendButton()}
        <ViewAllHostsLink
          queryParams={{
            fleet_id: teamId,
            profile_uuid: uuid,
            profile_status: status,
          }}
          condensed
          rowHover
        />
      </div>
    </div>
  );
};

export default ConfigProfileHostCountCell;

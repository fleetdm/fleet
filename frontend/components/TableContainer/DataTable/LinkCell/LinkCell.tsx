import React from "react";
import { useDispatch } from "react-redux";
import { push } from "react-router-redux";

import { IHost } from "interfaces/host";
import helpers from "kolide/helpers";
import PATHS from "router/paths";
import Button from "components/buttons/Button/Button";

interface ILinkCellProps {
  value: string;
  host: IHost;
}

const LinkCell = (props: ILinkCellProps): JSX.Element => {
  const { value, host } = props;

  const dispatch = useDispatch();

  const onHostClick = (selectedHost: IHost): void => {
    dispatch(push(PATHS.HOST_DETAILS(selectedHost)));
  };

  const lastSeenTime = (status: string, seenTime: string): string => {
    const { humanHostLastSeen } = helpers;

    if (status !== "online") {
      return `Last Seen: ${humanHostLastSeen(seenTime)} UTC`;
    }

    return "Online";
  };

  return (
    <Button
      onClick={() => onHostClick(host)}
      variant="text-link"
      title={lastSeenTime(host.status, host.seen_time)}
    >
      {value}
    </Button>
  );
};

export default LinkCell;

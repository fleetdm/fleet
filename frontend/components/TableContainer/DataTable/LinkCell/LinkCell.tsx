import React from "react";
import { useDispatch } from "react-redux";
import { push } from "react-router-redux";

import { IHost } from "interfaces/host";
import helpers from "kolide/helpers";
import PATHS from "router/paths";
import Button from "components/buttons/Button/Button";

interface ILinkCellProps<T> {
  value: string;
  data: T;
  path: string;
}

const LinkCell = (props: ILinkCellProps<any>): JSX.Element => {
  const { value, data, path } = props;

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
      onClick={() => onHostClick(data)}
      variant="text-link"
      title={lastSeenTime(data.status, data.seen_time)}
    >
      {value}
    </Button>
  );
};

export default LinkCell;

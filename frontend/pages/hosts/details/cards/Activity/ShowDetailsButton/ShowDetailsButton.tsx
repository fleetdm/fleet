import React from "react";

import { IActivity } from "interfaces/activity";

import Icon from "components/Icon";
import Button from "components/buttons/Button";

import { ShowActivityDetailsHandler } from "../Activity";

const baseClass = "show-details-button";

interface IShowDetailsButtonProps {
  activity: IActivity;
  onShowDetails: ShowActivityDetailsHandler;
}

const ShowDetailsButton = ({
  activity,
  onShowDetails,
}: IShowDetailsButtonProps) => {
  return (
    <Button
      className={baseClass}
      variant="text-link"
      onClick={() => onShowDetails?.(activity)}
    >
      Show details{" "}
      <Icon className={`${baseClass}__show-details-icon`} name="eye" />
    </Button>
  );
};

export default ShowDetailsButton;

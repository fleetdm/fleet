import React from "react";

import { ICommand } from "interfaces/command";

import ActivityItem from "components/ActivityItem";

const baseClass = "command-item";

interface ICommandItemProps {
  command: ICommand;
}

const CommandItem = ({ command }: ICommandItemProps) => {
  const { command_uuid, request_type, status, updated_at } = command;

  let statusVerb = "";
  switch (status) {
    case "Pending":
    case "NotNow":
      statusVerb = "will run";
      break;
    case "Acknowledged":
      statusVerb = "ran";
      break;
    case "Error":
      statusVerb = "failed";
      break;
    default:
      statusVerb = "ran";
  }
  return <div className={baseClass}>The test</div>;

  // for now we will use the ActivityItem component to render command items.
  // We've done this as the command items are styled and function the general same
  // way as activity items. In the future, if we need more customization or
  // specific features for command items, we can create a dedicated CommandItem component.
  // return (
  //   <ActivityItem
  //     className={baseClass}
  //     activity={{
  //       actor_full_name: "",
  //       actor_api_only: false,
  //       actor_gravatar: "",
  //       actor_id: 0,
  //       created_at: updated_at,
  //       fleet_initiated: true,
  //       actor_email: "",
  //       id: command_uuid,
  //       type: "mdm_command",
  //     }}
  //   >
  //     The {request_type} command {statusVerb}.
  //   </ActivityItem>
  // );
};

export default CommandItem;

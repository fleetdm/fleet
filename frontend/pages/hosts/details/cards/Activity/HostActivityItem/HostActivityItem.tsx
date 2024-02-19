import React from "react";
import ReactTooltip from "react-tooltip";
import { formatDistanceToNowStrict } from "date-fns";

import { IActivity } from "interfaces/activity";
import {
  addGravatarUrlToResource,
  internationalTimeFormat,
} from "utilities/helpers";
import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";

import Avatar from "components/Avatar";

import { COLORS } from "styles/var/colors";

import { ShowActivityDetailsHandler } from "../Activity";

const baseClass = "host-activity-item";

interface IHostActivityItemProps {
  activity: IActivity;
  children: React.ReactNode;
  onShowDetails?: ShowActivityDetailsHandler;
}

const HostActivityItem = ({
  activity,
  children,
  onShowDetails,
}: IHostActivityItemProps) => {
  const { actor_email } = activity;
  const { gravatar_url } = actor_email
    ? addGravatarUrlToResource({ email: actor_email })
    : { gravatar_url: DEFAULT_GRAVATAR_LINK };
  const activityCreatedAt = new Date(activity.created_at);

  return (
    <div className={baseClass}>
      <Avatar
        className={`${baseClass}__avatar-image`}
        user={{ gravatar_url }}
        size="small"
        hasWhiteBackground
      />
      <div className={`${baseClass}__details-wrapper`}>
        <div className={"activity-details"}>
          <span className={`${baseClass}__details-topline`}>
            <span>{children}</span>
          </span>
          <br />
          <span
            className={`${baseClass}__details-bottomline`}
            data-tip
            data-for={`activity-${activity.id}`}
          >
            {formatDistanceToNowStrict(activityCreatedAt, {
              addSuffix: true,
            })}
          </span>
          <ReactTooltip
            className="date-tooltip"
            place="top"
            type="dark"
            effect="solid"
            id={`activity-${activity.id}`}
            backgroundColor={COLORS["tooltip-bg"]}
          >
            {internationalTimeFormat(activityCreatedAt)}
          </ReactTooltip>
        </div>
      </div>
      <div className={`${baseClass}__dash`} />
    </div>
  );
};

export default HostActivityItem;

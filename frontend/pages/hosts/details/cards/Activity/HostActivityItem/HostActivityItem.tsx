import React from "react";
import ReactTooltip from "react-tooltip";
import classnames from "classnames";

import { IActivity } from "interfaces/activity";
import {
  addGravatarUrlToResource,
  internationalTimeFormat,
} from "utilities/helpers";
import { DEFAULT_GRAVATAR_LINK } from "utilities/constants";

import Avatar from "components/Avatar";

import { COLORS } from "styles/var/colors";
import { dateAgo } from "utilities/date_format";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import { noop } from "lodash";
import { ShowActivityDetailsHandler } from "../Activity";

const baseClass = "host-activity-item";

interface IHostActivityItemProps {
  activity: IActivity;
  children: React.ReactNode;
  /** Set this to `true` when rendering only this activity by itself. This will
   * change the styles for the activity item for solo rendering.
   * @default false */
  soloActivity?: boolean;
  /** Set this to `true` to hide the close button and prevent from rendering
   * @default false
   */
  hideClose?: boolean;
  className?: string;
  onShowDetails?: ShowActivityDetailsHandler;
  onCancel?: () => void;
}

/**
 * A wrapper that will render all the common elements of a host activity item.
 * This includes the avatar, the created at timestamp, and a dash to separate
 * the activity items. The `children` will be the specific details of the activity
 * implemented in the component that uses this wrapper.
 */
const HostActivityItem = ({
  activity,
  children,
  className,
  soloActivity,
  hideClose = false,
  onShowDetails = noop,
  onCancel = noop,
}: IHostActivityItemProps) => {
  const { actor_email } = activity;
  const { gravatar_url } = actor_email
    ? addGravatarUrlToResource({ email: actor_email })
    : { gravatar_url: DEFAULT_GRAVATAR_LINK };

  // wrapped just in case the date string does not parse correctly
  let activityCreatedAt: Date | null = null;
  try {
    activityCreatedAt = new Date(activity.created_at);
  } catch (e) {
    activityCreatedAt = null;
  }

  const classNames = classnames(baseClass, className, {
    [`${baseClass}__solo-activity`]: soloActivity,
  });

  const onShowActivityDetails = () => {
    onShowDetails({ type: activity.type, details: activity.details });
  };

  const onCancelActivity = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    onCancel();
  };

  return (
    <div className={classNames}>
      <div className={`${baseClass}__avatar-wrapper`}>
        <div className={`${baseClass}__avatar-upper-dash`} />
        <Avatar
          className={`${baseClass}__avatar-image`}
          user={{ gravatar_url }}
          size="small"
          hasWhiteBackground
        />
        <div className={`${baseClass}__avatar-lower-dash`} />
      </div>
      <div
        className={`${baseClass}__details-wrapper`}
        onClick={onShowActivityDetails}
      >
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
            {activityCreatedAt && dateAgo(activityCreatedAt)}
          </span>
          {activityCreatedAt && (
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
          )}
        </div>
        <div className={`${baseClass}__details-actions`}>
          <Button variant="icon" onClick={onShowActivityDetails}>
            <Icon name="info" size="medium" color="ui-fleet-black-75" />
          </Button>
          {!hideClose && (
            <Button variant="icon" onClick={onCancelActivity}>
              <Icon
                name="close"
                color="ui-fleet-black-75"
                className={`${baseClass}__close-icon`}
              />
            </Button>
          )}
        </div>
      </div>
      {/* <div className={`${baseClass}__dash`} /> */}
    </div>
  );
};

export default HostActivityItem;

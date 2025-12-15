import React from "react";
import classnames from "classnames";

import {
  IActivity,
  IActivityDetails,
  IHostPastActivity,
  IHostUpcomingActivity,
} from "interfaces/activity";
import { addGravatarUrlToResource } from "utilities/helpers";

import FeedListItem from "components/FeedListItem";

import { noop } from "lodash";

const baseClass = "activity-item";

export interface IShowActivityDetailsData {
  type: string;
  details?: IActivityDetails;
  created_at?: string;
}

/**
 * A handler that will show the details of an activity. This is used to pass
 * the details of an activity to the parent component to show the details of
 * the activity.
 */
export type ShowActivityDetailsHandler = ({
  type,
  details,
  created_at,
}: IShowActivityDetailsData) => void;

interface IActivityItemProps {
  activity: IActivity | IHostPastActivity | IHostUpcomingActivity;
  children: React.ReactNode;
  /**
   * Set this to `true` when rendering only this activity by itself. This will
   * change the styles for the activity item for solo rendering.
   * @default false */
  isSoloActivity?: boolean;
  /**
   * Set this to `true` to hide the show details button and prevent from rendering.
   * Not all activities can show details, so this is a way to hide the button.
   * @default false
   */
  hideShowDetails?: boolean;
  /**
   * Set this to `true` to hide the close button and prevent from rendering
   * @default false
   */
  hideCancel?: boolean;
  /**
   * Set this to `true` to disable the cancel button. It will still render but
   * will not be clickable.
   * @default false
   */
  disableCancel?: boolean;
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
const ActivityItem = ({
  activity,
  children,
  className,
  isSoloActivity = false,
  hideShowDetails = false,
  hideCancel = false,
  disableCancel = false,
  onShowDetails = noop,
  onCancel = noop,
}: IActivityItemProps) => {
  const { actor_email } = activity;
  const { gravatar_url } = actor_email
    ? addGravatarUrlToResource({ email: actor_email })
    : { gravatar_url: undefined };

  // wrapped just in case the date string does not parse correctly
  let activityCreatedAt: Date;
  try {
    activityCreatedAt = new Date(activity.created_at);
  } catch (e) {
    activityCreatedAt = new Date();
  }

  const classNames = classnames(baseClass, className, {
    [`${baseClass}__solo-activity`]: isSoloActivity,
    [`${baseClass}__no-details`]: hideShowDetails,
  });

  const onShowActivityDetails = (e: React.MouseEvent<HTMLButtonElement>) => {
    // added this stopPropagation as there is some weirdness around the event
    // bubbling up and calling the Modals onEnter handler.
    e.stopPropagation();
    onShowDetails({
      type: activity.type,
      details: activity.details,
      created_at: activity.created_at,
    });
  };

  const onCancelActivity = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    onCancel();
  };

  return (
    <FeedListItem
      useFleetAvatar={activity.fleet_initiated}
      useAPIOnlyAvatar={activity.actor_api_only}
      createdAt={activityCreatedAt}
      gravatarURL={gravatar_url}
      allowShowDetails={!hideShowDetails}
      allowCancel={!hideCancel}
      disableCancel={disableCancel}
      isSoloItem={isSoloActivity}
      onClickFeedItem={onShowActivityDetails}
      onClickCancel={onCancelActivity}
      className={baseClass}
    >
      {children}
    </FeedListItem>
  );

  // return (
  //   <div className={classNames}>
  //     <div className={`${baseClass}__avatar-wrapper`}>
  //       <div className={`${baseClass}__avatar-upper-dash`} />
  //       <Avatar
  //         className={`${baseClass}__avatar-image`}
  //         user={{ gravatar_url }}
  //         size="small"
  //         hasWhiteBackground
  //         useFleetAvatar={activity.fleet_initiated}
  //         useApiOnlyAvatar={activity.actor_api_only}
  //       />
  //       <div className={`${baseClass}__avatar-lower-dash`} />
  //     </div>
  //     <button
  //       disabled={hideShowDetails}
  //       className={`${baseClass}__details-wrapper`}
  //       onClick={onShowActivityDetails}
  //     >
  //       <div className="activity-details">
  //         <span className={`${baseClass}__details-topline`}>
  //           <span>{children}</span>
  //         </span>
  //         <br />
  //         <span
  //           className={`${baseClass}__details-bottomline`}
  //           data-tip
  //           data-for={tooltipId}
  //         >
  //           {activityCreatedAt && dateAgo(activityCreatedAt)}
  //         </span>
  //         {activityCreatedAt && (
  //           <ReactTooltip
  //             className="date-tooltip"
  //             place="top"
  //             type="dark"
  //             effect="solid"
  //             id={tooltipId}
  //             backgroundColor={COLORS["tooltip-bg"]}
  //           >
  //             {internationalTimeFormat(activityCreatedAt)}
  //           </ReactTooltip>
  //         )}
  //       </div>
  //       <div className={`${baseClass}__details-actions`}>
  //         {!hideShowDetails && (
  //           <Button
  //             className={`${baseClass}__action-button`}
  //             variant="icon"
  //             onClick={onShowActivityDetails}
  //           >
  //             <Icon name="info-outline" />
  //           </Button>
  //         )}
  //         {!hideCancel && (
  //           <Button
  //             className={`${baseClass}__action-button`}
  //             variant="icon"
  //             onClick={onCancelActivity}
  //             disabled={disableCancel}
  //           >
  //             <Icon
  //               name="close"
  //               color="ui-fleet-black-75"
  //               className={`${baseClass}__close-icon`}
  //             />
  //           </Button>
  //         )}
  //       </div>
  //     </button>
  //   </div>
  // );
};

export default ActivityItem;

import React, { ReactNode } from "react";
import classnames from "classnames";
import { noop } from "lodash";

import { dateAgo } from "utilities/date_format";
import { internationalTimeFormat } from "utilities/helpers";

import Avatar from "components/Avatar";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "feed-list-item";

interface IFeedListItemProps {
  children: ReactNode;
  useFleetAvatar: boolean;
  createdAt: Date;
  useAPIOnlyAvatar?: boolean;
  gravatarURL?: string;
  allowShowDetails?: boolean;
  allowCancel?: boolean;
  disableCancel?: boolean;
  isSoloItem?: boolean;
  onClickFeedItem?: (e: React.MouseEvent<HTMLButtonElement>) => void;
  onClickCancel?: (e: React.MouseEvent<HTMLButtonElement>) => void;
  className?: string;
}

const FeedListItem = ({
  children,
  useFleetAvatar,
  createdAt,
  gravatarURL,
  useAPIOnlyAvatar = false,
  allowShowDetails = false,
  allowCancel = false,
  isSoloItem = false,
  disableCancel = false,
  className,
  onClickFeedItem = noop,
  onClickCancel = noop,
}: IFeedListItemProps) => {
  const classNames = classnames(baseClass, className, {
    [`${baseClass}__solo-item`]: isSoloItem,
    [`${baseClass}__no-details`]: !allowShowDetails,
  });

  return (
    <div className={classNames}>
      <div className={`${baseClass}__avatar-wrapper`}>
        <div className={`${baseClass}__avatar-upper-dash`} />
        <Avatar
          className={`${baseClass}__avatar-image`}
          user={{ gravatar_url: gravatarURL }}
          size="small"
          hasWhiteBackground
          useFleetAvatar={useFleetAvatar}
          useApiOnlyAvatar={useAPIOnlyAvatar}
        />
        <div className={`${baseClass}__avatar-lower-dash`} />
      </div>
      <button
        disabled={!allowShowDetails}
        className={`${baseClass}__details-wrapper`}
        onClick={onClickFeedItem}
      >
        <div className="feed-details">
          <span className={`${baseClass}__details-topline`}>
            <span>{children}</span>
          </span>
          <br />
          <TooltipWrapper
            className={`${baseClass}__details-bottomline`}
            position="top"
            disableTooltip={!createdAt}
            tipContent={createdAt ? internationalTimeFormat(createdAt) : ""}
            underline={false}
            showArrow
          >
            {createdAt && dateAgo(createdAt)}
          </TooltipWrapper>
        </div>
        <div className={`${baseClass}__details-actions`}>
          {allowShowDetails && (
            <Button
              className={`${baseClass}__action-button`}
              variant="icon"
              onClick={onClickFeedItem}
              ariaLabel="show info"
            >
              <Icon name="info-outline" />
            </Button>
          )}
          {allowCancel && (
            <Button
              className={`${baseClass}__action-button`}
              variant="icon"
              onClick={onClickCancel}
              disabled={disableCancel}
              ariaLabel="cancel action"
            >
              <Icon
                name="close"
                color="ui-fleet-black-75"
                className={`${baseClass}__close-icon`}
              />
            </Button>
          )}
        </div>
      </button>
    </div>
  );
};

export default FeedListItem;

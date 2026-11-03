import React from "react";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import InfoBanner from "components/InfoBanner";
import List from "components/List";
import Spinner from "components/Spinner";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import {
  INotificationAction,
  INotificationItem,
  INotificationView,
  isNotificationItemInstalling,
} from "interfaces/device_notification";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

const baseClass = "device-notification-page";

const renderBoldMarkup = (text: string): React.ReactNode => {
  const parts = text.split(/(\*\*[^*]+\*\*)/g);
  return parts.map((part, i) => {
    if (part.startsWith("**") && part.endsWith("**")) {
      // eslint-disable-next-line react/no-array-index-key
      return <strong key={i}>{part.slice(2, -2)}</strong>;
    }
    // eslint-disable-next-line react/no-array-index-key
    return <React.Fragment key={i}>{part}</React.Fragment>;
  });
};

const renderNotificationItemStatus = (item: INotificationItem) => {
  if (!item.status) {
    return null;
  }
  if (isNotificationItemInstalling(item)) {
    return (
      <span className={`${baseClass}__item-status`}>
        <Spinner size="x-small" centered={false} delay={0} />
        {item.status}
      </span>
    );
  }
  if (item.install_status === "failed_install") {
    return (
      <span className={`${baseClass}__item-status`}>
        <Icon name="error" />
        {item.status}
      </span>
    );
  }
  if (item.install_status === "installed") {
    return (
      <span className={`${baseClass}__item-status`}>
        <Icon name="success" />
        {item.status}
      </span>
    );
  }
  return <span className={`${baseClass}__item-status`}>{item.status}</span>;
};

const renderNotificationItemRow = (item: INotificationItem) => (
  <>
    <span className={`${baseClass}__item-left`}>
      <SoftwareIcon name={item.name} url={item.icon_url} size="small" />
      <span className={`${baseClass}__item-name`}>
        <TooltipTruncatedText
          value={getDisplayedSoftwareName(item.name, item.display_name)}
        />
      </span>
    </span>
    {renderNotificationItemStatus(item)}
  </>
);

interface IDeviceNotificationCardProps {
  view: INotificationView;
  orgLogoURL: string;
  onPostAction: (action: INotificationAction, isPrimary: boolean) => void;
  isPosting?: boolean;
  isPostError?: boolean;
  cardRef?: React.Ref<HTMLDivElement>;
}

const DeviceNotificationCard = ({
  view,
  orgLogoURL,
  onPostAction,
  isPosting = false,
  isPostError = false,
  cardRef,
}: IDeviceNotificationCardProps): JSX.Element => {
  const { actions } = view;
  const primaryAction = actions[actions.length - 1];
  const secondaryAction = actions.length > 1 ? actions[0] : null;

  // When the error banner is visible AND the list has 5+ apps (so it was
  // already scrolling against the 4.5-row cap), shrink the cap by 48px
  // (one row) so the toast window stays stable. 1–4 apps fit without
  // scrolling and just let the window grow by the banner's height —
  // shrinking there would push a fits-fine 4-app list into a scroll
  // transition that reads as a jump.
  const itemCount = view.items.length;
  const shouldShrinkList = isPostError && itemCount >= 5;
  const cardStyle = shouldShrinkList
    ? ({ "--list-max-height-override": `${3.5 * 48}px` } as React.CSSProperties)
    : undefined;

  return (
    <div className={baseClass}>
      <div
        className={`${baseClass}__card`}
        ref={cardRef}
        style={cardStyle}
      >
        <div className={`${baseClass}__header`}>
          <p className={`${baseClass}__title`}>
            {renderBoldMarkup(view.title)}
          </p>
          <p className={`${baseClass}__description`}>
            {renderBoldMarkup(view.description)}
          </p>
        </div>
        <List
          data={view.items}
          idKey="software_title_id"
          renderItemRow={renderNotificationItemRow}
        />
        <div className={`${baseClass}__footer`}>
          <div className={`${baseClass}__actions`}>
            <OrgLogoIcon className={`${baseClass}__logo`} src={orgLogoURL} />
            {secondaryAction && (
              <Button
                variant="subdued"
                size="small"
                disabled={isPosting}
                onClick={() => onPostAction(secondaryAction, false)}
              >
                {secondaryAction.label}
              </Button>
            )}
            {primaryAction && (
              <Button
                size="small"
                disabled={isPosting}
                isLoading={isPosting}
                onClick={() => onPostAction(primaryAction, true)}
              >
                {primaryAction.label}
              </Button>
            )}
          </div>
          {isPostError && (
            <div role="alert">
              <InfoBanner
                color="red"
                size="compact"
                icon="error"
                iconColor="ui-error"
                className={`${baseClass}__error`}
              >
                Something went wrong. Please try again.
              </InfoBanner>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default DeviceNotificationCard;

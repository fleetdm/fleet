import React, { useState, useEffect } from "react";
import { browserHistory } from "react-router";

import Card from "components/Card";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

interface IInfoCardProps {
  title: string;
  titleDetail?: JSX.Element | string | null;
  description?: JSX.Element | string;
  actionUrl?: string;
  children: React.ReactNode;
  action?:
    | {
        type: "link";
        to?: string;
        text: string;
      }
    | {
        type: "button";
        text: string;
        onClick?: () => void;
      };
  total_host_count?: number;
  showTitle?: boolean;
}

const baseClass = "dashboard-info-card";

const useInfoCard = ({
  title,
  titleDetail: defaultTitleDetail,
  description: defaultDescription,
  actionUrl: defaultActionUrl,
  children,
  action,
  total_host_count,
  showTitle = true,
}: IInfoCardProps): JSX.Element => {
  const [actionLink, setActionURL] = useState<string | null>(
    defaultActionUrl || null
  );
  const [titleDetail, setTitleDetail] = useState<JSX.Element | string | null>(
    defaultTitleDetail || null
  );
  const [description, setDescription] = useState<JSX.Element | string | null>(
    defaultDescription || null
  );

  useEffect(() => {
    if (defaultTitleDetail) {
      setTitleDetail(defaultTitleDetail);
    }
  }, [defaultTitleDetail]);

  const renderAction = () => {
    if (action) {
      if (action.type === "button") {
        return (
          <Button
            className={`${baseClass}__action-button`}
            variant="inverse"
            size="small"
            onClick={action.onClick}
          >
            <>
              <span className={`${baseClass}__action-button-text`}>
                {action.text}
              </span>
            </>
          </Button>
        );
      }

      const linkTo = actionLink || action.to;
      if (linkTo) {
        const onClick = (): void => {
          browserHistory.push(linkTo);
        };

        return (
          <Button
            variant="inverse"
            onClick={onClick}
            className={`${baseClass}__action-button`}
            size="small"
          >
            <span className={`${baseClass}__action-button-text`}>
              {action.text}
            </span>
            <Icon name="arrow-internal-link" color="ui-fleet-black-75" />
          </Button>
        );
      }
    }

    return null;
  };

  const clonedChildren = React.Children.toArray(children).map((child) => {
    if (React.isValidElement(child)) {
      child = React.cloneElement(child, {
        setTitleDetail,
        setTitleDescription: setDescription,
        setActionURL,
      });
    }
    return child;
  });

  return (
    <Card className={baseClass} paddingSize="xlarge" borderRadiusSize="large">
      {showTitle && (
        <div>
          <div className={`${baseClass}__section-title-cta`}>
            <div className={`${baseClass}__section-title-group`}>
              <div className={`${baseClass}__section-title`}>
                <h2>{title}</h2>
                {total_host_count !== undefined && (
                  <span>{total_host_count}</span>
                )}
              </div>
              {titleDetail && (
                <div className={`${baseClass}__section-title-detail`}>
                  {titleDetail}
                </div>
              )}
            </div>
            {renderAction()}
          </div>
          {description && (
            <div className={`${baseClass}__section-description`}>
              {description}
            </div>
          )}
        </div>
      )}
      {clonedChildren}
    </Card>
  );
};

export default useInfoCard;

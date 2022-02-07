import React, { useState } from "react";
import { Link } from "react-router";

import Button from "components/buttons/Button";
import LinkArrow from "../../../../../assets/images/icon-arrow-right-vibrant-blue-10x18@2x.png";

interface IInfoCardProps {
  title: string;
  description?: JSX.Element | string;
  children: React.ReactChild | React.ReactChild[];
  action?:
    | {
        type: "link";
        to: string;
        text: string;
      }
    | {
        type: "button";
        text: string;
        onClick?: () => void;
      };
  total_host_count?: string | (() => string | undefined);
  showTitle?: boolean;
}

const baseClass = "homepage-info-card";

const useInfoCard = ({
  title,
  description,
  children,
  action,
  total_host_count,
  showTitle,
}: IInfoCardProps): JSX.Element => {
  const [actionLink, setActionLink] = useState<string | null>(null);
  const [titleDetail, setTitleDetail] = useState<JSX.Element | string | null>(
    null
  );

  const renderAction = () => {
    if (action) {
      if (action.type === "button") {
        return (
          <Button
            className={`${baseClass}__action-button`}
            variant="text-link"
            onClick={action.onClick}
          >
            <>
              <span className={`${baseClass}__action-button-text`}>
                {action.text}
              </span>
              <img src={LinkArrow} alt="link arrow" id="link-arrow" />
            </>
          </Button>
        );
      }

      return (
        <Link
          to={actionLink || action.to}
          className={`${baseClass}__action-button`}
        >
          <span className={`${baseClass}__action-button-text`}>
            {action.text}
          </span>
          <img src={LinkArrow} alt="link arrow" id="link-arrow" />
        </Link>
      );
    }

    return null;
  };

  const clonedChildren = React.Children.toArray(children).map((child) => {
    if (React.isValidElement(child)) {
      child = React.cloneElement(child, {
        setTitleDetail,
        setActionLink,
      });
    }
    return child;
  });

  return (
    <div className={baseClass}>
      {showTitle && (
        <>
          <div className={`${baseClass}__section-title-cta`}>
            <div className={`${baseClass}__section-title-group`}>
              <div className={`${baseClass}__section-title`}>
                <h2>{title}</h2>
                {total_host_count && <span>{total_host_count}</span>}
              </div>
              <div className={`${baseClass}__section-title-detail`}>
                {titleDetail}
              </div>
            </div>
            {renderAction()}
          </div>
          <div className={`${baseClass}__section-description`}>
            {description}
          </div>
        </>
      )}
      {clonedChildren}
    </div>
  );
};

export default useInfoCard;

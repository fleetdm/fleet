import React, { useState } from "react";
import { Link } from "react-router";

import Button from "components/buttons/Button";
import LinkArrow from "../../../../../assets/images/icon-arrow-right-vibrant-blue-10x18@2x.png";

interface IInfoCardProps {
  title: string;
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
  total_host_count?: string;
  isLoadingSoftware?: boolean;
  isLoadingActivityFeed?: boolean;
  showTitle?: boolean;
}

const baseClass = "homepage-info-card";

const InfoCard = ({
  title,
  children,
  action,
  total_host_count,
  isLoadingSoftware,
  isLoadingActivityFeed,
  showTitle,
}: IInfoCardProps) => {
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
              <span>{action.text}</span>
              <img src={LinkArrow} alt="link arrow" id="link-arrow" />
            </>
          </Button>
        );
      }

      return (
        <Link to={action.to} className={`${baseClass}__action-button`}>
          <span>{action.text}</span>
          <img src={LinkArrow} alt="link arrow" id="link-arrow" />
        </Link>
      );
    }

    return null;
  };

  const [subtitle, setSubtitle] = useState<JSX.Element | string | null>("");

  const clonedChildren = React.Children.toArray(children).map((child) => {
    if (React.isValidElement(child)) {
      child = React.cloneElement(child, {
        setSubtitle,
      });
    }
    return child;
  });

  return (
    <div className={baseClass}>
      {showTitle && (
        <div className={`${baseClass}__section-title-cta`}>
          <div className={`${baseClass}__section-title-group`}>
            <div className={`${baseClass}__section-title`}>
              <h2>{title}</h2>
              {total_host_count && <span>{total_host_count}</span>}
            </div>
            <div className={`${baseClass}__section-subtitle`}>{subtitle}</div>
          </div>
          {renderAction()}
        </div>
      )}
      {clonedChildren}
    </div>
  );
};

export default InfoCard;

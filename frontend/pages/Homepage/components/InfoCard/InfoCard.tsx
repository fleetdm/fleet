import React from "react";
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
}

const baseClass = "homepage-info-card";

const InfoCard = ({
  title,
  children,
  action,
  total_host_count,
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

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section-title-cta`}>
        <div className={`${baseClass}__section-title`}>
          <h2>{title}</h2>
          {total_host_count && <span>{total_host_count}</span>}
        </div>
        {renderAction()}
      </div>
      {children}
    </div>
  );
};

export default InfoCard;

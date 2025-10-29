import React from "react";
import { Link } from "react-router";
import classnames from "classnames";
import TooltipTruncatedText from "components/TooltipTruncatedText";

interface ISideNavItemProps {
  title: string;
  path: string;
  isActive: boolean;
}

const baseClass = "side-nav-item";

const SideNavItem = ({ title, path, isActive }: ISideNavItemProps) => {
  const wrapperClasses = classnames(baseClass, {
    [`${baseClass}--active`]: isActive,
  });

  return (
    <li className={wrapperClasses}>
      <Link to={path}>
        <TooltipTruncatedText value={title} tooltipPosition="right" />
      </Link>
    </li>
  );
};

export default SideNavItem;

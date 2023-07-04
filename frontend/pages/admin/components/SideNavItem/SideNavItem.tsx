import React from "react";
import { Link } from "react-router";
import classnames from "classnames";

interface ISideNavItemProps {
  title: string;
  path: string;
  isActive: boolean;
}

const baseClass = "side-nav-item";

const SideNavItem = ({ title, path, isActive }: ISideNavItemProps) => {
  const linkClassnames = classnames(`${baseClass}__nav-link`, {
    "active-nav": isActive,
  });

  return (
    <li className={baseClass}>
      <Link className={linkClassnames} to={path}>
        {title}
      </Link>
    </li>
  );
};

export default SideNavItem;

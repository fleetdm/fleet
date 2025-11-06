import React from "react";
import classnames from "classnames";

import { IAppConfigFormProps } from "pages/admin/OrgSettingsPage/cards/constants";

import SideNavItem from "../SideNavItem";

const baseClass = "side-nav";

export interface ISideNavItem<T> {
  title: string;
  urlSection: string;
  path: string;
  Card: React.ComponentType<T>;
}

interface ISideNavProps<T> {
  navItems: ISideNavItem<T>[];
  activeItem: string;
  CurrentCard: React.ReactNode;
  className?: string;
}

function SideNav<T = IAppConfigFormProps>({
  navItems,
  activeItem,
  CurrentCard,
  className,
}: ISideNavProps<T>) {
  const classes = classnames(baseClass, className);

  return (
    <div className={classes}>
      <div className={`${baseClass}__container`}>
        <nav aria-label="settings">
          <ul className={`${baseClass}__nav-list`}>
            {navItems.map((navItem) => (
              <SideNavItem
                key={navItem.title}
                title={navItem.title}
                path={navItem.path}
                isActive={navItem.urlSection === activeItem}
              />
            ))}
          </ul>
        </nav>
        <div className={`${baseClass}__card-container`}>{CurrentCard}</div>
      </div>
    </div>
  );
}

export default SideNav;

import React from "react";
import classnames from "classnames";

import { IAppConfigFormProps } from "pages/admin/AppSettingsPage/cards/constants";
import Spinner from "components/Spinner";

import SideNavItem from "../SideNavItem";

const baseClass = "side-nav";

export interface ISideNavItem<T> {
  title: string;
  urlSection: string;
  path: string;
  Card: (props: T) => JSX.Element;
}

interface ISideNavProps {
  navItems: ISideNavItem<IAppConfigFormProps>[];
  activeItem: string;
  isLoading: boolean;
  CurrentCard: (passedProps: any) => JSX.Element; // TODO: typing
  className?: string;
}

const SideNav = ({
  navItems,
  activeItem,
  isLoading,
  CurrentCard,
  className,
}: ISideNavProps) => {
  const classes = classnames(baseClass, className);

  return (
    <div className={classes}>
      {isLoading ? (
        <Spinner />
      ) : (
        <div className={`${baseClass}__container`}>
          <nav>
            <ul className={`${baseClass}__nav-list`}>
              {navItems.map((navItem) => (
                <SideNavItem
                  title={navItem.title}
                  path={navItem.path}
                  isActive={navItem.urlSection === activeItem}
                />
              ))}
            </ul>
          </nav>
          <div className={`${baseClass}__card-container`}>
            <CurrentCard />
          </div>
        </div>
      )}
    </div>
  );
};

export default SideNav;

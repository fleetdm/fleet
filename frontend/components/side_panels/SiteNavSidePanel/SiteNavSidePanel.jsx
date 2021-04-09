import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import userInterface from "interfaces/user";
import UserMenu from "components/side_panels/UserMenu";

import navItems from "./navItems";

import HostsIcon from "../../../../assets/images/icon-main-hosts@2x-16x16@2x.png";
import QueriesIcon from "../../../../assets/images/icon-main-queries@2x-16x16@2x.png";
import PacksIcon from "../../../../assets/images/icon-main-packs@2x-16x16@2x.png";
import AdminIcon from "../../../../assets/images/icon-main-settings@2x-16x16@2x.png";

class SiteNavSidePanel extends Component {
  static propTypes = {
    onLogoutUser: PropTypes.func,
    onNavItemClick: PropTypes.func,
    pathname: PropTypes.string,
    user: userInterface,
  };

  constructor(props) {
    super(props);

    const {
      user: { admin },
    } = this.props;

    this.userNavItems = navItems(admin);
  }

  renderNavItem = (navItem) => {
    const { name, iconName } = navItem;
    const { onNavItemClick, pathname } = this.props;
    const active = navItem.location.regex.test(pathname);
    const navItemBaseClass = "site-nav-item";

    const navItemClasses = classnames(`${navItemBaseClass}`, {
      [`${navItemBaseClass}--active`]: active,
    });

    const iconClasses = classnames([`${navItemBaseClass}__icon`]);

    let icon = (
      <img src={HostsIcon} alt={`${iconName} icon`} className={iconClasses} />
    );
    if (iconName === "queries")
      icon = (
        <img
          src={QueriesIcon}
          alt={`${iconName} icon`}
          className={iconClasses}
        />
      );
    else if (iconName === "packs")
      icon = (
        <img src={PacksIcon} alt={`${iconName} icon`} className={iconClasses} />
      );
    else if (iconName === "settings")
      icon = (
        <img src={AdminIcon} alt={`${iconName} icon`} className={iconClasses} />
      );

    return (
      <li className={navItemClasses} key={`nav-item-${name}`}>
        <a
          className={`${navItemBaseClass}__link`}
          onClick={onNavItemClick(navItem.location.pathname)}
        >
          {icon}
          <span className={`${navItemBaseClass}__name`}>{name}</span>
        </a>
      </li>
    );
  };

  renderNavItems = () => {
    const { renderNavItem, userNavItems } = this;
    const { onLogoutUser, user, onNavItemClick } = this.props;
    return (
      <div className="site-nav-container">
        <ul className="site-nav-list">
          {userNavItems.map((navItem) => {
            return renderNavItem(navItem);
          })}
        </ul>
        <UserMenu
          onLogout={onLogoutUser}
          onNavItemClick={onNavItemClick}
          user={user}
        />
      </div>
    );
  };

  render() {
    const { renderNavItems } = this;

    return renderNavItems();
  }
}

export default SiteNavSidePanel;

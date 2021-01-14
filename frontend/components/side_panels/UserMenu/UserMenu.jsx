import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import PATHS from 'router/paths';

import SettingsIcon from '../../../../assets/images/icon-main-settings-white-24x24@2x.png';
import HelpIcon from '../../../../assets/images/icon-main-help-white-24x24@2x.png';
import LogoutIcon from '../../../../assets/images/icon-main-logout-white-24x24@2x.png';

class UserMenu extends Component {
  static propTypes = {
    pathname: PropTypes.string,
    onLogout: PropTypes.func,
    onNavItemClick: PropTypes.func,
    user: PropTypes.shape({
      gravatarURL: PropTypes.string,
      name: PropTypes.string,
      username: PropTypes.string.isRequired,
      position: PropTypes.string,
    }).isRequired,
  };

  static defaultProps = {
    isOpened: false,
  };

  render () {
    const { pathname, onLogout, onNavItemClick } = this.props;

    const baseClass = 'user-menu';
    const userMenuClass = classnames(baseClass);

    let settingsActive;
    if (pathname.replace('/', '') === 'settings') settingsActive = true;
    const settingsNavItemBaseClass = classnames(
      `${baseClass}__nav-item`,
      {
        [`${baseClass}__nav-item--active`]: settingsActive,
      },
    );

    const iconClasses = classnames(
      'icon',
    );

    return (
      <div className={userMenuClass}>
        <nav className={`${baseClass}__nav`}>
          <ul className={`${baseClass}__nav-list`}>
            <li className={settingsNavItemBaseClass}><a href="#settings" onClick={onNavItemClick(PATHS.USER_SETTINGS)}><img src={SettingsIcon} alt="settings icon" className={iconClasses} /><span>Account</span></a></li>
            <li className={`${baseClass}__nav-item`}><a href="https://github.com/fleetdm/fleet/blob/master/docs/README.md" target="_blank" rel="noreferrer"><img src={HelpIcon} alt="help icon" className={iconClasses} /><span>Help</span></a></li>
            <li className={`${baseClass}__nav-item`}><a href="#logout" onClick={onLogout}><img src={LogoutIcon} alt="logout icon" className={iconClasses} /><span>Log out</span></a></li>
          </ul>
        </nav>
      </div>
    );
  }
}

export default UserMenu;

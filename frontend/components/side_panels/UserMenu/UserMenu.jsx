import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Icon from 'components/icons/Icon';
import PATHS from 'router/paths';

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
    const userMenuClass = classnames(
      baseClass,
    );

    let settingsActive;
    if (pathname.replace('/', '') === 'settings') settingsActive = true;
    const settingsNavItemBaseClass = classnames(
      `${baseClass}__nav-item`,
      {
        [`${baseClass}__nav-item--active`]: settingsActive,
      },
    );

    return (
      <div className={userMenuClass}>
        <nav className={`${baseClass}__nav`}>
          <ul className={`${baseClass}__nav-list`}>
            <li className={settingsNavItemBaseClass}><a href="#settings" onClick={onNavItemClick(PATHS.USER_SETTINGS)}><Icon name="user-settings" /><span>Settings</span></a></li>
            <li className={`${baseClass}__nav-item`}><a href="#logout" onClick={onLogout}><Icon name="logout" /><span>Log Out</span></a></li>
          </ul>
        </nav>
      </div>
    );
  }
}

export default UserMenu;

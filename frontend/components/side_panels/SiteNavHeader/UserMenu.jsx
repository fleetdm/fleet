import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Avatar from 'components/Avatar';
import Icon from 'components/icons/Icon';

class UserMenu extends Component {
  static propTypes = {
    isOpened: PropTypes.bool,
    onLogout: PropTypes.func,
    onNavItemClick: PropTypes.func,
    user: PropTypes.shape({
      gravatarURL: PropTypes.string,
      name: PropTypes.string,
      position: PropTypes.string,
    }).isRequired,
  };

  static defaultProps = {
    isOpened: false,
  };

  render () {
    const { isOpened, onLogout, onNavItemClick, user } = this.props;
    const { name, position, username } = user;

    const toggleBaseClass = 'user-menu-toggle';
    const userMenuClass = classnames(
      toggleBaseClass,
      { [`${toggleBaseClass}--open`]: isOpened }
    );

    return (
      <div className={userMenuClass}>
        <Avatar className={`${toggleBaseClass}__avatar`} user={user} />
        <p className={`${toggleBaseClass}__name`}>{ name || username }</p>
        <p className={`${toggleBaseClass}__position`}>{ position || <em>No job title specified</em> }</p>

        <nav className={`${toggleBaseClass}__nav`}>
          <ul className={`${toggleBaseClass}__nav-list`}>
            <li className={`${toggleBaseClass}__nav-item`}><a href="#settings" onClick={onNavItemClick('/settings')}><Icon name="user-settings" /><span>Account Settings</span></a></li>
            <li className={`${toggleBaseClass}__nav-item`}><a href="#logout" onClick={onLogout}><Icon name="logout" /><span>Log Out</span></a></li>
          </ul>
        </nav>
      </div>
    );
  }
}

export default UserMenu;

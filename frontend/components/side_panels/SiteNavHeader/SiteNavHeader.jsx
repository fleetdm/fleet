import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import classnames from 'classnames';

import configInterface from 'interfaces/config';
import { logoutUser } from 'redux/nodes/auth/actions';
import userInterface from 'interfaces/user';
import Icon from 'components/Icon';

import kolideLogo from '../../../../assets/images/kolide-logo.svg';
import UserMenu from './UserMenu';

class SiteNavSidePanel extends Component {
  static propTypes = {
    config: configInterface,
    dispatch: PropTypes.func,
    user: userInterface,
  };

  constructor (props) {
    super(props);

    this.state = {
      userMenuOpened: false,
    };
  }

  componentDidMount = () => {
    const { document } = global;
    const { closeUserMenu } = this;
    document.addEventListener('mousedown', closeUserMenu, false);
  }

  onLogout = (evt) => {
    evt.preventDefault();

    const { dispatch } = this.props;

    dispatch(logoutUser());

    return false;
  }

  closeUserMenu = (evt) => {
    const { headerNav } = this;

    if (headerNav && !headerNav.contains(evt.target)) {
      this.setState({ userMenuOpened: false });
    }
  }

  toggleUserMenu = (evt) => {
    evt.preventDefault();
    const { userMenuOpened } = this.state;

    this.setState({ userMenuOpened: !userMenuOpened });
  }

  render () {
    const {
      config: {
        org_name: orgName,
      },
      user,
    } = this.props;

    const { userMenuOpened } = this.state;
    const { onLogout, toggleUserMenu } = this;
    const { enabled, username } = user;

    const headerBaseClass = 'site-nav-header';

    const headerToggleClass = classnames(
      `${headerBaseClass}__button`,
      'button',
      'button--unstyled',
      { [`${headerBaseClass}__button--open`]: userMenuOpened }
    );

    const userStatusClass = classnames(
      `${headerBaseClass}__user-status`,
      { [`${headerBaseClass}__user-status--enabled`]: enabled }
    );

    return (
      <header className={headerBaseClass}>
        <button className={headerToggleClass} onClick={toggleUserMenu} ref={(r) => { this.headerNav = r; }}>
          <div className={`${headerBaseClass}__org`}>
            <img
              alt="Company logo"
              src={kolideLogo}
              className={`${headerBaseClass}__logo`}
            />
            <h1 className={`${headerBaseClass}__org-name`}>{orgName}</h1>
            <div className={userStatusClass} />
            <h2 className={`${headerBaseClass}__username`}>{username}</h2>
            <Icon name="chevrondown" className={`${headerBaseClass}__org-chevron`} />
          </div>

          <UserMenu
            isOpened={userMenuOpened}
            onLogout={onLogout}
            user={user}
          />
        </button>
      </header>
    );
  }
}

const ConnectedComponent = connect()(SiteNavSidePanel);
export default ConnectedComponent;

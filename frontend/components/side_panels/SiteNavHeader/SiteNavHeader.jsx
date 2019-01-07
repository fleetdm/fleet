import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import configInterface from 'interfaces/config';
import OrgLogoIcon from 'components/icons/OrgLogoIcon';
import Icon from 'components/icons/Icon';
import userInterface from 'interfaces/user';
import UserMenu from 'components/side_panels/SiteNavHeader/UserMenu';

class SiteNavHeader extends Component {
  static propTypes = {
    config: configInterface,
    onLogoutUser: PropTypes.func,
    onNavItemClick: PropTypes.func,
    user: userInterface,
  };

  constructor (props) {
    super(props);

    this.state = { userMenuOpened: false };
  }

  componentDidMount = () => {
    const { document } = global;
    const { closeUserMenu } = this;

    document.addEventListener('mousedown', closeUserMenu, false);
  }

  setHeaderNav = (ref) => {
    this.headerNav = ref;

    return false;
  }

  closeUserMenu = ({ target }) => {
    const { headerNav } = this;

    if (headerNav && !headerNav.contains(target)) {
      this.setState({ userMenuOpened: false });
    }

    return false;
  }

  toggleUserMenu = (evt) => {
    evt.preventDefault();

    const { userMenuOpened } = this.state;

    this.setState({ userMenuOpened: !userMenuOpened });

    return false;
  }

  render () {
    const {
      config: {
        org_name: orgName,
        org_logo_url: orgLogoURL,
      },
      onLogoutUser,
      onNavItemClick,
      user,
    } = this.props;

    const { userMenuOpened } = this.state;
    const { setHeaderNav, toggleUserMenu } = this;
    const { enabled, username } = user;

    const headerBaseClass = 'site-nav-header';

    const headerToggleClass = classnames(
      `${headerBaseClass}__button`,
      'button',
      'button--unstyled'
    );

    const userStatusClass = classnames(
      `${headerBaseClass}__user-status`,
      { [`${headerBaseClass}__user-status--enabled`]: enabled }
    );

    return (
      <header className={headerBaseClass}>
        <div className={headerToggleClass} onClick={toggleUserMenu} ref={setHeaderNav}>
          <div className={`${headerBaseClass}__org`}>
            <OrgLogoIcon className={`${headerBaseClass}__logo`} src={orgLogoURL} />
            <h1 className={`${headerBaseClass}__org-name`}>{orgName}</h1>
            <div className={userStatusClass} />
            <h2 className={`${headerBaseClass}__username`}>{username}</h2>
            <Icon name="downcarat" className={`${headerBaseClass}__carat`} />
          </div>

          <UserMenu
            isOpened={userMenuOpened}
            onLogout={onLogoutUser}
            onNavItemClick={onNavItemClick}
            user={user}
          />
        </div>
      </header>
    );
  }
}

export default SiteNavHeader;

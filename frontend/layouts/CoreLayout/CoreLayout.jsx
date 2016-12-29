import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import { logoutUser } from 'redux/nodes/auth/actions';
import { push } from 'react-router-redux';

import configInterface from 'interfaces/config';
import SiteNavHeader from 'components/side_panels/SiteNavHeader';
import SiteNavSidePanel from 'components/side_panels/SiteNavSidePanel';
import userInterface from 'interfaces/user';

export class CoreLayout extends Component {
  static propTypes = {
    children: PropTypes.node,
    config: configInterface,
    dispatch: PropTypes.func,
    user: userInterface,
  };

  onLogoutUser = () => {
    const { dispatch } = this.props;

    dispatch(logoutUser());

    return false;
  }

  onNavItemClick = (path) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;

      dispatch(push(path));

      return false;
    };
  }

  render () {
    const { children, config, user } = this.props;

    if (!user) return false;

    const { onLogoutUser, onNavItemClick } = this;
    const { pathname } = global.window.location;

    return (
      <div className="app-wrap">
        <nav className="site-nav">
          <SiteNavHeader
            config={config}
            onLogoutUser={onLogoutUser}
            onNavItemClick={onNavItemClick}
            user={user}
          />
          <SiteNavSidePanel
            config={config}
            onNavItemClick={onNavItemClick}
            pathname={pathname}
            user={user}
          />
        </nav>
        <div className="core-wrapper">
          {children}
        </div>
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const {
    app: { config },
    auth: { user },
  } = state;

  return {
    config,
    user,
  };
};

export default connect(mapStateToProps)(CoreLayout);

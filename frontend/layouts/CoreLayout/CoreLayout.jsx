import React, { Component, PropTypes } from 'react';
import { connect } from 'react-redux';
import classnames from 'classnames';
import { logoutUser } from 'redux/nodes/auth/actions';
import { push } from 'react-router-redux';

import configInterface from 'interfaces/config';
import FlashMessage from 'components/FlashMessage';
import { hideFlash } from 'redux/nodes/notifications/actions';
import SiteNavHeader from 'components/side_panels/SiteNavHeader';
import SiteNavSidePanel from 'components/side_panels/SiteNavSidePanel';
import notificationInterface from 'interfaces/notification';
import userInterface from 'interfaces/user';

export class CoreLayout extends Component {
  static propTypes = {
    children: PropTypes.node,
    config: configInterface,
    dispatch: PropTypes.func,
    notifications: notificationInterface,
    showRightSidePanel: PropTypes.bool,
    user: userInterface,
  };

  onLogoutUser = () => {
    const { dispatch } = this.props;

    dispatch(logoutUser());

    return false;
  }

  onNavItemClick = (path) => {
    const { dispatch } = this.props;

    dispatch(push(path));

    return false;
  }

  onRemoveFlash = () => {
    const { dispatch } = this.props;

    dispatch(hideFlash);

    return false;
  }

  onUndoActionClick = (undoAction) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;
      const { onRemoveFlash } = this;

      dispatch(undoAction);

      return onRemoveFlash();
    };
  }

  render () {
    const { children, config, notifications, showRightSidePanel, user } = this.props;
    const wrapperClass = classnames(
      'core-wrapper',
      { 'core-wrapper--show-panel': showRightSidePanel }
    );

    if (!user) return false;

    const { onLogoutUser, onNavItemClick, onRemoveFlash, onUndoActionClick } = this;
    const { pathname } = global.window.location;

    return (
      <div className="app-wrap">
        <nav className="site-nav">
          <SiteNavHeader
            config={config}
            onLogoutUser={onLogoutUser}
            user={user}
          />
          <SiteNavSidePanel
            config={config}
            onNavItemClick={onNavItemClick}
            pathname={pathname}
            user={user}
          />
        </nav>
        <div className={wrapperClass}>
          <FlashMessage
            notification={notifications}
            onRemoveFlash={onRemoveFlash}
            onUndoActionClick={onUndoActionClick}
          />
          {children}
        </div>
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const {
    app: { config, showRightSidePanel },
    auth: { user },
    notifications,
  } = state;

  return {
    config,
    notifications,
    showRightSidePanel,
    user,
  };
};

export default connect(mapStateToProps)(CoreLayout);

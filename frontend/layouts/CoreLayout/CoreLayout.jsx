import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import LoadingBar from "react-redux-loading-bar";
import { logoutUser } from "redux/nodes/auth/actions";
import { push } from "react-router-redux";

import configInterface from "interfaces/config";
import FlashMessage from "components/flash_messages/FlashMessage";
import PersistentFlash from "components/flash_messages/PersistentFlash";
import SiteNavHeader from "components/side_panels/SiteNavHeader";
import SiteNavSidePanel from "components/side_panels/SiteNavSidePanel";
import userInterface from "interfaces/user";
import notificationInterface from "interfaces/notification";
import { hideFlash } from "redux/nodes/notifications/actions";

export class CoreLayout extends Component {
  static propTypes = {
    children: PropTypes.node,
    config: configInterface,
    dispatch: PropTypes.func,
    user: userInterface,
    fullWidthFlash: PropTypes.bool,
    notifications: notificationInterface,
    persistentFlash: PropTypes.shape({
      showFlash: PropTypes.bool.isRequired,
      message: PropTypes.string.isRequired,
    }).isRequired,
  };

  onLogoutUser = () => {
    const { dispatch } = this.props;

    dispatch(logoutUser());

    return false;
  };

  onNavItemClick = (path) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;

      if (path.indexOf("http") !== -1) {
        global.window.open(path, "_blank");

        return false;
      }

      dispatch(push(path));

      return false;
    };
  };

  onRemoveFlash = () => {
    const { dispatch } = this.props;

    dispatch(hideFlash);

    return false;
  };

  onUndoActionClick = (undoAction) => {
    return (evt) => {
      evt.preventDefault();

      const { dispatch } = this.props;
      const { onRemoveFlash } = this;

      dispatch(undoAction);

      return onRemoveFlash();
    };
  };

  render() {
    const {
      fullWidthFlash,
      notifications,
      children,
      config,
      persistentFlash,
      user,
    } = this.props;
    const { onRemoveFlash, onUndoActionClick } = this;

    if (!user) return false;

    const { onLogoutUser, onNavItemClick } = this;
    const { pathname } = global.window.location;

    return (
      <div className="app-wrap">
        <LoadingBar />
        <nav className="site-nav">
          <SiteNavHeader
            config={config}
            onLogoutUser={onLogoutUser}
            onNavItemClick={onNavItemClick}
            user={user}
          />
          <SiteNavSidePanel
            config={config}
            onLogoutUser={onLogoutUser}
            onNavItemClick={onNavItemClick}
            pathname={pathname}
            user={user}
          />
        </nav>
        <div className="core-wrapper">
          {persistentFlash.showFlash && (
            <PersistentFlash message={persistentFlash.message} />
          )}
          <FlashMessage
            fullWidth={fullWidthFlash}
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
    app: { config },
    auth: { user },
    notifications,
    persistentFlash,
  } = state;

  const fullWidthFlash = !user;

  return {
    config,
    fullWidthFlash,
    notifications,
    persistentFlash,
    user,
  };
};

export default connect(mapStateToProps)(CoreLayout);

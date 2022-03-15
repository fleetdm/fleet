import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { push } from "react-router-redux";
import { TableContext } from "context/table";

import { isEqual } from "lodash";

import permissionUtils from "utilities/permissions";
import configInterface from "interfaces/config";
import FleetDesktopTopNav from "components/top_nav/FleetDesktopTopNav";
import userInterface from "interfaces/user";
import notificationInterface from "interfaces/notification";

export class FleetDesktopLayout extends Component {
  static propTypes = {
    children: PropTypes.node,
    config: configInterface,
    dispatch: PropTypes.func,
    user: userInterface,
    fullWidthFlash: PropTypes.bool,
    notifications: notificationInterface,
    isPremiumTier: PropTypes.bool,
  };

  constructor(props) {
    super(props);
  }

  componentWillReceiveProps(nextProps) {
    const { notifications, config } = nextProps;
    const table = this.context;

    // on success of an action, the table will reset its checkboxes.
    // setTimeout is to help with race conditions as table reloads
    // in some instances (i.e. Manage Hosts)
    if (!isEqual(this.props.notifications, notifications)) {
      if (notifications.alertType === "success") {
        setTimeout(() => {
          table.setResetSelectedRows(true);
          setTimeout(() => {
            table.setResetSelectedRows(false);
          }, 300);
        }, 0);
      }
    }
  }

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

  static contextType = TableContext;

  render() {
    const { children, config, user } = this.props;

    if (!user) return false;

    const { onLogoutUser, onNavItemClick } = this;
    const { pathname } = global.window.location;

    return (
      <div className="app-wrap">
        <nav className="site-nav">
          <FleetDesktopTopNav
            config={config}
            onLogoutUser={onLogoutUser}
            onNavItemClick={onNavItemClick}
            pathname={pathname}
            currentUser={user}
          />
        </nav>
        <div className="fleet-desktop-wrapper">{children}</div>
      </div>
    );
  }
}

const mapStateToProps = (state) => {
  const {
    app: { config },
    auth: { user },
    notifications,
  } = state;

  const isPremiumTier = permissionUtils.isPremiumTier(state.app.config);

  const fullWidthFlash = !user;

  return {
    config,
    fullWidthFlash,
    notifications,
    user,
    isPremiumTier,
  };
};

export default connect(mapStateToProps)(FleetDesktopLayout);

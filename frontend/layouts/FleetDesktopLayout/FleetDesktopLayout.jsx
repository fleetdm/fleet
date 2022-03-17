import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { TableContext } from "context/table";

import configInterface from "interfaces/config";
import FleetDesktopTopNav from "components/top_nav/FleetDesktopTopNav";
import userInterface from "interfaces/user";

export class FleetDesktopLayout extends Component {
  static propTypes = {
    children: PropTypes.node,
    config: configInterface,
    dispatch: PropTypes.func,
    user: userInterface,
    fullWidthFlash: PropTypes.bool,
  };

  constructor(props) {
    super(props);
  }

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
  } = state;

  const fullWidthFlash = !user;

  return {
    config,
    fullWidthFlash,
    user,
  };
};

export default connect(mapStateToProps)(FleetDesktopLayout);

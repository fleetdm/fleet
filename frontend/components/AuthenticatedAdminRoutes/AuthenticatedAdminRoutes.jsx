import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { push } from "react-router-redux";
import paths from "router/paths";
import userInterface from "interfaces/user";
import { renderFlash } from "redux/nodes/notifications/actions";

export class AuthenticatedAdminRoutes extends Component {
  static propTypes = {
    children: PropTypes.node,
    dispatch: PropTypes.func,
    user: userInterface,
  };

  componentWillMount() {
    const {
      dispatch,
      user: { global_role },
    } = this.props;
    const { HOME } = paths;

    if (global_role !== "admin") {
      dispatch(push(HOME));
      dispatch(
        renderFlash("error", "You do not have permissions for that page")
      );
    }
    return false;
  }

  render() {
    const { children, user } = this.props;

    if (!user) {
      return false;
    }

    return <>{children}</>;
  }
}

const mapStateToProps = (state) => {
  const { user } = state.auth;

  return { user };
};

export default connect(mapStateToProps)(AuthenticatedAdminRoutes);

import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { push } from "react-router-redux";

import paths from "../../router/paths";
import userInterface from "../../interfaces/user";

export class AuthenticatedAdminRoutes extends Component {
  static propTypes = {
    children: PropTypes.node,
    dispatch: PropTypes.func,
    user: userInterface,
  };

  componentWillMount() {
    const {
      dispatch,
      user: { admin },
    } = this.props;
    const { HOME } = paths;

    if (!admin) {
      dispatch(push(HOME));
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

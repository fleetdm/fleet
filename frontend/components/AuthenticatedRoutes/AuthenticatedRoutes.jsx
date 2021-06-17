import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import { isEqual } from "lodash";
import { push } from "react-router-redux";

import paths from "router/paths";
import redirectLocationInterface from "interfaces/redirect_location";
import { setRedirectLocation } from "redux/nodes/redirectLocation/actions";
import userInterface from "interfaces/user";

export class AuthenticatedRoutes extends Component {
  static propTypes = {
    children: PropTypes.element,
    dispatch: PropTypes.func,
    loading: PropTypes.bool.isRequired,
    locationBeforeTransitions: redirectLocationInterface,
    user: userInterface,
  };

  componentWillMount() {
    const { loading, user } = this.props;
    const {
      redirectToLogin,
      redirectToPasswordReset,
      redirectToApiUserOnly,
    } = this;

    if (!loading && !user) {
      return redirectToLogin();
    }

    if (user && user.force_password_reset) {
      return redirectToPasswordReset();
    }

    if (user && user.api_only) {
      return redirectToApiUserOnly();
    }

    return false;
  }

  componentWillReceiveProps(nextProps) {
    if (isEqual(this.props, nextProps)) return false;

    const { loading, user } = nextProps;
    const {
      redirectToLogin,
      redirectToPasswordReset,
      redirectToApiUserOnly,
    } = this;

    if (!loading && !user) {
      return redirectToLogin();
    }

    if (user && user.force_password_reset) {
      return redirectToPasswordReset();
    }

    if (user && user.api_only) {
      return redirectToApiUserOnly();
    }

    return false;
  }

  redirectToLogin = () => {
    const { dispatch, locationBeforeTransitions } = this.props;
    const { LOGIN } = paths;

    dispatch(setRedirectLocation(locationBeforeTransitions));
    return dispatch(push(LOGIN));
  };

  redirectToPasswordReset = () => {
    const { dispatch } = this.props;
    const { RESET_PASSWORD } = paths;

    return dispatch(push(RESET_PASSWORD));
  };

  redirectToApiUserOnly = () => {
    const { dispatch } = this.props;
    const { API_ONLY_USER } = paths;

    return dispatch(push(API_ONLY_USER));
  };

  render() {
    const { children, user } = this.props;

    if (!user) {
      return false;
    }

    return <div>{children}</div>;
  }
}

const mapStateToProps = (state) => {
  const { loading, user } = state.auth;
  const { locationBeforeTransitions } = state.routing;

  return { loading, locationBeforeTransitions, user };
};

export default connect(mapStateToProps)(AuthenticatedRoutes);

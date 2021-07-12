import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";
import queryString from "query-string";

import { hideBackgroundImage } from "redux/nodes/app/actions";
import { ssoSettings } from "redux/nodes/auth/actions";
import LoginPage from "pages/LoginPage";
import PATHS from "router/paths";

export class LoginRoutes extends Component {
  static propTypes = {
    children: PropTypes.element,
    dispatch: PropTypes.func,
    isResetPassPage: PropTypes.bool,
    isForgotPassPage: PropTypes.bool,
    pathname: PropTypes.string,
    token: PropTypes.string,
  };

  componentWillMount() {
    const { dispatch } = this.props;

    dispatch(ssoSettings()).catch(() => false);

    dispatch(hideBackgroundImage);
  }

  componentWillUnmount() {
    const { dispatch } = this.props;

    dispatch(hideBackgroundImage);
  }

  render() {
    const {
      isResetPassPage,
      isForgotPassPage,
      otherLoginPage,
      pathname,
      token,
    } = this.props;

    return (
      <div className="login-routes">
        {otherLoginPage || (
          <LoginPage
            pathname={pathname}
            token={token}
            isForgotPassPage={isForgotPassPage}
            isResetPassPage={isResetPassPage}
          />
        )}
      </div>
    );
  }
}

const mapStateToProps = (state, ownProps) => {
  const {
    location: { pathname, search },
    children,
  } = ownProps;
  const { token } = queryString.parse(search);

  const isForgotPassPage = pathname.endsWith("/login/forgot");
  const isResetPassPage = pathname.endsWith("/login/reset");

  // Updating to react-router v5 forces a change where we need to look
  // deep within the route's children. It's weird and we should be able to
  // do better on react-router v6. - MP 7/12/21
  const isOtherLoginRoute = (path = pathname) =>
    path.includes(PATHS.LOGIN_INVITE) || path.includes(PATHS.LOGIN_INVITE_SSO);
  const otherLoginRoute =
    (isOtherLoginRoute() &&
      children.props.children.find((route) => isOtherLoginRoute(route.path))) ||
    undefined;

  console.log(otherLoginRoute);
  const otherLoginPage = otherLoginRoute
    ? // ? React.createElement(otherLoginRoute.props.component)
      otherLoginRoute.props.render()
    : undefined;

  return {
    isForgotPassPage,
    isResetPassPage,
    otherLoginPage,
    pathname,
    token,
  };
};

export default connect(mapStateToProps)(LoginRoutes);

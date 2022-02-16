import React, { Component } from "react";
import PropTypes from "prop-types";
import { connect } from "react-redux";

import { hideBackgroundImage } from "redux/nodes/app/actions";
import { ssoSettings } from "redux/nodes/auth/actions";
import LoginPage, { PreviewLoginPage } from "pages/LoginPage";

export class LoginRoutes extends Component {
  static propTypes = {
    children: PropTypes.element,
    dispatch: PropTypes.func,
    isResetPassPage: PropTypes.bool,
    isForgotPassPage: PropTypes.bool,
    isPreviewLoginPage: PropTypes.bool,
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
      children,
      isResetPassPage,
      isForgotPassPage,
      isPreviewLoginPage,
      pathname,
      token,
    } = this.props;

    if (isPreviewLoginPage) {
      return <PreviewLoginPage />;
    }

    return (
      <div className="login-routes">
        {children || (
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
    location: { pathname, query },
  } = ownProps;
  const { token } = query;

  const isForgotPassPage = pathname.endsWith("/login/forgot");
  const isResetPassPage = pathname.endsWith("/login/reset");
  const isPreviewLoginPage = pathname.endsWith("/previewlogin");

  return {
    isForgotPassPage,
    isResetPassPage,
    isPreviewLoginPage,
    pathname,
    token,
  };
};

export default connect(mapStateToProps)(LoginRoutes);

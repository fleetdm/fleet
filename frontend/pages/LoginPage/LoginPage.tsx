import React, { useState, useEffect, useContext } from "react";
import { connect } from "react-redux";
import { size } from "lodash";
import { push } from "react-router-redux";
import { Dispatch } from "redux";

// @ts-ignore
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import {
  clearAuthErrors,
  loginUser,
  ssoRedirect, // @ts-ignore
} from "redux/nodes/auth/actions"; // @ts-ignore
import { clearRedirectLocation } from "redux/nodes/redirectLocation/actions"; // @ts-ignore
import debounce from "utilities/debounce"; // @ts-ignore
import LoginForm from "components/forms/LoginForm"; // @ts-ignore
import LoginSuccessfulPage from "pages/LoginSuccessfulPage"; // @ts-ignore
import ForgotPasswordPage from "pages/ForgotPasswordPage"; // @ts-ignore
import ResetPasswordPage from "pages/ResetPasswordPage";
import paths from "router/paths";
import { IRedirectLocation } from "interfaces/redirect_location";
import { IUser } from "interfaces/user";
import { ISSOSettings } from "interfaces/ssoSettings";
import { AppContext } from "context/app";

interface ILoginPageProps {
  dispatch: Dispatch;
  errors: {
    base: string;
  };
  pathname: string;
  isForgotPassPage: boolean;
  isResetPassPage: boolean;
  token: string;
  redirectLocation: IRedirectLocation;
  user: IUser;
  ssoSettings: ISSOSettings;
}

const LoginPage = ({
  dispatch,
  errors,
  pathname,
  isForgotPassPage,
  isResetPassPage,
  token,
  redirectLocation,
  user,
  ssoSettings,
}: ILoginPageProps) => {
  const { setCurrentUser } = useContext(AppContext);
  const [loginVisible, setLoginVisible] = useState<boolean>(true);

  useEffect(() => {
    const { HOME, LOGIN } = paths;

    if (user && pathname === LOGIN) {
      dispatch(push(HOME));
    }
  }, []);

  const onChange = () => {
    if (size(errors)) {
      return dispatch(clearAuthErrors);
    }

    return false;
  };

  const onSubmit = debounce((formData: any) => {
    const { HOME } = paths;
    const redirectTime = 1500;
    return dispatch(loginUser(formData))
      .then((returnedUser: IUser) => {
        setLoginVisible(false);

        // Redirect to password reset page if user is forced to reset password.
        // Any other requests will fail.
        if (returnedUser.force_password_reset) {
          return dispatch(push(paths.RESET_PASSWORD));
        }

        // transitioning to context API - 9/1/21 MP
        setCurrentUser(returnedUser);

        setTimeout(() => {
          const nextLocation = redirectLocation || HOME;
          dispatch(clearRedirectLocation);
          return dispatch(push(nextLocation));
        }, redirectTime);
      })
      .catch(() => false);
  });

  const ssoSignOn = () => {
    const { HOME } = paths;
    let returnToAfterAuth = HOME;
    if (redirectLocation != null) {
      returnToAfterAuth = redirectLocation.pathname;
    }

    dispatch(ssoRedirect(returnToAfterAuth))
      .then((result: any) => {
        window.location.href = result.payload.ssoRedirectURL;
      })
      .catch(() => false);
  };

  return (
    <AuthenticationFormWrapper>
      <LoginSuccessfulPage />
      <LoginForm
        onChangeFunc={onChange}
        handleSubmit={onSubmit}
        isHidden={!loginVisible}
        serverErrors={errors}
        ssoSettings={ssoSettings}
        handleSSOSignOn={ssoSignOn}
      />
      {isForgotPassPage && <ForgotPasswordPage />}
      {isResetPassPage && <ResetPasswordPage token={token} />}
    </AuthenticationFormWrapper>
  );
};

const mapStateToProps = (state: any) => {
  const { errors, loading, user, ssoSettings } = state.auth;
  const { redirectLocation } = state;

  return {
    errors,
    loading,
    redirectLocation,
    user,
    ssoSettings,
  };
};

export default connect(mapStateToProps)(LoginPage);

import React, { useState, useEffect, useContext } from "react";
import { useDispatch } from "react-redux";
import { push } from "react-router-redux";

import paths from "router/paths";
import { AppContext } from "context/app";
import { IUser } from "interfaces/user"; // @ts-ignore
import { loginUser } from "redux/nodes/auth/actions"; // @ts-ignore
import debounce from "utilities/debounce"; // @ts-ignore

// @ts-ignore
import LoginSuccessfulPage from "pages/LoginSuccessfulPage"; // @ts-ignore
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper"; // @ts-ignore
import LoginForm from "components/forms/LoginForm"; // @ts-ignore

interface ILoginData {
  email: string;
  password: string;
}

const PreviewLoginPage = () => {
  const dispatch = useDispatch();
  const { isPreviewMode, setCurrentUser } = useContext(AppContext);
  const [loginVisible, setLoginVisible] = useState<boolean>(true);

  const onSubmit = debounce((formData: ILoginData) => {
    const { HOME } = paths;
    const redirectTime = 1500;
    return dispatch(loginUser(formData))
      .then((returnedUser: IUser) => {
        setLoginVisible(false);

        // transitioning to context API - 9/1/21 MP
        setCurrentUser(returnedUser);

        setTimeout(() => {
          return dispatch(push(HOME));
        }, redirectTime);
      })
      .catch(() => false);
  });

  useEffect(() => {
    if (isPreviewMode) {
      onSubmit({
        email: "admin@example.com",
        password: "admin123#",
      });
    }
  }, []);

  return (
    <AuthenticationFormWrapper>
      <LoginSuccessfulPage />
      <LoginForm handleSubmit={onSubmit} isHidden={!loginVisible} />
    </AuthenticationFormWrapper>
  );
};

export default PreviewLoginPage;

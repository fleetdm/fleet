import React, { useEffect } from "react";
import { InjectedRouter, browserHistory } from "react-router";

import paths from "router/paths";

import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";

const baseClass = "authentication-nav";

interface IAuthenticationNav {
  previousLocation?: string;
  router?: InjectedRouter;
}

const AuthenticationNav = ({
  previousLocation,
  router,
}: IAuthenticationNav): JSX.Element => {
  useEffect(() => {
    const closeWithEscapeKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && router) {
        router.push(paths.LOGIN);
      }
    };

    document.addEventListener("keydown", closeWithEscapeKey);

    return () => {
      document.removeEventListener("keydown", closeWithEscapeKey);
    };
  }, []);

  const onClick = (): void => {
    if (previousLocation) {
      browserHistory.push(previousLocation);
    } else browserHistory.goBack();
  };

  return (
    <div className={`${baseClass}__back`}>
      <Button
        onClick={onClick}
        className={`${baseClass}__back-link`}
        variant="inverse"
      >
        <Icon name="close" color="core-fleet-black" />
      </Button>
    </div>
  );
};

export default AuthenticationNav;

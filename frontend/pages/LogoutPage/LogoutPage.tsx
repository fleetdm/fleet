import { useContext, useEffect } from "react";
import { InjectedRouter } from "react-router";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import sessionsAPI from "services/entities/sessions";
import authToken from "utilities/auth_token";

interface ILogoutPageProps {
  router: InjectedRouter;
}

const LogoutPage = ({ router }: ILogoutPageProps) => {
  const { isSandboxMode } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

  useEffect(() => {
    const logoutUser = async () => {
      try {
        await sessionsAPI.destroy();
        authToken.remove();
        setTimeout(() => {
          window.location.href = isSandboxMode
            ? "https://www.fleetdm.com/logout"
            : "/";
        }, 500);
      } catch (response) {
        console.error(response);
        router.goBack();
        return renderFlash("error", "Unable to log out of your account");
      }
    };

    logoutUser();
  }, []);

  return null;
};

export default LogoutPage;

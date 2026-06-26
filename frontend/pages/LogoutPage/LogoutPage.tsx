import { useContext, useEffect } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { notify } from "components/ToastNotification";
import sessionsAPI from "services/entities/sessions";
import authToken from "utilities/auth_token";

interface ILogoutPageProps {
  router: InjectedRouter;
}

const LogoutPage = ({ router }: ILogoutPageProps) => {
  const { isSandboxMode } = useContext(AppContext);

  useEffect(() => {
    const logoutUser = async () => {
      try {
        await sessionsAPI.destroy();
        authToken.remove();
        setTimeout(() => {
          window.location.href = isSandboxMode
            ? "https://www.fleetdm.com/logout"
            : PATHS.ROOT;
        }, 500);
      } catch (response) {
        console.error(response);
        router.goBack();
        notify.error("Unable to log out of your account", { response });
      }
    };

    logoutUser();
  }, []);

  return null;
};

export default LogoutPage;

import { useEffect } from "react";
import { InjectedRouter } from "react-router";

import PATHS from "router/paths";
import { notify } from "components/ToastNotification";
import sessionsAPI from "services/entities/sessions";
import authToken from "utilities/auth_token";

interface ILogoutPageProps {
  router: InjectedRouter;
}

const LogoutPage = ({ router }: ILogoutPageProps) => {
  useEffect(() => {
    const logoutUser = async () => {
      try {
        await sessionsAPI.destroy();
        authToken.remove();
        // SPA-navigate, not a reload: on a reload body.dark-mode isn't set
        // until bundle.js runs, so dark-mode users see the viewport flash
        // white during the gap.
        router.replace(PATHS.LOGIN);
      } catch (response) {
        console.error(response);
        router.goBack();
        notify.error("Unable to log out of your account", { response });
      }
    };

    logoutUser();
  }, [router]);

  return null;
};

export default LogoutPage;

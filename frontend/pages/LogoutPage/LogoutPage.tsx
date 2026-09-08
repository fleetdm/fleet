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
        // Prefer SPA navigation over a full reload: a reload paints the
        // viewport white until bundle.js reapplies body.dark-mode, which
        // dark-mode users see as the whole page flashing white.
        router.replace(PATHS.LOGIN);
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

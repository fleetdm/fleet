import { useContext, useEffect } from "react";
import { InjectedRouter } from "react-router";

import { NotificationContext } from "context/notification";
import sessionsAPI from "services/entities/sessions";
import { clearToken } from "utilities/local";

interface ILogoutPageProps {
  router: InjectedRouter;
}

const LogoutPage = ({ router }: ILogoutPageProps) => {
  const { renderFlash } = useContext(NotificationContext);

  useEffect(() => {
    const logoutUser = async () => {
      try {
        await sessionsAPI.destroy();
        clearToken();
        setTimeout(() => {
          window.location.href = "/";
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

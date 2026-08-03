import { useContext } from "react";
import { InjectedRouter } from "react-router";
import { Params } from "react-router/lib/Router";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { notify } from "components/ToastNotification";
import useDeepEffect from "hooks/useDeepEffect";
import usersAPI from "services/entities/users";

interface IEmailTokenRedirectProps {
  router: InjectedRouter; // v3
  params: Params;
}

const EmailTokenRedirect = ({
  router,
  params: { token },
}: IEmailTokenRedirectProps) => {
  const { currentUser } = useContext(AppContext);

  useDeepEffect(() => {
    const confirmEmailChange = async () => {
      if (currentUser && token) {
        try {
          await usersAPI.confirmEmailChange(currentUser, token);
          notify.success("Email updated successfully.");
          router.push(PATHS.ACCOUNT);
        } catch (error) {
          console.log(error);
          router.push(PATHS.LOGIN);
        }
      }
    };

    confirmEmailChange();
  }, [currentUser, token]);

  return null;
};

export default EmailTokenRedirect;

import React, { useContext } from "react";
import { useDispatch, useSelector } from "react-redux";
import classnames from "classnames";

import { QueryClient, QueryClientProvider } from "react-query";

import { authToken } from "utilities/local"; // @ts-ignore
import { useDeepEffect } from "utilities/hooks"; // @ts-ignore
import { fetchCurrentUser } from "redux/nodes/auth/actions"; // @ts-ignore
import { getConfig, getEnrollSecret } from "redux/nodes/app/actions";
import { IUser } from "interfaces/user";
import { IConfig } from "interfaces/config";
import TableProvider from "context/table";
import QueryProvider from "context/query";
import { AppContext } from "context/app";
import { IEnrollSecret } from "interfaces/enroll_secret";

interface IAppProps {
  children: JSX.Element;
}

interface IRootState {
  auth: {
    user: IUser;
  };
}

const App = ({ children }: IAppProps) => {
  const dispatch = useDispatch();
  const user = useSelector((state: IRootState) => state.auth.user);
  const queryClient = new QueryClient();
  const {
    setCurrentUser,
    setConfig,
    setEnrollSecret,
    currentUser,
    isGlobalObserver,
    isOnlyObserver,
    isAnyTeamMaintainer,
    enrollSecret,
  } = useContext(AppContext);

  useDeepEffect(() => {
    // on page refresh
    if (!user && authToken()) {
      dispatch(fetchCurrentUser()).catch(() => false);
    }

    if (user) {
      setCurrentUser(user);
      dispatch(getConfig())
        .then((config: IConfig) => setConfig(config))
        .catch(() => false);
    }
  }, [user]);

  // this puts the enroll secret into redux
  useDeepEffect(() => {
    const canGetEnrollSecret =
      currentUser &&
      typeof isGlobalObserver !== "undefined" &&
      !isGlobalObserver &&
      typeof isOnlyObserver !== "undefined" &&
      !isOnlyObserver &&
      typeof isAnyTeamMaintainer !== "undefined" &&
      !isAnyTeamMaintainer;

    if (canGetEnrollSecret) {
      dispatch(getEnrollSecret())
        .then((secret: IEnrollSecret) => {
          setEnrollSecret(secret);
        })
        .catch(() => false);
    }
  }, [currentUser, isGlobalObserver, isOnlyObserver]);

  const wrapperStyles = classnames("wrapper");
  return (
    <QueryClientProvider client={queryClient}>
      <TableProvider>
        <QueryProvider>
          <div className={wrapperStyles}>{children}</div>
        </QueryProvider>
      </TableProvider>
    </QueryClientProvider>
  );
};

export default App;

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
import FleetErrorBoundary from "pages/errors/FleetErrorBoundary";

interface IAppProps {
  children: JSX.Element;
}

interface ISecretResponse {
  spec: {
    secrets: IEnrollSecret[];
  };
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
    isAtLeastAnyTeamMaintainer,
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

  useDeepEffect(() => {
    const canGetEnrollSecret =
      currentUser &&
      typeof isGlobalObserver !== "undefined" &&
      !isGlobalObserver &&
      typeof isOnlyObserver !== "undefined" &&
      !isOnlyObserver &&
      typeof isAtLeastAnyTeamMaintainer !== "undefined" &&
      !isAtLeastAnyTeamMaintainer;

    if (canGetEnrollSecret) {
      dispatch(getEnrollSecret())
        .then((response: ISecretResponse) => {
          setEnrollSecret(response.spec.secrets);
        })
        .catch(() => false);
    }
  }, [currentUser, isGlobalObserver, isOnlyObserver]);

  const wrapperStyles = classnames("wrapper");
  return (
    <QueryClientProvider client={queryClient}>
      <TableProvider>
        <QueryProvider>
          <FleetErrorBoundary>
            <div className={wrapperStyles}>{children}</div>
          </FleetErrorBoundary>
        </QueryProvider>
      </TableProvider>
    </QueryClientProvider>
  );
};

export default App;

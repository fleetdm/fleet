import React, { useContext, useState } from "react";
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
import PolicyProvider from "context/policy";
import { AppContext } from "context/app";
import { IEnrollSecret } from "interfaces/enroll_secret";
import FleetErrorBoundary from "pages/errors/FleetErrorBoundary";
import Spinner from "components/Spinner";

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

const App = ({ children }: IAppProps): JSX.Element => {
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
    isAnyTeamMaintainerOrTeamAdmin,
  } = useContext(AppContext);

  const [isLoading, setIsLoading] = useState<boolean>(false);

  useDeepEffect(() => {
    // on page refresh
    if (!user && authToken()) {
      dispatch(fetchCurrentUser()).catch(() => false);
    }

    if (user) {
      setIsLoading(true);
      setCurrentUser(user);
      dispatch(getConfig())
        .then((config: IConfig) => {
          setConfig(config);
        })
        .catch(() => false)
        .finally(() => {
          setIsLoading(false);
        });
    }
  }, [user]);

  useDeepEffect(() => {
    const canGetEnrollSecret =
      currentUser &&
      typeof isGlobalObserver !== "undefined" &&
      !isGlobalObserver &&
      typeof isOnlyObserver !== "undefined" &&
      !isOnlyObserver &&
      typeof isAnyTeamMaintainerOrTeamAdmin !== "undefined" &&
      !isAnyTeamMaintainerOrTeamAdmin;

    if (canGetEnrollSecret) {
      dispatch(getEnrollSecret())
        .then((response: ISecretResponse) => {
          setEnrollSecret(response.spec.secrets);
        })
        .catch(() => false);
    }
  }, [currentUser, isGlobalObserver, isOnlyObserver]);

  const wrapperStyles = classnames("wrapper");
  return isLoading ? (
    <Spinner />
  ) : (
    <QueryClientProvider client={queryClient}>
      <TableProvider>
        <QueryProvider>
          <PolicyProvider>
            <FleetErrorBoundary>
              <div className={wrapperStyles}>{children}</div>
            </FleetErrorBoundary>
          </PolicyProvider>
        </QueryProvider>
      </TableProvider>
    </QueryClientProvider>
  );
};

export default App;

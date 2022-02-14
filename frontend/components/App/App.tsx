import React, { useContext, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import classnames from "classnames";
import { AxiosResponse } from "axios";

import { QueryClient, QueryClientProvider } from "react-query";

import { authToken } from "utilities/local"; // @ts-ignore
import { useDeepEffect } from "utilities/hooks"; // @ts-ignore
import { fetchCurrentUser } from "redux/nodes/auth/actions"; // @ts-ignore
import { getConfig, getEnrollSecret } from "redux/nodes/app/actions";
import { IConfig } from "interfaces/config";
import { IEnrollSecret } from "interfaces/enroll_secret";
import { ITeamSummary } from "interfaces/team";
import { IUser } from "interfaces/user";
import TableProvider from "context/table";
import QueryProvider from "context/query";
import PolicyProvider from "context/policy";
import { AppContext } from "context/app";

import { ErrorBoundary } from "react-error-boundary"; // @ts-ignore
import Fleet403 from "pages/errors/Fleet403"; // @ts-ignore
import Fleet404 from "pages/errors/Fleet404"; // @ts-ignore
import Fleet500 from "pages/errors/Fleet500";
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
    available_teams: ITeamSummary[];
  };
}

const App = ({ children }: IAppProps): JSX.Element => {
  const dispatch = useDispatch();
  const user = useSelector((state: IRootState) => state.auth.user);
  const availableTeams = useSelector(
    (state: IRootState) => state.auth.available_teams
  );
  const queryClient = new QueryClient();
  const {
    setAvailableTeams,
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
      // Auth token is not turning to null fast enough so the user is refetched and is making an unneeded API call to enroll_secret
      dispatch(fetchCurrentUser()).catch(() => false);
    }

    if (user) {
      setIsLoading(true);
      setCurrentUser(user);
      setAvailableTeams(availableTeams);
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

  // "any" is used on purpose. We are using Axios but this
  // function expects a native React Error type, which is incompatible.
  const renderErrorOverlay = ({ error }: any) => {
    console.error(error);

    const overlayError = error as AxiosResponse;
    if (overlayError.status === 403) {
      return <Fleet403 />;
    }

    if (overlayError.status === 404) {
      return <Fleet404 />;
    }

    return <Fleet500 />;
  };

  const wrapperStyles = classnames("wrapper");
  return isLoading ? (
    <Spinner />
  ) : (
    <QueryClientProvider client={queryClient}>
      <TableProvider>
        <QueryProvider>
          <PolicyProvider>
            <ErrorBoundary
              fallbackRender={renderErrorOverlay}
              resetKeys={[location.pathname]}
            >
              <div className={wrapperStyles}>{children}</div>
            </ErrorBoundary>
          </PolicyProvider>
        </QueryProvider>
      </TableProvider>
    </QueryClientProvider>
  );
};

export default App;

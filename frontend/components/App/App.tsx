import React, { useContext, useEffect, useState } from "react";
import { AxiosResponse } from "axios";
import { QueryClient, QueryClientProvider } from "react-query";

import page_titles from "router/page_titles";
import TableProvider from "context/table";
import QueryProvider from "context/query";
import PolicyProvider from "context/policy";
import NotificationProvider from "context/notification";
import { AppContext } from "context/app";
import { authToken, clearToken } from "utilities/local";
import useDeepEffect from "hooks/useDeepEffect";

import usersAPI from "services/entities/users";
import configAPI from "services/entities/config";
import hostCountAPI from "services/entities/host_count";

import { ErrorBoundary } from "react-error-boundary";
// @ts-ignore
import Fleet403 from "pages/errors/Fleet403";
// @ts-ignore
import Fleet404 from "pages/errors/Fleet404";
// @ts-ignore
import Fleet500 from "pages/errors/Fleet500";
import Spinner from "components/Spinner";
import { QueryParams } from "utilities/url";
import { randomUUID } from "crypto";
import { uniqueId } from "lodash";

interface IAppProps {
  children: JSX.Element;
  location?: {
    pathname: string;
    search: string;
    hash?: string;
    query: QueryParams;
  };
}

const baseClass = "app";

const App = ({ children, location }: IAppProps): JSX.Element => {
  const queryClient = new QueryClient();
  const {
    config,
    currentUser,
    isGlobalObserver,
    isOnlyObserver,
    isAnyTeamMaintainerOrTeamAdmin,
    setAvailableTeams,
    setCurrentUser,
    setConfig,
    setEnrollSecret,
    setSandboxExpiry,
    setNoSandboxHosts,
  } = useContext(AppContext);

  const curUUID = crypto.randomUUID();

  const [isLoading, setIsLoading] = useState(false);

  // // approach 1 - keep track of, then delete the promise
  // const promises = [];
  // const handleBeforeUnload = (event) => {
  //   // delete all promise objects
  //   promises.forEach(promise => { delete promise; });
  // };

  // useEffect(() => {
  //   window.addEventListener("beforeunload", handleBeforeUnload);
  //   return () => {
  //     window.removeEventListener("beforeUnload", handleBeforeUnload);
  //   };
  // });

  // const foo = async () => {
  //   return Promise.reject("baz");
  // };

  const fetchConfig = async () => {
    try {
      const configResponse = await configAPI.loadAll();
      if (configResponse.sandbox_enabled) {
        const timestamp = await configAPI.loadSandboxExpiry();
        setSandboxExpiry(timestamp as string);
        const hostCount = await hostCountAPI.load({});
        const noSandboxHosts = hostCount.count === 0;
        setNoSandboxHosts(noSandboxHosts);
      }
      setConfig(configResponse);
    } catch (error) {
      console.error(error);
      return false;
    } finally {
      setIsLoading(false);
    }
    return true;
  };

  const fetchCurrentUser = async () => {
    console.log(curUUID, "entered fetchCurrentUser");
    try {
      // throw undefined;
      console.log(curUUID, "fetch me");
      // const bar = await foo();
      // console.log("bar: ", bar);
      // const { user, available_teams } = await usersAPI
      //   .me()
      //   .then((r) => r)
      //   .catch((e) => {
      //     console.log(curUUID, "usersAPI.me error: ", e);
      //     throw e;
      //   });
      const { user, available_teams } = await usersAPI.me();
      console.log(curUUID, "set user");
      setCurrentUser(user);
      console.log(curUUID, "set user available teams");
      setAvailableTeams(user, available_teams);
      console.log(curUUID, "fetch config");
      fetchConfig();
    } catch (error) {
      // if (!location?.pathname.includes("/login/reset") error !== undefined) {
      if (!location?.pathname.includes("/login/reset")) {
        console.log(curUUID, "fetchCurrentUser error:", error);
        clearToken();

        // if this is not the device user page,
        // redirect to login
        if (!location?.pathname.includes("/device/")) {
          window.location.href = "/login";
        }
      }
    }
    return true;
  };

  useEffect(() => {
    console.log(curUUID, "inside useEffect to refetch current user");
    if (authToken() && !location?.pathname.includes("/device/")) {
      console.log("inside 'if' of useEffect to refetch current user");
      fetchCurrentUser();
    }
  }, [location?.pathname]);

  // Updates title that shows up on browser tabs
  useEffect(() => {
    // Also applies title to subpaths such as settings/organization/webaddress
    const curTitle = page_titles.find((item) =>
      location?.pathname.includes(item.path)
    );

    // Override Controls page title if MDM not configured
    if (
      !config?.mdm.enabled_and_configured &&
      curTitle?.path === "/controls/os-updates"
    ) {
      curTitle.title = "Manage OS hosts | Fleet for osquery";
    }

    if (curTitle && curTitle.title) {
      document.title = curTitle.title;
    }
  }, [location, config]);

  useDeepEffect(() => {
    const canGetEnrollSecret =
      currentUser &&
      typeof isGlobalObserver !== "undefined" &&
      !isGlobalObserver &&
      typeof isOnlyObserver !== "undefined" &&
      !isOnlyObserver &&
      typeof isAnyTeamMaintainerOrTeamAdmin !== "undefined" &&
      !isAnyTeamMaintainerOrTeamAdmin &&
      !location?.pathname.includes("/device/");

    const getEnrollSecret = async () => {
      try {
        const { spec } = await configAPI.loadEnrollSecret();
        setEnrollSecret(spec.secrets);
      } catch (error) {
        console.error(error);
        return false;
      }
    };

    if (canGetEnrollSecret) {
      getEnrollSecret();
    }
  }, [currentUser, isGlobalObserver, isOnlyObserver]);

  // "any" is used on purpose. We are using Axios but this
  // function expects a native React Error type, which is incompatible.
  const renderErrorOverlay = ({ error }: any) => {
    // @ts-ignore
    console.error(error);

    const overlayError = error as AxiosResponse;
    if (overlayError.status === 403 || overlayError.status === 402) {
      return <Fleet403 />;
    }

    if (overlayError.status === 404) {
      return <Fleet404 />;
    }

    return <Fleet500 />;
  };

  return isLoading ? (
    <Spinner />
  ) : (
    <QueryClientProvider client={queryClient}>
      <TableProvider>
        <QueryProvider>
          <PolicyProvider>
            <NotificationProvider>
              <ErrorBoundary
                fallbackRender={renderErrorOverlay}
                resetKeys={[location?.pathname]}
              >
                <div className={baseClass}>{children}</div>
              </ErrorBoundary>
            </NotificationProvider>
          </PolicyProvider>
        </QueryProvider>
      </TableProvider>
    </QueryClientProvider>
  );
};

export default App;

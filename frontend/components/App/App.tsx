import React, { FC, useContext, useEffect, useState } from "react";
import { AxiosResponse } from "axios";
import {
  QueryClient,
  QueryClientProvider,
  QueryClientProviderProps,
} from "react-query";

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
import mdmAppleBMAPI from "services/entities/mdm_apple_bm";
import mdmAppleAPI from "services/entities/mdm_apple";

import { ErrorBoundary } from "react-error-boundary";
// @ts-ignore
import Fleet403 from "pages/errors/Fleet403";
// @ts-ignore
import Fleet404 from "pages/errors/Fleet404";
// @ts-ignore
import Fleet500 from "pages/errors/Fleet500";
import Spinner from "components/Spinner";
import { QueryParams } from "utilities/url";

interface IAppProps {
  children: JSX.Element;
  location?: {
    pathname: string;
    search: string;
    hash?: string;
    query: QueryParams;
  };
}

// We create a CustomQueryClientProvider that takes the same props as the original
// QueryClientProvider but adds the children prop as a ReactNode. This children
// prop is not required explicitly in React 18. We do it this way to avoid
// having to update the react-query package version and typings for now.
// When we upgrade React Query we should be able to remove this.
type ICustomQueryClientProviderProps = React.PropsWithChildren<QueryClientProviderProps>;
const CustomQueryClientProvider: FC<ICustomQueryClientProviderProps> = QueryClientProvider;

const baseClass = "app";

const App = ({ children, location }: IAppProps): JSX.Element => {
  const queryClient = new QueryClient();
  const {
    config,
    currentUser,
    isGlobalAdmin,
    isGlobalObserver,
    isOnlyObserver,
    isAnyTeamMaintainerOrTeamAdmin,
    setAvailableTeams,
    setCurrentUser,
    setConfig,
    setEnrollSecret,
    setABMExpiry,
    setAPNsExpiry,
    setVppExpiry,
    setSandboxExpiry,
    setNoSandboxHosts,
  } = useContext(AppContext);

  const [isLoading, setIsLoading] = useState(false);

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

      // These endpoints 403 for non-global admins
      if (isGlobalAdmin) {
        if (configResponse.mdm.apple_bm_enabled_and_configured) {
          // FIXME: we need to catch and check for a 400 status code because the
          // API behaves this way when the token is already expired or invalid.
          //
          // This is a quick fix to not completely break the UI, but it doesn't
          // allow us to show ABM information when the token is expired so it
          // should be fixed upstream.
          try {
            const abmInfo = await mdmAppleBMAPI.getAppleBMInfo();
            setABMExpiry(abmInfo.renew_date);
          } catch (error) {
            console.error(error);
            const abmError = error as AxiosResponse;
            if (abmError.status === 400) {
              const pastDate = "2024-06-03T17:28:44Z";
              setABMExpiry(pastDate);
            }
          }
        }
        if (configResponse.mdm.enabled_and_configured) {
          try {
            const apnsInfo = await mdmAppleAPI.getAppleAPNInfo();
            setAPNsExpiry(apnsInfo.renew_date);
          } catch (error) {
            console.error(error);
          }
          try {
            const vppInfo = await mdmAppleAPI.getVppInfo();
            setVppExpiry(vppInfo.renew_date);
          } catch (e) {
            console.log(e);
          }
        }
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
    try {
      const { user, available_teams } = await usersAPI.me();
      setCurrentUser(user);
      setAvailableTeams(user, available_teams);
      fetchConfig();
    } catch (error) {
      if (
        // reseting a user's password requires the current token
        location?.pathname.includes("/login/reset") ||
        // these errors can occur when user refreshes their page at certain intervals,
        // in which case we don't want to log them out
        (typeof error === "string" &&
          // in Firefox and Chrome, this error is "Request aborted"
          // in Safari, it's "Network Error"
          error.match(/request aborted|network error/i))
      ) {
        return true;
      }
      clearToken();
      // if this is not the device user page,
      // redirect to login
      if (!location?.pathname.includes("/device/")) {
        window.location.href = "/login";
      }
    }
    return true;
  };

  useEffect(() => {
    if (authToken() && !location?.pathname.includes("/device/")) {
      fetchCurrentUser();
    }
  }, [location?.pathname]);

  // Updates title that shows up on browser tabs
  useEffect(() => {
    // Also applies title to subpaths such as settings/organization/webaddress
    // TODO - handle different kinds of paths from PATHS - string, function w/params
    const curTitle = page_titles.find((item) =>
      location?.pathname.includes(item.path)
    );

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
    <CustomQueryClientProvider client={queryClient}>
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
    </CustomQueryClientProvider>
  );
};

export default App;

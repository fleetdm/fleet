import React, { useContext, useEffect, useState } from "react";
import { AxiosError, AxiosResponse } from "axios";
import { useQuery } from "react-query";
import { ErrorBoundary } from "react-error-boundary";
import { isBefore } from "date-fns";

import page_titles from "router/page_titles";
import TableProvider from "context/table";
import QueryProvider from "context/query";
import PolicyProvider from "context/policy";
import NotificationProvider from "context/notification";
import { AppContext } from "context/app";
import { authToken, clearToken } from "utilities/local";
import useDeepEffect from "hooks/useDeepEffect";
import { QueryParams } from "utilities/url";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import usersAPI from "services/entities/users";
import configAPI from "services/entities/config";
import hostCountAPI from "services/entities/host_count";
import mdmAppleBMAPI, {
  IGetAbmTokensResponse,
} from "services/entities/mdm_apple_bm";
import mdmAppleAPI, {
  IGetVppTokensResponse,
} from "services/entities/mdm_apple";

// @ts-ignore
import Fleet403 from "pages/errors/Fleet403";
// @ts-ignore
import Fleet404 from "pages/errors/Fleet404";
// @ts-ignore
import Fleet500 from "pages/errors/Fleet500";

import Spinner from "components/Spinner";

interface IAppProps {
  children: JSX.Element;
  location?: {
    pathname: string;
    search: string;
    hash?: string;
    query: QueryParams;
  };
}

interface RecordWithRenewDate {
  renew_date: string;
}

const GUARANTEED_PAST_DATE = "2000-01-01T01:00:00Z";

// TODO: add tests for this function
const getEarliestExpiry = (records: RecordWithRenewDate[]): string => {
  const earliest = records.reduce((acc, record) => {
    const renewDate = new Date(record.renew_date);
    return isBefore(acc, renewDate) ? acc : renewDate;
  }, new Date(NaN));

  if (isNaN(earliest.valueOf())) {
    // this should never happen assuming the API always returns valid dates, but just in case we'll
    // return a guaranteed past date and log a warning to aid debugging
    console.warn("No valid renew dates found, returning guaranteed past date.");
    return GUARANTEED_PAST_DATE;
  }

  return earliest.toISOString();
};

const baseClass = "app";

const App = ({ children, location }: IAppProps): JSX.Element => {
  const {
    config,
    currentUser,
    isGlobalAdmin,
    isGlobalObserver,
    isOnlyObserver,
    isAnyTeamMaintainerOrTeamAdmin,
    setAvailableTeams,
    setUserSettings,
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

  // We will do a series of API calls to get the data that we need to display
  // warnings to the user about various token expirations.

  // Get the ABM tokens
  useQuery<IGetAbmTokensResponse, AxiosError>(
    ["abm_tokens"],
    () => mdmAppleBMAPI.getTokens(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: !!isGlobalAdmin && !!config?.mdm.enabled_and_configured,
      onSuccess: ({ abm_tokens }) => {
        abm_tokens.length &&
          setABMExpiry({
            earliestExpiry: getEarliestExpiry(abm_tokens),
            needsAbmTermsRenewal: abm_tokens.some(
              (token) => token.terms_expired
            ),
          });
      },
      // TODO: Do we need to catch and check for a 400 status code? The old
      // API behaved this way when the token is already expired or invalid.
      onError: (err) => {
        if (err.status === 400) {
          setABMExpiry({
            earliestExpiry: GUARANTEED_PAST_DATE,
            needsAbmTermsRenewal: true, // TODO: if order of precedence for banners changes, we may need to upate this
          });
        }
      },
    }
  );

  // Get the Apple Push Notification token expiration date
  useQuery(["apns"], () => mdmAppleAPI.getAppleAPNInfo(), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: !!isGlobalAdmin && !!config?.mdm.enabled_and_configured,
    onSuccess: (data) => {
      setAPNsExpiry(data.renew_date);
    },
  });

  // Get the Apple VPP token expiration date
  useQuery<IGetVppTokensResponse>(
    ["vpp_tokens"],
    () => mdmAppleAPI.getVppTokens(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: !!isGlobalAdmin && !!config?.mdm.enabled_and_configured,
      onSuccess: ({ vpp_tokens }) => {
        vpp_tokens.length && setVppExpiry(getEarliestExpiry(vpp_tokens));
      },
    }
  );

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
    try {
      const { user, available_teams, user_settings } = await usersAPI.me();
      setCurrentUser(user);
      setAvailableTeams(user, available_teams);
      setUserSettings(user_settings);
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
  );
};

export default App;

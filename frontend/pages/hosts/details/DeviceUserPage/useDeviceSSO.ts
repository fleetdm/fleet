import { useCallback, useEffect, useState } from "react";

import deviceUserAPI from "services/entities/device_user";

import {
  canAutoInitiateDeviceSSO,
  clearDeviceSSOAttempt,
  isSSORequiredError,
  recordDeviceSSOAttempt,
} from "./helpers";

interface IUseDeviceSSOOptions {
  deviceAuthToken: string;
  /** Errors from the page's device API queries; any one carrying the
   * sso_required marker puts the whole page behind sign-in. */
  errors: unknown[];
  /** The `sso_error` query param the SSO callback redirects back with. */
  ssoErrorParam?: string;
  /** Whether the page was opened to show only the Setup Experience screen. */
  isSetupOnly: boolean;
  /** Whether the host-details call has succeeded, proving the current
   * session works. */
  hasSession: boolean;
}

interface IUseDeviceSSOResult {
  isSSORequired: boolean;
  /** True from the moment an sso_required refusal arrives until the browser
   * leaves for the IdP (or the initiate call fails), so the terminal error
   * never flashes on the way there. */
  isRedirecting: boolean;
  retry: () => void;
}

const useDeviceSSO = ({
  deviceAuthToken,
  errors,
  ssoErrorParam,
  isSetupOnly,
  hasSession,
}: IUseDeviceSSOOptions): IUseDeviceSSOResult => {
  const [isNavigating, setIsNavigating] = useState(false);
  const [canAutoInitiate, setCanAutoInitiate] = useState(() =>
    canAutoInitiateDeviceSSO(deviceAuthToken)
  );

  const isSSORequired = errors.some(isSSORequiredError);

  const redirectToIdP = useCallback(async () => {
    setIsNavigating(true);
    try {
      const { url } = await deviceUserAPI.initiateDeviceSSO(deviceAuthToken);
      window.location.href = url;
    } catch {
      setIsNavigating(false);
    }
  }, [deviceAuthToken]);

  // The callback flags its own failures via ssoErrorParam on the way back
  // here. Initiating again on that signal loops the end user between this
  // page and the IdP, so the remaining attempts have to be theirs to make.
  const autoInitiateAllowed =
    canAutoInitiate &&
    !ssoErrorParam &&
    // The server exempts hosts still in Setup Experience; this keeps an IdP
    // prompt out of Setup Assistant even if that exemption ever misses.
    !isSetupOnly;

  useEffect(() => {
    if (isSSORequired && autoInitiateAllowed && !isNavigating) {
      setCanAutoInitiate(false);
      // An attempt the browser can't remember can't be allowed to run: the
      // one-attempt guard only works if the flag survives the navigation.
      if (recordDeviceSSOAttempt(deviceAuthToken)) {
        redirectToIdP();
      }
    }
  }, [
    isSSORequired,
    autoInitiateAllowed,
    isNavigating,
    deviceAuthToken,
    redirectToIdP,
  ]);

  useEffect(() => {
    // A page with a working session gives the next expiry its own automatic
    // attempt.
    if (hasSession && !isSSORequired) {
      setCanAutoInitiate(clearDeviceSSOAttempt(deviceAuthToken));
    }
  }, [hasSession, isSSORequired, deviceAuthToken]);

  const retry = useCallback(() => {
    setCanAutoInitiate(false);
    recordDeviceSSOAttempt(deviceAuthToken);
    redirectToIdP();
  }, [deviceAuthToken, redirectToIdP]);

  return {
    isSSORequired,
    isRedirecting: isNavigating || (isSSORequired && autoInitiateAllowed),
    retry,
  };
};

export default useDeviceSSO;

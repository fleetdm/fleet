import { useEffect, useState } from "react";
import { useMutation } from "react-query";

import deviceUserAPI, {
  IGetDeviceGoogleConditionalAccessResponse,
} from "services/entities/device_user";

import { replaceLocation } from "./helpers";

interface IUseGoogleConditionalAccessSignInOptions {
  deviceAuthToken: string;
  /** Whether the device details said this host should sign in to Google. */
  isSignInRequired: boolean;
}

interface IUseGoogleConditionalAccessSignInResult {
  isChecking: boolean;
  isRedirecting: boolean;
}

const useGoogleConditionalAccessSignIn = ({
  deviceAuthToken,
  isSignInRequired,
}: IUseGoogleConditionalAccessSignInOptions): IUseGoogleConditionalAccessSignInResult => {
  const { mutate: startSignIn, data, isIdle, isLoading } = useMutation<
    IGetDeviceGoogleConditionalAccessResponse,
    Error,
    string
  >(deviceUserAPI.getGoogleConditionalAccessStatus);

  const shouldStart = !!deviceAuthToken && isSignInRequired;

  // Every call on an unregistered host counts toward the server's retry cap
  // and mints a new single-use token, so it runs once per page load: the
  // mutation is never idle again.
  useEffect(() => {
    if (shouldStart && isIdle) {
      startSignIn(deviceAuthToken);
    }
  }, [shouldStart, isIdle, deviceAuthToken, startSignIn]);

  const [hasReturned, setHasReturned] = useState(false);

  // Absent when the host registered in the meantime or is retry-blocked.
  const signInURL = data?.sign_in_url;

  useEffect(() => {
    if (!signInURL) {
      return undefined;
    }
    replaceLocation(signInURL);

    // The page can outlive the navigation: a full-screen web clip opens the
    // sign-in in a sheet over it, and the browser can restore it from its
    // back/forward cache. Its single-use URL is spent by then, so it shows
    // Self-service and leaves the next sign-in to the next open.
    const onPageShow = (e: PageTransitionEvent) => {
      if (e.persisted) {
        setHasReturned(true);
      }
    };
    const onVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        setHasReturned(true);
      }
    };
    window.addEventListener("pageshow", onPageShow);
    document.addEventListener("visibilitychange", onVisibilityChange);
    return () => {
      window.removeEventListener("pageshow", onPageShow);
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [signInURL]);

  return {
    isChecking: shouldStart && (isIdle || isLoading),
    isRedirecting: !!signInURL && !hasReturned,
  };
};

export default useGoogleConditionalAccessSignIn;

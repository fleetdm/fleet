import React, { useState } from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { WithRouterProps } from "react-router";

import mdmAPI, { IMDMSSOParams } from "services/entities/mdm";

import SSOError from "components/MDM/SSOError";
import Spinner from "components/Spinner/Spinner";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import { IMdmSSOResponse } from "interfaces/mdm";
import AuthenticationFormWrapper from "components/AuthenticationFormWrapper";
import { Params } from "react-router/lib/Router";

const baseClass = "mdm-apple-sso-page";

const DEPSSOLoginPage = ({
  location: { pathname, query },
  params,
}: WithRouterProps<Params, IMDMSSOParams>) => {
  const [clickedLogin, setClickedLogin] = useState(false);
  localStorage.setItem("deviceinfo", query.deviceinfo || "");
  let initiator = query.initiator;
  if (!initiator) {
    if (pathname.startsWith("/mdm/apple/account_driven_enroll/sso")) {
      // While I acknowledge startsWith for route matching is a bit brittle
      // I couldn't find a better way, since the pathname is the actual resolved value and not the placeholder route.
      if (params.token) {
        initiator = `account_driven_enroll:${params.token}`;
      } else {
        initiator = "account_driven_enroll";
      }
    } else {
      initiator = "mdm_sso";
    }
  }
  const { error } = useQuery<IMdmSSOResponse, AxiosError>(
    ["dep_sso"],
    () => mdmAPI.initiateMDMAppleSSO({ ...query, initiator }),
    {
      enabled: clickedLogin || initiator !== "setup_experience",
      retry: false,
      refetchOnWindowFocus: false,
      onSuccess: ({ url }) => {
        window.location.href = url;
      },
    }
  );

  if (initiator === "setup_experience") {
    return (
      <AuthenticationFormWrapper header="Authentication required">
        <div className={`${baseClass} form`}>
          <p>
            Your organization requires you to authenticate before setting up
            your device. Please sign in to continue.
          </p>
          <Button
            className={`${baseClass}__sso-btn`}
            type="button"
            title="Single sign-on"
            onClick={() => setClickedLogin(true)}
            isLoading={clickedLogin}
          >
            <div>Sign in</div>
          </Button>
          <p className={`${baseClass}__transparency-link`}>
            <CustomLink
              text="Why am I seeing this?"
              url="https://fleetdm.com/better"
              newTab
            />
          </p>
        </div>
      </AuthenticationFormWrapper>
    );
  }

  return <div className={baseClass}>{error ? <SSOError /> : <Spinner />}</div>;
};

export default DEPSSOLoginPage;

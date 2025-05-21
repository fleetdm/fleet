import React from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { WithRouterProps } from "react-router";

import mdmAPI, { IMDMSSOParams } from "services/entities/mdm";

import SSOError from "components/MDM/SSOError";
import Spinner from "components/Spinner/Spinner";
import { IMdmSSOReponse } from "interfaces/mdm";

const baseClass = "mdm-apple-sso-page";

const DEPSSOLoginPage = ({
  location: { query },
}: WithRouterProps<object, IMDMSSOParams>) => {
  localStorage.setItem("deviceinfo", query.deviceinfo || "");

  const { error } = useQuery<IMdmSSOReponse, AxiosError>(
    ["dep_sso"],
    () => mdmAPI.initiateMDMAppleSSO(query),
    {
      retry: false,
      refetchOnWindowFocus: false,
      onSuccess: ({ url }) => {
        window.location.href = url;
      },
    }
  );

  return <div className={baseClass}>{error ? <SSOError /> : <Spinner />}</div>;
};

export default DEPSSOLoginPage;

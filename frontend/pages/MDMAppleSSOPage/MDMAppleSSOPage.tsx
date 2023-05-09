import React from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import mdmAPI from "services/entities/mdm";

import SSOError from "components/MDM/SSOError";
import Spinner from "components/Spinner/Spinner";
import { IMdmSSOReponse } from "interfaces/mdm";

const baseClass = "mdm-apple-sso-page";

const DEPSSOLoginPage = () => {
  const { error } = useQuery<void, AxiosError, IMdmSSOReponse>(
    ["dep_sso"],
    () => mdmAPI.initiateMDMAppleSSO(),
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

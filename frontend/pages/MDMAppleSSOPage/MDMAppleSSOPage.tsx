import React from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import mdmAPI from "services/entities/mdm";

import DataError from "components/DataError";
import Spinner from "components/Spinner/Spinner";

const baseClass = "mdm-apple-sso-page";

const SSOError = () => {
  return (
    <DataError className={`${baseClass}__sso-error`}>
      <p>Please contact your IT admin at +1-(415)-651-2575.</p>
    </DataError>
  );
};

const DEPSSOLoginPage = () => {
  const { error } = useQuery<string | void, AxiosError>(
    ["dep_sso"],
    () =>
      mdmAPI.initiateMDMAppleSSO().then(({ url }) => {
        window.location.href = url;
      }),
    {
      retry: false,
      refetchOnWindowFocus: false,
    }
  );

  return <div className={baseClass}>{error ? <SSOError /> : <Spinner />}</div>;
};

export default DEPSSOLoginPage;

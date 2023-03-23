import React from "react";
import { useQuery } from "react-query";
import { AxiosError } from "axios";

import mdmAPI from "services/entities/mdm";

const DEPSSOLoginPage = () => {
  const { error } = useQuery<string | void, AxiosError>(
    ["dep_sso"],
    () =>
      mdmAPI.initiateDEPSSO().then(({ url }) => {
        window.location.href = url;
      }),
    {
      refetchOnWindowFocus: false,
    }
  );

  if (error) {
    return <h1>{error.message}</h1>;
  }

  return <h1>Loading...</h1>;
};

export default DEPSSOLoginPage;

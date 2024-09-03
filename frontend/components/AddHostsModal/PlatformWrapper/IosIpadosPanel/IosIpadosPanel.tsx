import React, { useContext } from "react";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import teamsAPI from "services/entities/teams";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Spinner from "components/Spinner";
import DataError from "components/DataError";
import { IEnrollSecret } from "interfaces/enroll_secret";

const generateUrl = (serverUrl: string, enrollSecrets: IEnrollSecret[]) => {
  if (enrollSecrets.length === 1) {
    return `${serverUrl}/enroll?enroll_secret=${enrollSecrets[0].secret}`;
  }

  enrollSecrets.sort((a, b) => {
    // handle cases where created_at is undefined
    if (a.created_at === undefined && b.created_at === undefined) {
      return 0;
    } else if (a.created_at === undefined) {
      return -1;
    } else if (b.created_at === undefined) {
      return 1;
    }

    return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
  });
};

const baseClass = "ios-ipados-panel";

const IosIpadosPanel = () => {
  const { config, currentTeam } = useContext(AppContext);
  console.log(config);

  const { data: team, isLoading, isError } = useQuery(
    ["team", currentTeam?.id],
    () => teamsAPI.load(currentTeam?.id),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      select: (res) => res.team,
    }
  );

  const helpText =
    "When the end user navigates to this URL, the enrollment profile " +
    "will download in their browser. End users will have to install the profile " +
    "to enroll to Fleet.";

  if (isLoading) {
    return <Spinner className={`${baseClass}__spinner`} centered={false} />;
  }

  if (isError) {
    <DataError />;
  }

  if (!team || !config || !team.secrets) return null;

  const url = generateUrl(config.server_settings.server_url, team.secrets);

  return (
    <div className={baseClass}>
      <InputField
        label="Send this to your end users:"
        enableCopy
        readOnly
        inputWrapperClass
        name="enroll-link"
        value={url}
        helpText={helpText}
      />
    </div>
  );
};

export default IosIpadosPanel;

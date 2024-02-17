import React, { useCallback, useContext, useEffect, useState } from "react";

import { useQuery } from "react-query";

import { NotificationContext } from "context/notification";

import useTeamIdParam from "hooks/useTeamIdParam";

import { IApiError } from "interfaces/errors";
import { IConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";
import { ITeamSubnavProps } from "interfaces/team_subnav";

import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";

import TeamHostExpiryToggle from "./components/TeamHostExpiryToggle";

const baseClass = "team-settings";

const HOST_EXPIRY_ERROR_TEXT = "Host expiry window must be a positive number.";

const TeamSettings = ({ location, router }: ITeamSubnavProps) => {
  const [
    formDataTeamHostExpiryEnabled,
    setFormDataTeamHostExpiryEnabled,
  ] = useState(false); // default false until API response
  const [
    formDataTeamHostExpiryWindow,
    setFormDataTeamHostExpiryWindow,
  ] = useState<number | string>("");
  const [updatingTeamSettings, setUpdatingTeamSettings] = useState(false);
  const [formErrors, setFormErrors] = useState<Record<string, string | null>>(
    {}
  );

  const { renderFlash } = useContext(NotificationContext);

  const { isRouteOk, teamIdForApi } = useTeamIdParam({
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: false,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: false,
      observer: false,
      observer_plus: false,
    },
  });

  const {
    data: appConfig,
    isLoading: isLoadingAppConfig,
    error: errorLoadGlobalConfig,
  } = useQuery<IConfig, Error, IConfig>(
    ["globalConfig"],
    () => configAPI.loadAll(),
    { refetchOnWindowFocus: false }
  );
  const {
    host_expiry_settings: {
      host_expiry_enabled: globalHostExpiryEnabled,
      host_expiry_window: globalHostExpiryWindow,
    },
  } = appConfig ?? { host_expiry_settings: {} };

  const {
    isLoading: isLoadingTeamConfig,
    refetch: refetchTeamConfig,
    error: errorLoadTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["teamConfig", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      enabled: isRouteOk && !!teamIdForApi,
      select: (data) => data.team,
      onSuccess: (teamConfig) => {
        // default this setting to current team setting
        // can be updated by user actions
        setFormDataTeamHostExpiryEnabled(
          teamConfig?.host_expiry_settings?.host_expiry_enabled ?? false
        );
        setFormDataTeamHostExpiryWindow(
          teamConfig.host_expiry_settings?.host_expiry_window ?? ""
        );
      },
      refetchOnWindowFocus: false,
    }
  );

  const validate = useCallback(() => {
    const errors: Record<string, string> = {};
    const numHostExpiryWindow = Number(formDataTeamHostExpiryWindow);
    if (
      // with no global setting, team window can't be empty if enabled
      (!globalHostExpiryEnabled &&
        formDataTeamHostExpiryEnabled &&
        !numHostExpiryWindow) ||
      // if nonempty, must be a positive number
      isNaN(numHostExpiryWindow) ||
      // if overriding a global setting, can be empty to disable local setting
      numHostExpiryWindow < 0
    ) {
      errors.host_expiry_window = HOST_EXPIRY_ERROR_TEXT;
    }

    setFormErrors(errors);
  }, [
    formDataTeamHostExpiryEnabled,
    formDataTeamHostExpiryWindow,
    globalHostExpiryEnabled,
  ]);

  useEffect(() => {
    validate();
  }, [formDataTeamHostExpiryEnabled, formDataTeamHostExpiryWindow, validate]);

  const updateTeamHostExpiry = useCallback(
    (evt: React.MouseEvent<HTMLFormElement>) => {
      evt.preventDefault();
      setUpdatingTeamSettings(true);
      const castedHostExpiryWindow = Number(formDataTeamHostExpiryWindow);
      let enableHostExpiry;
      if (globalHostExpiryEnabled) {
        if (!castedHostExpiryWindow) {
          enableHostExpiry = false;
        } else {
          enableHostExpiry = formDataTeamHostExpiryEnabled;
        }
      } else {
        enableHostExpiry = formDataTeamHostExpiryEnabled;
      }
      teamsAPI
        .update(
          {
            host_expiry_settings: {
              host_expiry_enabled: enableHostExpiry,
              host_expiry_window: castedHostExpiryWindow,
            },
          },
          teamIdForApi
        )
        .then(() => {
          renderFlash("success", "Successfully updated settings.");
          refetchTeamConfig();
        })
        .catch((errorResponse: { data: IApiError }) => {
          renderFlash(
            "error",
            `Could not update team settings. ${errorResponse.data.errors[0].reason}`
          );
        })
        .finally(() => {
          setUpdatingTeamSettings(false);
        });
    },
    [
      formDataTeamHostExpiryEnabled,
      formDataTeamHostExpiryWindow,
      globalHostExpiryEnabled,
      refetchTeamConfig,
      renderFlash,
      teamIdForApi,
    ]
  );

  const renderForm = () => {
    if (errorLoadGlobalConfig || errorLoadTeamConfig) {
      return <DataError />;
    }
    if (isLoadingTeamConfig || isLoadingAppConfig) {
      return <Spinner />;
    }
    return (
      <form onSubmit={updateTeamHostExpiry}>
        {globalHostExpiryEnabled !== undefined && (
          <TeamHostExpiryToggle
            globalHostExpiryEnabled={globalHostExpiryEnabled}
            globalHostExpiryWindow={globalHostExpiryWindow}
            teamExpiryEnabled={formDataTeamHostExpiryEnabled}
            setTeamExpiryEnabled={setFormDataTeamHostExpiryEnabled}
          />
        )}
        {formDataTeamHostExpiryEnabled && (
          <InputField
            label="Host expiry window"
            // type="text" allows `validate` to differentiate between
            // non-numerical input and an empty input
            type="text"
            onChange={setFormDataTeamHostExpiryWindow}
            name="host-expiry-window"
            value={formDataTeamHostExpiryWindow}
            error={formErrors.host_expiry_window}
          />
        )}
        <Button
          type="submit"
          variant="brand"
          className="button-wrap"
          isLoading={updatingTeamSettings}
          disabled={Object.keys(formErrors).length > 0}
        >
          Save
        </Button>
      </form>
    );
  };

  return (
    <section className={`${baseClass}`}>
      <SectionHeader title="Host expiry settings" />
      {renderForm()}
    </section>
  );
};
export default TeamSettings;

import React, { useCallback, useContext, useEffect, useState } from "react";

import { useQuery } from "react-query";

import { NotificationContext } from "context/notification";

import useTeamIdParam from "hooks/useTeamIdParam";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  HOST_STATUS_WEBHOOK_HOST_PERCENTAGE_DROPDOWN_OPTIONS,
  HOST_STATUS_WEBHOOK_WINDOW_DROPDOWN_OPTIONS,
} from "utilities/constants";

import { IApiError } from "interfaces/errors";
import { IConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";
import { ITeamSubnavProps } from "interfaces/team_subnav";
import { IDropdownOption } from "interfaces/dropdownOption";

import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import { getCustomDropdownOptions } from "utilities/helpers";

import HostStatusWebhookPreviewModal from "pages/admin/components/HostStatusWebhookPreviewModal";

import validURL from "components/forms/validators/valid_url";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Checkbox from "components/forms/fields/Checkbox";

import TeamHostExpiryToggle from "./components/TeamHostExpiryToggle";

const baseClass = "team-settings";

type ITeamSettingsFormData = {
  teamHostExpiryEnabled: boolean;
  teamHostExpiryWindow: number | string;
  teamHostStatusWebhookEnabled: boolean;
  teamHostStatusWebhookDestinationUrl: string;
  teamHostStatusWebhookHostPercentage: number;
  teamHostStatusWebhookWindow: number;
};

type FormNames = keyof ITeamSettingsFormData;

const HOST_EXPIRY_ERROR_TEXT = "Host expiry window must be a positive number.";

const validateTeamSettingsFormData = (
  // will never be called if global setting is not loaded, default to satisfy typechecking
  curGlobalHostExpiryEnabled = false,
  curFormData: ITeamSettingsFormData
) => {
  const errors: Record<string, string> = {};

  // validate host expiry fields
  const numHostExpiryWindow = Number(curFormData.teamHostExpiryWindow);
  if (
    // with no global setting, team window can't be empty if enabled
    (!curGlobalHostExpiryEnabled &&
      curFormData.teamHostExpiryEnabled &&
      !numHostExpiryWindow) ||
    // if nonempty, must be a positive number
    isNaN(numHostExpiryWindow) ||
    // if overriding a global setting, can be empty to disable local setting
    numHostExpiryWindow < 0
  ) {
    errors.host_expiry_window = HOST_EXPIRY_ERROR_TEXT;
  }

  // validate host webhook fields
  if (curFormData.teamHostStatusWebhookEnabled) {
    if (!validURL({ url: curFormData.teamHostStatusWebhookDestinationUrl })) {
      const errorPrefix = curFormData.teamHostStatusWebhookDestinationUrl
        ? `${curFormData.teamHostStatusWebhookDestinationUrl} is not`
        : "Please enter";
      errors.host_status_webhook_destination_url = `${errorPrefix} a valid webhook destination URL`;
    }
  }

  return errors;
};

const TeamSettings = ({ location, router }: ITeamSubnavProps) => {
  const [formData, setFormData] = useState<ITeamSettingsFormData>({
    teamHostExpiryEnabled: false,
    teamHostExpiryWindow: "" as number | string,
    teamHostStatusWebhookEnabled: false,
    teamHostStatusWebhookDestinationUrl: "",
    teamHostStatusWebhookHostPercentage: 1,
    teamHostStatusWebhookWindow: 1,
  });
  // stateful approach required since initial options come from team config api response
  const [isInitialTeamConfig, setIsInitialTeamConfig] = useState(true);
  const [
    percentageHostsDropdownOptions,
    setPercentageHostsDropdownOptions,
  ] = useState<IDropdownOption[]>([]);
  const [windowDropdownOptions, setWindowDropdownOptions] = useState<
    IDropdownOption[]
  >([]);
  const [updatingTeamSettings, setUpdatingTeamSettings] = useState(false);
  const [formErrors, setFormErrors] = useState<Record<string, string | null>>(
    {}
  );
  const [
    showHostStatusWebhookPreviewModal,
    setShowHostStatusWebhookPreviewModal,
  ] = useState(false);

  const toggleHostStatusWebhookPreviewModal = () => {
    setShowHostStatusWebhookPreviewModal(!showHostStatusWebhookPreviewModal);
  };

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
    data: teamConfig,
    isLoading: isLoadingTeamConfig,
    refetch: refetchTeamConfig,
    error: errorLoadTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["teamConfig", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isRouteOk && !!teamIdForApi,
      select: (data) => data.team,
      onSuccess: (tC) => {
        setFormData({
          // host expiry settings
          teamHostExpiryEnabled:
            tC?.host_expiry_settings?.host_expiry_enabled ?? false,
          teamHostExpiryWindow:
            tC?.host_expiry_settings?.host_expiry_window ?? "",
          // host status webhook settings
          teamHostStatusWebhookEnabled:
            tC?.webhook_settings?.host_status_webhook
              ?.enable_host_status_webhook ?? false,
          teamHostStatusWebhookDestinationUrl:
            tC?.webhook_settings?.host_status_webhook?.destination_url ?? "",
          teamHostStatusWebhookHostPercentage:
            tC?.webhook_settings?.host_status_webhook?.host_percentage ?? 1,
          teamHostStatusWebhookWindow:
            tC?.webhook_settings?.host_status_webhook?.days_count ?? 1,
        });
      },
    }
  );

  useEffect(() => {
    if (isInitialTeamConfig) {
      setPercentageHostsDropdownOptions(
        getCustomDropdownOptions(
          HOST_STATUS_WEBHOOK_HOST_PERCENTAGE_DROPDOWN_OPTIONS,
          teamConfig?.webhook_settings?.host_status_webhook?.host_percentage ??
            1,
          (val) => `${val}%`
        )
      );

      setWindowDropdownOptions(
        getCustomDropdownOptions(
          HOST_STATUS_WEBHOOK_WINDOW_DROPDOWN_OPTIONS,
          teamConfig?.webhook_settings?.host_status_webhook?.days_count ?? 1,
          (val) => `${val} day${val !== 1 ? "s" : ""}`
        )
      );
    }
    // no need for isInitialTeamConfig dependence, since this effect should only run on initial
    // config load
  }, [teamConfig]);

  const onInputChange = useCallback(
    (newVal: { name: FormNames; value: string | number | boolean }) => {
      const { name, value } = newVal;
      const newFormData = { ...formData, [name]: value };
      setFormData(newFormData);
      setFormErrors(
        validateTeamSettingsFormData(globalHostExpiryEnabled, newFormData)
      );
    },
    [formData, globalHostExpiryEnabled]
  );

  const updateTeamSettings = useCallback(
    (evt: React.MouseEvent<HTMLFormElement>) => {
      evt.preventDefault();

      setUpdatingTeamSettings(true);
      const castedHostExpiryWindow = Number(formData.teamHostExpiryWindow);
      let enableHostExpiry;
      if (globalHostExpiryEnabled) {
        if (!castedHostExpiryWindow) {
          enableHostExpiry = false;
        } else {
          enableHostExpiry = formData.teamHostExpiryEnabled;
        }
      } else {
        enableHostExpiry = formData.teamHostExpiryEnabled;
      }
      teamsAPI
        .update(
          {
            host_expiry_settings: {
              host_expiry_enabled: enableHostExpiry,
              host_expiry_window: castedHostExpiryWindow,
            },
            webhook_settings: {
              host_status_webhook: {
                enable_host_status_webhook:
                  formData.teamHostStatusWebhookEnabled,
                destination_url: formData.teamHostStatusWebhookDestinationUrl,
                host_percentage: formData.teamHostStatusWebhookHostPercentage,
                days_count: formData.teamHostStatusWebhookWindow,
              },
            },
          },
          teamIdForApi
        )
        .then(() => {
          renderFlash("success", "Successfully updated settings.");
          refetchTeamConfig();
          setIsInitialTeamConfig(false);
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
      formData,
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
      <form onSubmit={updateTeamSettings}>
        <SectionHeader title="Webhook settings" />
        <Checkbox
          name="teamHostStatusWebhookEnabled"
          onChange={onInputChange}
          parseTarget
          value={formData.teamHostStatusWebhookEnabled}
          helpText="This will trigger webhooks specific to this team, separate from the global host status webhook."
          tooltipContent="Send an alert if a portion of your hosts go offline."
        >
          Enable host status webhook
        </Checkbox>
        <Button
          type="button"
          variant="text-link"
          onClick={toggleHostStatusWebhookPreviewModal}
        >
          Preview request
        </Button>
        {formData.teamHostStatusWebhookEnabled && (
          <>
            <InputField
              placeholder="https://server.com/example"
              label="Host status webhook destination URL"
              onChange={onInputChange}
              name="teamHostStatusWebhookDestinationUrl"
              value={formData.teamHostStatusWebhookDestinationUrl}
              parseTarget
              error={formErrors.host_status_webhook_destination_url}
              tooltip={
                <p>
                  Provide a URL to deliver <br />
                  the webhook request to.
                </p>
              }
            />
            <Dropdown
              label="Host status webhook %"
              options={percentageHostsDropdownOptions}
              onChange={onInputChange}
              name="teamHostStatusWebhookHostPercentage"
              value={formData.teamHostStatusWebhookHostPercentage}
              parseTarget
              searchable={false}
              tooltip={
                <p>
                  Select the minimum percentage of hosts that
                  <br />
                  must fail to check into Fleet in order to trigger
                  <br />
                  the webhook request.
                </p>
              }
            />
            <Dropdown
              label="Host status webhook window"
              options={windowDropdownOptions}
              onChange={onInputChange}
              name="teamHostStatusWebhookWindow"
              value={formData.teamHostStatusWebhookWindow}
              parseTarget
              searchable={false}
              tooltip={
                <p>
                  Select the minimum number of days that the
                  <br />
                  configured <b>Percentage of hosts</b> must fail to
                  <br />
                  check into Fleet in order to trigger the
                  <br />
                  webhook request.
                </p>
              }
            />
          </>
        )}
        <SectionHeader title="Host expiry settings" />
        {globalHostExpiryEnabled !== undefined && (
          <TeamHostExpiryToggle
            globalHostExpiryEnabled={globalHostExpiryEnabled}
            globalHostExpiryWindow={globalHostExpiryWindow}
            teamExpiryEnabled={formData.teamHostExpiryEnabled}
            setTeamExpiryEnabled={(isEnabled: boolean) =>
              onInputChange({ name: "teamHostExpiryEnabled", value: isEnabled })
            }
          />
        )}
        {formData.teamHostExpiryEnabled && (
          <InputField
            label="Host expiry window"
            // type="text" allows `validate` to differentiate between
            // non-numerical input and an empty input
            type="text"
            onChange={onInputChange}
            parseTarget
            name="teamHostExpiryWindow"
            value={formData.teamHostExpiryWindow}
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
      {renderForm()}
      {showHostStatusWebhookPreviewModal && (
        <HostStatusWebhookPreviewModal
          toggleModal={toggleHostStatusWebhookPreviewModal}
          isTeamScope
        />
      )}
    </section>
  );
};
export default TeamSettings;

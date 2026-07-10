import React, { useContext, useEffect, useState } from "react";
import { useMutation, useQuery } from "react-query";

import { AppContext } from "context/app";
import { notify } from "components/ToastNotification";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import { getErrorReason } from "interfaces/errors";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";

import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import configAPI from "services/entities/config";
import hostNameTemplateAPI from "services/entities/host_name_template";

import PATHS from "router/paths";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
import EmptyState from "components/EmptyState";
import InputField from "components/forms/fields/InputField";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import Spinner from "components/Spinner";
import SectionHeader from "components/SectionHeader";
import PageDescription from "components/PageDescription";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import { IOSSettingsCommonProps } from "../../OSSettingsNavItems";

const baseClass = "host-name-template";
const NAME_TEMPLATE_MAX_LENGTH = 255;

const HostNameTemplate = ({
  currentTeamId,
  router,
  onMutation,
}: IOSSettingsCommonProps) => {
  const { isPremiumTier, config, setConfig } = useContext(AppContext);

  const mdmEnabled = config?.mdm.enabled_and_configured;

  // "No team" stores its template on the global app config (there is no team
  // row for No team), mirroring how DiskEncryption sources its value by scope.
  const isNoTeam = currentTeamId === API_NO_TEAM_ID;

  const [nameTemplate, setNameTemplate] = useState<string>();
  const [savedNameTemplate, setSavedNameTemplate] = useState<string>();

  // Seed the No-team value from app config once it's available, and re-seed
  // after a save refreshes the config. Fleets seed from the team query below.
  useEffect(() => {
    if (isNoTeam) {
      const loaded = config?.mdm.name_template ?? "";
      setNameTemplate(loaded);
      setSavedNameTemplate(loaded);
    }
  }, [isNoTeam, config?.mdm.name_template]);

  const { isLoading: isLoadingTeam, isError: isTeamError } = useQuery<
    ILoadTeamResponse,
    Error,
    ITeamConfig
  >(["team", currentTeamId], () => teamsAPI.load(currentTeamId), {
    ...DEFAULT_USE_QUERY_OPTIONS,
    enabled: isPremiumTier && !!mdmEnabled && !isNoTeam,
    select: (res) => res.fleet,
    onSuccess: (res) => {
      const loaded = res.mdm?.name_template ?? "";
      setNameTemplate(loaded);
      setSavedNameTemplate(loaded);
    },
    onError: (err) => {
      notify.error("Couldn't load fleet settings. Please try again.", {
        response: err,
      });
    },
  });

  const { mutate: saveNameTemplate, isLoading: updating } = useMutation(
    (tmpl: string) =>
      hostNameTemplateAPI.updateHostNameTemplate(tmpl, currentTeamId),
    {
      onSuccess: async (_data, tmpl) => {
        // Optimistically adopt the saved value as the new baseline.
        setSavedNameTemplate(tmpl);
        notify.success("Successfully updated host name template.");
        onMutation();
        // The No-team template lives on the global app config, so refresh the
        // cached config to keep it in sync (mirrors DiskEncryption).
        if (isNoTeam) {
          try {
            setConfig(await configAPI.loadAll());
          } catch (err) {
            notify.error(
              "Could not retrieve updated app config. Please try again.",
              { response: err }
            );
          }
        }
      },
      onError: (e) => {
        // The server's 422 carries the specific invalid-variable message;
        // surface it verbatim.
        notify.error(
          getErrorReason(e) ||
            "Couldn't update host name template. Please try again.",
          { response: e }
        );
      },
    }
  );

  const isFormDisabled =
    isLoadingTeam || isTeamError || nameTemplate === undefined;
  const isPristine = nameTemplate === savedNameTemplate;

  const renderCardBody = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }
    // Waiting on app config to know whether MDM is configured.
    if (mdmEnabled === undefined) {
      return <Spinner />;
    }
    if (!mdmEnabled) {
      return (
        <EmptyState
          variant="form"
          header="Manage your hosts"
          info="MDM must be turned on to apply host name settings."
          primaryButton={
            <Button onClick={() => router.push(PATHS.ADMIN_INTEGRATIONS_MDM)}>
              Turn on
            </Button>
          }
        />
      );
    }
    if (isLoadingTeam) {
      return <Spinner />;
    }
    return (
      <div className={`form ${baseClass}__content`}>
        <InputField
          label="Name template"
          name="name-template"
          value={nameTemplate ?? ""}
          onChange={(value: string) => setNameTemplate(value)}
          placeholder="iPad $FLEET_VAR_HOST_HARDWARE_SERIAL"
          helpText="This will be the host's name in Fleet and on the device itself."
          disabled={isFormDisabled || config?.gitops?.gitops_mode_enabled}
          inputOptions={{ maxLength: NAME_TEMPLATE_MAX_LENGTH }}
        />
        <div className="button-wrap">
          <GitOpsModeTooltipWrapper
            tipOffset={8}
            renderChildren={(gitopsDisabled) => (
              <Button
                disabled={
                  isFormDisabled || isPristine || updating || gitopsDisabled
                }
                isLoading={updating}
                className={`${baseClass}__save-button`}
                onClick={() =>
                  nameTemplate !== undefined && saveNameTemplate(nameTemplate)
                }
              >
                Save
              </Button>
            )}
          />
        </div>
      </div>
    );
  };

  return (
    <div className={baseClass}>
      <SectionHeader title="Host names" alignLeftHeaderVertically />
      <PageDescription
        variant="right-panel"
        content={
          <>
            Set a naming convention for all macOS, iOS, or iPadOS hosts in this
            fleet. Use{" "}
            <CustomLink
              text="built-in"
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/built-in-variables`}
              newTab
            />{" "}
            or{" "}
            <CustomLink
              text="custom"
              url={getPathWithQueryParams(PATHS.CONTROLS_VARIABLES, {
                fleet_id: currentTeamId,
              })}
            />{" "}
            variables to differentiate between hosts.
          </>
        }
      />
      {renderCardBody()}
    </div>
  );
};

export default HostNameTemplate;

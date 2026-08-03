import React, { useContext, useEffect, useRef, useState } from "react";
import { useMutation, useQuery } from "react-query";

import { AppContext } from "context/app";
import { notify } from "components/ToastNotification";
import { API_NO_TEAM_ID } from "interfaces/team";
import { getErrorReason } from "interfaces/errors";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  LEARN_MORE_ABOUT_BASE_LINK,
} from "utilities/constants";

import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import configAPI from "services/entities/config";
import hostNameTemplateAPI from "services/entities/host_name_template";

import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

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

  const {
    data: teamData,
    isLoading: isLoadingTeam,
    isError: isTeamError,
  } = useQuery<ILoadTeamResponse, Error>(
    ["team", currentTeamId],
    () => teamsAPI.load(currentTeamId),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier && !!mdmEnabled && !isNoTeam,
      onError: (err) => {
        notify.error("Couldn't load fleet settings. Please try again.", {
          response: err,
        });
      },
    }
  );

  // Seed the form once per scope (team) from whichever source owns the
  // template — the global app config for "No team", or the team query for a
  // fleet. An effect keeps state seeding in one place (react-query deprecates
  // useQuery's onSuccess in newer versions). Guarding on the scope prevents a
  // post-save config refresh (No team) or a background team refetch from
  // clobbering in-progress edits; switching teams re-seeds for the new scope.
  const seededTeamRef = useRef<number | null>(null);
  useEffect(() => {
    if (seededTeamRef.current === currentTeamId) {
      return;
    }
    if (isNoTeam) {
      if (!config) {
        return; // wait until app config is available
      }
      const loaded = config.mdm.name_template ?? "";
      setNameTemplate(loaded);
      setSavedNameTemplate(loaded);
      seededTeamRef.current = currentTeamId;
    } else if (teamData) {
      const loaded = teamData.fleet?.mdm?.name_template ?? "";
      setNameTemplate(loaded);
      setSavedNameTemplate(loaded);
      seededTeamRef.current = currentTeamId;
    }
  }, [currentTeamId, isNoTeam, config, teamData]);

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

  const scopeSuffix = isNoTeam ? "." : " in this fleet.";
  const builtInVariablesUrl = `${LEARN_MORE_ABOUT_BASE_LINK}/built-in-variables`;
  const customVariablesUrl = getPathWithQueryParams(PATHS.CONTROLS_VARIABLES, {
    fleet_id: currentTeamId,
  });

  const description = (
    <>
      Set a naming convention for all macOS, iOS, or iPadOS hosts{scopeSuffix}{" "}
      Use <CustomLink text="built-in" url={builtInVariablesUrl} newTab /> or{" "}
      <CustomLink text="custom" url={customVariablesUrl} /> variables to
      differentiate between hosts.
    </>
  );

  return (
    <div className={baseClass}>
      <SectionHeader title="Host names" alignLeftHeaderVertically />
      <PageDescription variant="right-panel" content={description} />
      {renderCardBody()}
    </div>
  );
};

export default HostNameTemplate;

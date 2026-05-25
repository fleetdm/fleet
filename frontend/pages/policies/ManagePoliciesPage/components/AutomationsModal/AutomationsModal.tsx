import React, { useContext, useRef, useState } from "react";
import { useQueryClient } from "react-query";
import { InjectedRouter } from "react-router/lib/Router";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import { IConfig, isConditionalAccessConfigured } from "interfaces/config";
import { ITeamIntegrations } from "interfaces/integration";
import { API_NO_TEAM_ID, ITeamConfig } from "interfaces/team";
import configAPI from "services/entities/config";
import teamsAPI, {
  ILoadTeamResponse,
  IUpdateTeamFormData,
} from "services/entities/teams";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import OtherWorkflowsModal, {
  IOtherWorkflowsModalHandle,
} from "../OtherWorkflowsModal/OtherWorkflowsModal";
import CalendarEventsModal, {
  ICalendarEventsModalHandle,
} from "../CalendarEventsModal/CalendarEventsModal";
import ConditionalAccessModal, {
  IConditionalAccessModalHandle,
} from "../ConditionalAccessModal/ConditionalAccessModal";
import CalendarEventPreviewModal from "../CalendarEventPreviewModal";

const baseClass = "automations-modal";

const SUCCESS_MSG = "Successfully updated policy automations.";
const ERR_MSG = "Could not update policy automations.";

interface IAutomationsModalProps {
  router: InjectedRouter;
  /** -1 = "All teams" sentinel from useTeamIdParam; otherwise team ID (0 for "No team"). */
  isAllTeamsSelected: boolean;
  /** undefined for "All teams", 0 for "No team", positive for a team. */
  teamIdForApi: number | undefined;
  globalConfig: IConfig | undefined;
  teamConfig: ITeamConfig | undefined;
  gitOpsModeEnabled?: boolean;
  /** Refresh policies after a successful save (so per-policy automation flags
   *  show fresh values where applicable). */
  refetchPolicies: () => void;
  onExit: () => void;
}

const AutomationsModal = ({
  router,
  isAllTeamsSelected,
  teamIdForApi,
  globalConfig,
  teamConfig,
  gitOpsModeEnabled = false,
  refetchPolicies,
  onExit,
}: IAutomationsModalProps): JSX.Element | null => {
  const queryClient = useQueryClient();
  const {
    setConfig,
    isGlobalAdmin,
    isTeamAdmin,
    isGlobalMaintainer,
    isTeamMaintainer,
  } = useContext(AppContext);
  const isAdmin = isGlobalAdmin || isTeamAdmin;
  const isMaintainer = isGlobalMaintainer || isTeamMaintainer;
  const { renderFlash } = useContext(NotificationContext);

  const otherFormRef = useRef<IOtherWorkflowsModalHandle>(null);
  const calendarFormRef = useRef<ICalendarEventsModalHandle>(null);
  const conditionalAccessFormRef = useRef<IConditionalAccessModalHandle>(null);

  const [isUpdating, setIsUpdating] = useState(false);

  // Use teamConfig when a team is selected (including No team); otherwise globalConfig
  // is the source for the "Other workflows" automations config in the All-teams view.
  const automationsConfig = isAllTeamsSelected ? globalConfig : teamConfig;

  // Available jira/zendesk integrations are stored at the global level.
  const availableIntegrations =
    globalConfig?.integrations ?? automationsConfig?.integrations;

  const isCalEventsConfigured =
    (globalConfig?.integrations.google_calendar &&
      globalConfig?.integrations.google_calendar.length > 0) ??
    false;
  const isCalEventsEnabled =
    teamConfig?.integrations.google_calendar?.enable_calendar_events ?? false;
  const calendarUrl =
    teamConfig?.integrations.google_calendar?.webhook_url || "";
  const [showPreviewCalendarEvent, setShowPreviewCalendarEvent] = useState(
    false
  );
  const togglePreviewCalendarEvent = () =>
    setShowPreviewCalendarEvent(!showPreviewCalendarEvent);

  const isCAConfigured = isConditionalAccessConfigured(globalConfig);
  const isCAEnabled =
    (teamIdForApi === API_NO_TEAM_ID
      ? globalConfig?.integrations.conditional_access_enabled
      : teamConfig?.integrations.conditional_access_enabled) ?? false;

  const isManagedCloud = globalConfig?.license?.managed_cloud || false;
  const conditionalAccessProviderText = isManagedCloud
    ? "Okta or Microsoft Entra"
    : "Okta";

  const updateGlobalConfigCache = (updatedConfig: IConfig) => {
    queryClient.setQueryData(["config"], updatedConfig);
    setConfig(updatedConfig);
  };
  const updateTeamConfigCache = (updatedTeamResponse: ILoadTeamResponse) => {
    queryClient.setQueryData(["teams", teamIdForApi], updatedTeamResponse);
  };

  const onSave = async () => {
    // Validate every visible form. We always show OtherWorkflowsModal.
    const otherValid = otherFormRef.current?.validate() ?? true;
    const calendarValid = calendarFormRef.current?.validate() ?? true;
    const caValid = conditionalAccessFormRef.current?.validate() ?? true;
    if (!otherValid || !calendarValid || !caValid) {
      return;
    }

    const otherData = otherFormRef.current?.getFormData() ?? null;
    const calendarData = calendarFormRef.current?.getFormData() ?? null;
    const caData = conditionalAccessFormRef.current?.getFormData() ?? null;

    setIsUpdating(true);
    try {
      if (isAllTeamsSelected) {
        // Global ("All teams"): only Other is editable.
        if (otherData) {
          const updatedConfig = await configAPI.update(otherData);
          updateGlobalConfigCache(updatedConfig);
        }
      } else if (teamIdForApi === API_NO_TEAM_ID) {
        // "No team": webhook_settings + google_calendar live on the team record,
        // conditional_access_enabled lives on the global config.
        const integrations: ITeamIntegrations = {
          jira: otherData?.integrations.jira ?? [],
          zendesk: otherData?.integrations.zendesk ?? [],
        };
        if (calendarData) {
          integrations.google_calendar = {
            enable_calendar_events: calendarData.enabled,
            webhook_url: calendarData.url,
          };
        }

        const teamPayload: Partial<IUpdateTeamFormData> = { integrations };
        if (otherData) {
          teamPayload.webhook_settings = otherData.webhook_settings;
        }

        const promises: Promise<unknown>[] = [];
        if (otherData || calendarData) {
          promises.push(
            teamsAPI
              .update(teamPayload, teamIdForApi)
              .then(updateTeamConfigCache)
          );
        }
        if (caData) {
          promises.push(
            configAPI
              .update({
                integrations: {
                  conditional_access_enabled: caData.enabled,
                },
              })
              .then(updateGlobalConfigCache)
          );
        }
        await Promise.all(promises);
      } else if (teamIdForApi !== undefined) {
        // A real team: everything goes to teams.update in a single payload.
        const integrations: ITeamIntegrations = {
          jira: otherData?.integrations.jira ?? [],
          zendesk: otherData?.integrations.zendesk ?? [],
        };
        if (calendarData) {
          integrations.google_calendar = {
            enable_calendar_events: calendarData.enabled,
            webhook_url: calendarData.url,
          };
        }
        if (caData) {
          integrations.conditional_access_enabled = caData.enabled;
        }

        const teamPayload: Partial<IUpdateTeamFormData> = { integrations };
        if (otherData) {
          teamPayload.webhook_settings = otherData.webhook_settings;
        }

        if (otherData || calendarData || caData) {
          const updatedTeam = await teamsAPI.update(teamPayload, teamIdForApi);
          updateTeamConfigCache(updatedTeam);
        }
      }

      renderFlash("success", SUCCESS_MSG);
      refetchPolicies();
      onExit();
    } catch {
      renderFlash("error", ERR_MSG);
    } finally {
      setIsUpdating(false);
    }
  };

  if (!automationsConfig || !availableIntegrations) {
    return null;
  }

  return (
    <Modal
      title="Automations"
      onExit={onExit}
      className={baseClass}
      width="large"
      isContentDisabled={isUpdating}
    >
      <div className={`${baseClass}__body`}>
        {isAdmin && (
          <section className={`${baseClass}__section`}>
            {!isAllTeamsSelected && (
              <h2 className={`${baseClass}__section-title`}>
                Webhooks or tickets
              </h2>
            )}
            <OtherWorkflowsModal
              ref={otherFormRef}
              router={router}
              automationsConfig={automationsConfig}
              availableIntegrations={availableIntegrations}
              gitOpsModeEnabled={gitOpsModeEnabled}
            />
          </section>
        )}

        {!isAllTeamsSelected && (isAdmin || isMaintainer) && (
          <>
            {isAdmin && <hr className={`${baseClass}__divider`} />}
            <section className={`${baseClass}__section ${baseClass}__section`}>
              <div className={`${baseClass}__calendar-events-title-wrapper`}>
                <h2 className={`${baseClass}__section-title`}>
                  Calendar events
                </h2>
                {isCalEventsConfigured && (
                  <>
                    <Button
                      type="button"
                      variant="brand-inverse-icon"
                      onClick={togglePreviewCalendarEvent}
                    >
                      Preview calendar event
                    </Button>

                    {showPreviewCalendarEvent && (
                      <CalendarEventPreviewModal
                        onCancel={togglePreviewCalendarEvent}
                      />
                    )}
                  </>
                )}
              </div>
              <CalendarEventsModal
                ref={calendarFormRef}
                configured={isCalEventsConfigured}
                enabled={isCalEventsEnabled}
                url={calendarUrl}
                gitOpsModeEnabled={gitOpsModeEnabled}
              />
            </section>

            {isAdmin && (
              <>
                <hr className={`${baseClass}__divider`} />
                <section className={`${baseClass}__section`}>
                  <h2 className={`${baseClass}__section-title`}>
                    Conditional access
                  </h2>
                  <ConditionalAccessModal
                    ref={conditionalAccessFormRef}
                    configured={isCAConfigured}
                    enabled={isCAEnabled}
                    gitOpsModeEnabled={gitOpsModeEnabled}
                    providerText={conditionalAccessProviderText}
                  />
                </section>
              </>
            )}
          </>
        )}
      </div>
      <div className="modal-cta-wrap">
        <Button onClick={onExit} variant="inverse">
          Cancel
        </Button>
        <Button onClick={onSave} isLoading={isUpdating} disabled={isUpdating}>
          Save
        </Button>
      </div>
    </Modal>
  );
};

export default AutomationsModal;

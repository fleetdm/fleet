import React, { useState, useEffect } from "react";
import { Link } from "react-router";
import { isEmpty, noop, omit } from "lodash";

import { IAutomationsConfig, IWebhookSettings } from "interfaces/config";
import {
  IGlobalIntegrations,
  IIntegration,
  IZendeskJiraIntegrations,
  ITeamIntegrations,
} from "interfaces/integration";
import { IPolicy } from "interfaces/policy";
import { ITeamAutomationsConfig } from "interfaces/team";
import PATHS from "router/paths";

import Modal from "components/Modal";
import Slider from "components/forms/fields/Slider";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Radio from "components/forms/fields/Radio";
import validUrl from "components/forms/validators/valid_url";
import RevealButton from "components/buttons/RevealButton";
import CustomLink from "components/CustomLink";
import ExampleTicket from "../ExampleTicket";
import ExamplePayload from "../ExamplePayload";

import PoliciesPaginatedList, {
  IFormPolicy,
} from "../PoliciesPaginatedList/PoliciesPaginatedList";

interface IOtherWorkflowsModalProps {
  automationsConfig: IAutomationsConfig | ITeamAutomationsConfig;
  availableIntegrations: IGlobalIntegrations | ITeamIntegrations;
  availablePolicies: IPolicy[];
  isUpdating: boolean;
  onExit: () => void;
  onSubmit: (formData: {
    webhook_settings: Pick<IWebhookSettings, "failing_policies_webhook">;
    integrations: IGlobalIntegrations | ITeamIntegrations;
  }) => void;
  teamId: number;
  gitOpsModeEnabled?: boolean;
}

const findEnabledIntegration = ({
  jira,
  zendesk,
}: IZendeskJiraIntegrations) => {
  return (
    jira?.find((j) => j.enable_failing_policies) ||
    zendesk?.find((z) => z.enable_failing_policies)
  );
};

const getIntegrationType = (integration?: IIntegration) => {
  return (
    (!!integration?.group_id && "zendesk") ||
    (!!integration?.project_key && "jira") ||
    undefined
  );
};

const baseClass = "other-workflows-modal";

const OtherWorkflowsModal = ({
  automationsConfig,
  availableIntegrations,
  availablePolicies,
  isUpdating,
  onExit,
  onSubmit,
  teamId,
  gitOpsModeEnabled = false,
}: IOtherWorkflowsModalProps): JSX.Element => {
  const {
    webhook_settings: { failing_policies_webhook: webhook },
  } = automationsConfig;

  const [newPolicyIds, setNewPolicyIds] = useState(webhook.policy_ids || []);

  const { jira, zendesk } = availableIntegrations || {};
  const allIntegrations: IIntegration[] = [];
  jira && allIntegrations.push(...jira);
  zendesk && allIntegrations.push(...zendesk);

  const dropdownOptions = allIntegrations.map(
    ({ group_id, project_key, url }) => ({
      value: group_id || project_key,
      label: `${url} - ${group_id || project_key}`,
    })
  );

  const serverEnabledIntegration = findEnabledIntegration(
    automationsConfig.integrations
  );

  const [isPolicyAutomationsEnabled, setIsPolicyAutomationsEnabled] = useState(
    !!webhook.enable_failing_policies_webhook || !!serverEnabledIntegration
  );

  const [isWebhookEnabled, setIsWebhookEnabled] = useState(
    !isPolicyAutomationsEnabled || webhook.enable_failing_policies_webhook
  );

  const [destinationUrl, setDestinationUrl] = useState(
    webhook.destination_url || ""
  );

  const [selectedIntegration, setSelectedIntegration] = useState<
    IIntegration | undefined
  >(serverEnabledIntegration);

  const [showExamplePayload, setShowExamplePayload] = useState(false);
  const [showExampleTicket, setShowExampleTicket] = useState(false);

  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  const onChangeUrl = (value: string) => {
    setDestinationUrl(value);
    setErrors((errs) => omit(errs, "url"));
  };

  const onChangeRadio = (val: string) => {
    switch (val) {
      case "webhook":
        setIsWebhookEnabled(true);
        setSelectedIntegration(undefined);
        break;
      case "ticket":
        setIsWebhookEnabled(false);
        break;
      default:
        noop();
    }
  };

  const onSelectIntegration = (selected: string | number) => {
    setSelectedIntegration(
      allIntegrations.find(
        ({ group_id, project_key }) =>
          group_id === selected || project_key === selected
      )
    );
  };

  const onUpdateOtherWorkflows = () => {
    const newErrors = { ...errors };

    if (isPolicyAutomationsEnabled) {
      // Ticket workflow validation
      if (newPolicyIds.length && !isWebhookEnabled && !selectedIntegration) {
        newErrors.integration = "Please enable at least one integration:";
      } else {
        delete newErrors.integration;
      }

      // Webhook workflow validation
      if (isWebhookEnabled) {
        if (!destinationUrl) {
          newErrors.url = "Please add a destination URL";
        } else if (!validUrl({ url: destinationUrl })) {
          newErrors.url = `${destinationUrl} is not a valid URL`;
        } else {
          delete newErrors.url;
        }
      }
    }

    if (!isEmpty(newErrors)) {
      setErrors(newErrors);
      return;
    }

    const newJira =
      availableIntegrations.jira?.map((j) => ({
        ...j,
        enable_failing_policies:
          isPolicyAutomationsEnabled &&
          !isWebhookEnabled &&
          j.project_key === selectedIntegration?.project_key,
      })) || null;

    const newZendesk =
      availableIntegrations.zendesk?.map((z) => ({
        ...z,
        enable_failing_policies:
          isPolicyAutomationsEnabled &&
          !isWebhookEnabled &&
          z.group_id === selectedIntegration?.group_id,
      })) || null;

    // NOTE: backend uses webhook_settings to store automated policy ids for both webhooks and integrations
    const newWebhook = {
      failing_policies_webhook: {
        destination_url: destinationUrl,
        policy_ids: newPolicyIds,
        enable_failing_policies_webhook:
          isPolicyAutomationsEnabled && isWebhookEnabled,
      },
    };

    onSubmit({
      webhook_settings: newWebhook,
      integrations: {
        jira: newJira,
        zendesk: newZendesk,
        google_calendar: null, // When null, the backend does not update google_calendar
      },
    });

    setErrors(newErrors);
  };

  useEffect(() => {
    const listener = (event: KeyboardEvent) => {
      if (event.code === "Enter" || event.code === "NumpadEnter") {
        event.preventDefault();
        onUpdateOtherWorkflows();
      }
    };
    document.addEventListener("keydown", listener);
    return () => {
      document.removeEventListener("keydown", listener);
    };
  });

  const renderWebhook = () => {
    return (
      <>
        <InputField
          inputWrapperClass={`${baseClass}__url-input`}
          name="webhook-url"
          label="Destination URL"
          type="text"
          value={destinationUrl}
          onChange={onChangeUrl}
          error={errors.url}
          helpText='For each policy, Fleet will send a JSON payload to this URL with a list of the hosts that updated their answer to "No."'
          placeholder="https://server.com/example"
          tooltip="Provide a URL to deliver a webhook request to."
          disabled={!isPolicyAutomationsEnabled || gitOpsModeEnabled}
        />
        <RevealButton
          isShowing={showExamplePayload}
          className={baseClass}
          hideText="Hide example payload"
          showText="Show example payload"
          caretPosition="after"
          onClick={() => setShowExamplePayload(!showExamplePayload)}
          disabled={!isPolicyAutomationsEnabled || gitOpsModeEnabled}
        />
        {showExamplePayload && <ExamplePayload />}
      </>
    );
  };

  const renderIntegrations = () => {
    return jira?.length || zendesk?.length ? (
      <>
        <div className={`${baseClass}__integrations`}>
          <Dropdown
            options={dropdownOptions}
            onChange={onSelectIntegration}
            placeholder="Select integration"
            value={
              selectedIntegration?.group_id || selectedIntegration?.project_key
            }
            label="Integration"
            error={errors.integration}
            wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--frequency`}
            hint={
              "For each policy, Fleet will create a ticket with a list of the failing hosts."
            }
          />
        </div>
        <RevealButton
          isShowing={showExampleTicket}
          className={baseClass}
          hideText="Hide example ticket"
          showText="Show example ticket"
          caretPosition="after"
          onClick={() => setShowExampleTicket(!showExampleTicket)}
        />
        {showExampleTicket && (
          <ExampleTicket
            integrationType={getIntegrationType(selectedIntegration)}
          />
        )}
      </>
    ) : (
      <div className={`form-field ${baseClass}__no-integrations`}>
        <div className="form-field__label">You have no integrations.</div>
        <Link
          to={PATHS.ADMIN_INTEGRATIONS}
          className={`${baseClass}__add-integration-link`}
        >
          Add integration
        </Link>
      </div>
    );
  };

  return (
    <Modal
      onExit={onExit}
      title="Other workflows"
      className={baseClass}
      width="large"
      isContentDisabled={isUpdating}
    >
      <div className={`${baseClass} form`}>
        <Slider
          value={isPolicyAutomationsEnabled}
          onChange={() => {
            setIsPolicyAutomationsEnabled(!isPolicyAutomationsEnabled);
            setErrors({});
          }}
          inactiveText="Disabled"
          activeText="Enabled"
          autoFocus
          disabled={gitOpsModeEnabled}
        />
        <div
          className={`form ${baseClass}__policy-automations__${
            isPolicyAutomationsEnabled ? "enabled" : "disabled"
          }`}
        >
          <div className={`form-field ${baseClass}__workflow`}>
            <div className="form-field__label">Workflow</div>
            <Radio
              className={`${baseClass}__radio-input`}
              label="Ticket"
              id="ticket-radio-btn"
              checked={!isWebhookEnabled}
              value="ticket"
              name="workflow-type"
              onChange={onChangeRadio}
              disabled={!isPolicyAutomationsEnabled || gitOpsModeEnabled}
            />
            <Radio
              className={`${baseClass}__radio-input`}
              label="Webhook"
              id="webhook-radio-btn"
              checked={isWebhookEnabled}
              value="webhook"
              name="workflow-type"
              onChange={onChangeRadio}
              disabled={!isPolicyAutomationsEnabled || gitOpsModeEnabled}
            />
          </div>
          {isWebhookEnabled ? renderWebhook() : renderIntegrations()}
        </div>
        <div className="form-field">
          {availablePolicies?.length ? (
            <PoliciesPaginatedList
              isSelected={(item: IFormPolicy) => {
                return newPolicyIds?.indexOf(item.id) > -1;
              }}
              onToggleItem={(item: IFormPolicy) => {
                const updatedPolicyIds = newPolicyIds.slice();
                const index = newPolicyIds?.indexOf(item.id);
                if (index > -1) {
                  updatedPolicyIds.splice(index, 1);
                } else {
                  updatedPolicyIds.push(item.id);
                }
                setNewPolicyIds(updatedPolicyIds);
                return item;
              }}
              helpText={
                <>
                  The workflow will be triggered when hosts fail these policies.{" "}
                  <CustomLink
                    url="https://www.fleetdm.com/learn-more-about/policy-automations"
                    text="Learn more"
                    newTab
                    disableKeyboardNavigation={!isPolicyAutomationsEnabled}
                  />
                </>
              }
              isUpdating={isUpdating}
              onSubmit={onUpdateOtherWorkflows}
              onCancel={onExit}
              teamId={teamId}
              disableList={!isPolicyAutomationsEnabled}
            />
          ) : (
            <>
              <b>You have no policies.</b>
              <p>Add a policy to turn on automations.</p>
            </>
          )}
        </div>
      </div>
    </Modal>
  );
};

export default OtherWorkflowsModal;

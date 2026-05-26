import React, { forwardRef, useImperativeHandle, useState } from "react";
import { InjectedRouter } from "react-router";
import { isEmpty, noop, omit } from "lodash";

import { IAutomationsConfig, IWebhookSettings } from "interfaces/config";
import {
  IGlobalIntegrations,
  IIntegration,
  IZendeskJiraIntegrations,
  ITeamIntegrations,
} from "interfaces/integration";
import { ITeamAutomationsConfig } from "interfaces/team";
import PATHS from "router/paths";

import Slider from "components/forms/fields/Slider";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import Radio from "components/forms/fields/Radio";
import validUrl from "components/forms/validators/valid_url";
import RevealButton from "components/buttons/RevealButton";
import CustomLink from "components/CustomLink";
import ExampleTicket from "../ExampleTicket";
import ExamplePayload from "../ExamplePayload";

const baseClass = "other-workflows-modal";

export interface IOtherWorkflowsModalSubmit {
  webhook_settings: Pick<IWebhookSettings, "failing_policies_webhook">;
  integrations: IGlobalIntegrations | ITeamIntegrations;
}

export interface IOtherWorkflowsModalHandle {
  getFormData: () => IOtherWorkflowsModalSubmit | null;
  validate: () => boolean;
  isDirty: () => boolean;
}

interface IOtherWorkflowsModalProps {
  router: InjectedRouter;
  automationsConfig: IAutomationsConfig | ITeamAutomationsConfig;
  availableIntegrations: IGlobalIntegrations | ITeamIntegrations;
  gitOpsModeEnabled?: boolean;
}

const findEnabledIntegration = ({
  jira,
  zendesk,
}: IZendeskJiraIntegrations): IIntegration | undefined =>
  jira?.find((j) => j.enable_failing_policies) ||
  zendesk?.find((z) => z.enable_failing_policies);

const getIntegrationType = (integration?: IIntegration) =>
  (!!integration?.group_id && "zendesk") ||
  (!!integration?.project_key && "jira") ||
  undefined;

const OtherWorkflowsModal = forwardRef<
  IOtherWorkflowsModalHandle,
  IOtherWorkflowsModalProps
>(
  (
    {
      router,
      automationsConfig,
      availableIntegrations,
      gitOpsModeEnabled = false,
    }: IOtherWorkflowsModalProps,
    ref
  ) => {
    const {
      webhook_settings: { failing_policies_webhook: webhook },
    } = automationsConfig;

    const { jira, zendesk } = availableIntegrations || {};
    const allIntegrations: IIntegration[] = [];
    jira && allIntegrations.push(...jira);
    zendesk && allIntegrations.push(...zendesk);
    const hasAvailableIntegrations = allIntegrations.length > 0;

    const dropdownOptions = allIntegrations.map(
      ({ group_id, project_key, url }) => ({
        value: group_id || project_key,
        label: `${url} - ${group_id || project_key}`,
      })
    );

    const serverEnabledIntegration = findEnabledIntegration(
      automationsConfig.integrations
    );

    const initialIsPolicyAutomationsEnabled =
      !!webhook.enable_failing_policies_webhook || !!serverEnabledIntegration;
    const initialIsWebhookEnabled =
      !initialIsPolicyAutomationsEnabled ||
      webhook.enable_failing_policies_webhook;
    const initialDestinationUrl = webhook.destination_url || "";

    const [
      isPolicyAutomationsEnabled,
      setIsPolicyAutomationsEnabled,
    ] = useState(initialIsPolicyAutomationsEnabled);
    const [isWebhookEnabled, setIsWebhookEnabled] = useState(
      initialIsWebhookEnabled
    );
    const [destinationUrl, setDestinationUrl] = useState(initialDestinationUrl);
    const [selectedIntegration, setSelectedIntegration] = useState<
      IIntegration | undefined
    >(serverEnabledIntegration);
    const [showExamplePayload, setShowExamplePayload] = useState(false);
    const [showExampleTicket, setShowExampleTicket] = useState(false);
    const [errors, setErrors] = useState<{ [key: string]: string }>({});

    const buildSubmitData = (): IOtherWorkflowsModalSubmit => {
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
          policy_ids: webhook.policy_ids || [],
          enable_failing_policies_webhook:
            isPolicyAutomationsEnabled && isWebhookEnabled,
        },
      };

      return {
        webhook_settings: newWebhook,
        integrations: {
          jira: newJira,
          zendesk: newZendesk,
          google_calendar: null, // When null, backend does not update google_calendar
        },
      };
    };

    const runValidation = () => {
      const newErrors: { [key: string]: string } = {};

      if (isPolicyAutomationsEnabled) {
        if (!isWebhookEnabled && !selectedIntegration) {
          newErrors.integration = hasAvailableIntegrations
            ? "Please enable at least one integration:"
            : "Add an integration to create tickets for policy automations.";
        }
        if (isWebhookEnabled) {
          if (!destinationUrl) {
            newErrors.url = "Please add a destination URL";
          } else if (!validUrl({ url: destinationUrl })) {
            newErrors.url = "Destination URL is not a valid URL";
          }
        }
      }
      return newErrors;
    };

    useImperativeHandle(ref, () => ({
      getFormData: () => buildSubmitData(),
      validate: () => {
        const newErrors = runValidation();
        setErrors(newErrors);
        return isEmpty(newErrors);
      },
      isDirty: () => {
        if (isPolicyAutomationsEnabled !== initialIsPolicyAutomationsEnabled)
          return true;
        if (isPolicyAutomationsEnabled) {
          if (isWebhookEnabled !== !!initialIsWebhookEnabled) return true;
          if (isWebhookEnabled && destinationUrl !== initialDestinationUrl)
            return true;
          if (
            !isWebhookEnabled &&
            (selectedIntegration?.project_key !==
              serverEnabledIntegration?.project_key ||
              selectedIntegration?.group_id !==
                serverEnabledIntegration?.group_id)
          )
            return true;
        }
        return false;
      },
    }));

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

    const onAddIntegration = () => {
      router.push(PATHS.ADMIN_INTEGRATIONS);
    };

    const onSelectIntegration = (selected: string | number) => {
      setSelectedIntegration(
        allIntegrations.find(
          ({ group_id, project_key }) =>
            group_id === selected || project_key === selected
        )
      );
    };

    const renderWebhook = () => (
      <>
        <InputField
          inputWrapperClass={`${baseClass}__url-input`}
          name="webhook-url"
          label="Destination URL"
          type="text"
          value={destinationUrl}
          onChange={onChangeUrl}
          error={errors.url}
          helpText="For configured policies, Fleet will send a JSON payload to this URL with a list of hosts whose statuses changed from pass to fail."
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

    const renderIntegrations = () =>
      hasAvailableIntegrations ? (
        <>
          <div className={`${baseClass}__integrations`}>
            <Dropdown
              options={dropdownOptions}
              onChange={onSelectIntegration}
              placeholder="Select integration"
              value={
                selectedIntegration?.group_id ||
                selectedIntegration?.project_key
              }
              label="Integration"
              error={errors.integration}
              wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--frequency`}
              hint="For each policy, Fleet will create a ticket with a list of the failing hosts."
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
          <div>
            <Button
              onClick={onAddIntegration}
              disabled={gitOpsModeEnabled || !isPolicyAutomationsEnabled}
            >
              Add integration
            </Button>
          </div>
          {errors.integration && (
            <div className={`${baseClass}__error`}>{errors.integration}</div>
          )}
        </div>
      );

    return (
      <div className={`${baseClass} form`}>
        <p className={`${baseClass}__description`}>
          Create tickets or fire webhooks when hosts fail policies.{" "}
          <CustomLink
            url="https://www.fleetdm.com/learn-more-about/policy-automations"
            text="Learn more"
            newTab
          />
        </p>
        <Slider
          value={isPolicyAutomationsEnabled}
          onChange={() => {
            setIsPolicyAutomationsEnabled(!isPolicyAutomationsEnabled);
            setErrors({});
          }}
          inactiveText="Disabled"
          activeText="Enabled"
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
      </div>
    );
  }
);

OtherWorkflowsModal.displayName = "OtherWorkflowsModal";

export default OtherWorkflowsModal;

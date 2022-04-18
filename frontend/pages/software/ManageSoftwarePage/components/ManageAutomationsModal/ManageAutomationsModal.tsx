import React, { useState } from "react";
import { useQuery } from "react-query";

import { Link } from "react-router";
import PATHS from "router/paths";

import {
  IJiraIntegration,
  IJiraIntegrationIndexed,
} from "interfaces/integration";
import { IConfig } from "interfaces/config";
import configAPI from "services/entities/config";

import ReactTooltip from "react-tooltip";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Slider from "components/forms/fields/Slider";
import Radio from "components/forms/fields/Radio";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

import { IWebhookSoftwareVulnerabilities } from "interfaces/webhook";
import { useDeepEffect } from "utilities/hooks";
import _, { size } from "lodash";

import PreviewPayloadModal from "../PreviewPayloadModal";

interface ISoftwareAutomations {
  webhook_settings: {
    vulnerabilities_webhook: IWebhookSoftwareVulnerabilities;
  };
  integrations: {
    jira: IJiraIntegration[];
  };
}

interface IManageAutomationsModalProps {
  onCancel: () => void;
  onCreateWebhookSubmit: (formData: ISoftwareAutomations) => void;
  togglePreviewPayloadModal: () => void;
  showPreviewPayloadModal: boolean;
  softwareVulnerabilityAutomationEnabled?: boolean;
  softwareVulnerabilityWebhookEnabled?: boolean;
  currentDestinationUrl?: string;
  recentVulnerabilityMaxAge?: number;
}

const validateWebhookURL = (url: string) => {
  const errors: { [key: string]: string } = {};

  if (url === "") {
    errors.url = "Please add a destination URL";
  }

  const valid = !size(errors);
  return { valid, errors };
};

const baseClass = "manage-automations-modal";

const ManageAutomationsModal = ({
  onCancel: onReturnToApp,
  onCreateWebhookSubmit,
  togglePreviewPayloadModal,
  showPreviewPayloadModal,
  softwareVulnerabilityAutomationEnabled,
  softwareVulnerabilityWebhookEnabled,
  currentDestinationUrl,
  recentVulnerabilityMaxAge,
}: IManageAutomationsModalProps): JSX.Element => {
  const [destinationUrl, setDestinationUrl] = useState<string>(
    currentDestinationUrl || ""
  );
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  const [
    softwareAutomationsEnabled,
    setSoftwareAutomationsEnabled,
  ] = useState<boolean>(softwareVulnerabilityAutomationEnabled || false);
  const [jiraEnabled, setJiraEnabled] = useState<boolean>(
    !softwareVulnerabilityWebhookEnabled
  );
  const [integrationsIndexed, setIntegrationsIndexed] = useState<
    IJiraIntegrationIndexed[]
  >();
  const [
    selectedIntegration,
    setSelectedIntegration,
  ] = useState<IJiraIntegration>();
  useDeepEffect(() => {
    setSoftwareAutomationsEnabled(
      softwareVulnerabilityAutomationEnabled || false
    );
  }, [softwareVulnerabilityAutomationEnabled]);

  useDeepEffect(() => {
    if (destinationUrl) {
      setErrors({});
    }
  }, [destinationUrl]);

  const { data: integrations } = useQuery<IConfig, Error, IJiraIntegration[]>(
    ["integrations"],
    () => configAPI.loadAll(),
    {
      select: (data: IConfig) => {
        return data.integrations.jira;
      },
      onSuccess: (data) => {
        if (data) {
          const addIndex = data.map((integration, index) => {
            return { ...integration, index };
          });
          setIntegrationsIndexed(addIndex);
          const currentSelectedJiraIntegration = addIndex.find(
            (integration) => {
              return integration.enable_software_vulnerabilities === true;
            }
          );
          setSelectedIntegration(currentSelectedJiraIntegration);
        }
      },
    }
  );

  const onURLChange = (value: string) => {
    setDestinationUrl(value);
  };

  const handleSaveAutomation = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const { valid: validUrl, errors: newErrors } = validateWebhookURL(
      destinationUrl
    );
    setErrors({
      ...errors,
      ...newErrors,
    });

    // Original config keys for software automation (webhook_settings, integrations)
    const configSoftwareAutomations: ISoftwareAutomations = {
      webhook_settings: {
        vulnerabilities_webhook: {
          destination_url: destinationUrl,
          enable_vulnerabilities_webhook: softwareVulnerabilityWebhookEnabled,
        },
      },
      integrations: {
        jira: integrations || [],
      },
    };

    const updateSoftwareAutomation = () => {
      if (!softwareAutomationsEnabled) {
        // set enable_vulnerabilities_webhook to false and all jira.enable_software_vulnerabilities to false
        configSoftwareAutomations.webhook_settings.vulnerabilities_webhook.enable_vulnerabilities_webhook = false;
        const disableAllJira = configSoftwareAutomations.integrations.jira.map(
          (integration) => {
            return { ...integration, enable_software_vulnerabilities: false };
          }
        );
        configSoftwareAutomations.integrations.jira = disableAllJira;
        return;
      }
      if (!jiraEnabled) {
        if (!validUrl) {
          return;
        }
        // set enable_vulnerabilities_webhook to true and all jira.enable_software_vulnerabilities to false
        configSoftwareAutomations.webhook_settings.vulnerabilities_webhook.enable_vulnerabilities_webhook = true;
        const disableAllJira = configSoftwareAutomations.integrations.jira.map(
          (integration) => {
            return {
              ...integration,
              enable_software_vulnerabilities: false,
            };
          }
        );
        configSoftwareAutomations.integrations.jira = disableAllJira;
        return;
      }
      // set enable_vulnerabilities_webhook to false and all jira.enable_software_vulnerabilities to false
      // except the one jira integration selected
      configSoftwareAutomations.webhook_settings.vulnerabilities_webhook.enable_vulnerabilities_webhook = false;
      const enableSelectedJiraIntegrationOnly = configSoftwareAutomations.integrations.jira.map(
        (integration, index) => {
          return {
            ...integration,
            enable_software_vulnerabilities:
              index === selectedIntegration?.index,
          };
        }
      );
      configSoftwareAutomations.integrations.jira = enableSelectedJiraIntegrationOnly;
    };

    updateSoftwareAutomation();
    onCreateWebhookSubmit(configSoftwareAutomations);
    onReturnToApp();
  };

  const createIntegrationDropdownOptions = () => {
    const integrationOptions = integrationsIndexed?.map((i) => {
      return {
        value: String(i.index),
        label: `${i.url} - ${i.project_key}`,
      };
    });
    return integrationOptions;
  };

  const onChangeSelectIntegration = (selectIntegrationIndex: string) => {
    const integrationWithIndex:
      | IJiraIntegrationIndexed
      | undefined = integrationsIndexed?.find(
      (integ: IJiraIntegrationIndexed) =>
        integ.index === parseInt(selectIntegrationIndex, 10)
    );
    setSelectedIntegration(integrationWithIndex);
  };

  const onRadioChange = (jira: boolean): ((evt: string) => void) => {
    return () => {
      setJiraEnabled(jira);
    };
  };

  const renderTicket = () => {
    return (
      <div className={`${baseClass}__ticket`}>
        <div className={`${baseClass}__software-automation-description`}>
          <p>
            A ticket will be created in your <b>Integration</b> if a detected
            vulnerability (CVE) was published in the last{" "}
            {recentVulnerabilityMaxAge || "30"} days.
          </p>
        </div>
        {integrationsIndexed && integrationsIndexed.length > 0 ? (
          <Dropdown
            searchable
            options={createIntegrationDropdownOptions()}
            onChange={onChangeSelectIntegration}
            placeholder={"Select Jira integration"}
            value={selectedIntegration?.index}
            label={"Integration"}
            wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--frequency`}
            hint={
              "For each new vulnerability detected, Fleet will create a ticket with a list of the affected hosts."
            }
          />
        ) : (
          <div className={`${baseClass}__no-integrations`}>
            <div>
              <b>You have no integrations.</b>
            </div>
            <div className={`${baseClass}__no-integration--cta`}>
              <Link
                to={PATHS.ADMIN_INTEGRATIONS}
                className={`${baseClass}__add-integration-link`}
              >
                <span>Add integration</span>
              </Link>
            </div>
          </div>
        )}
      </div>
    );
  };

  const renderWebhook = () => {
    return (
      <div className={`${baseClass}__webhook`}>
        <div className={`${baseClass}__software-automation-description`}>
          <p>
            A request will be sent to your configured <b>Destination URL</b> if
            a detected vulnerability (CVE) was published in the last{" "}
            {recentVulnerabilityMaxAge || "30"} days.
          </p>
        </div>
        <InputField
          inputWrapperClass={`${baseClass}__url-input`}
          name="webhook-url"
          label={"Destination URL"}
          type={"text"}
          value={destinationUrl}
          onChange={onURLChange}
          error={errors.url}
          hint={
            "For each new vulnerability detected, Fleet will send a JSON payload to this URL with a list of the affected hosts."
          }
          placeholder={"https://server.com/example"}
          tooltip="Provide a URL to deliver a webhook request to."
        />
        <Button
          type="button"
          variant="text-link"
          onClick={togglePreviewPayloadModal}
        >
          Preview payload
        </Button>
      </div>
    );
  };

  if (showPreviewPayloadModal) {
    return <PreviewPayloadModal onCancel={togglePreviewPayloadModal} />;
  }

  return (
    <Modal
      onExit={onReturnToApp}
      title={"Manage automations"}
      className={baseClass}
    >
      <div className={baseClass}>
        <div className={`${baseClass}__software-select-items`}>
          <Slider
            value={softwareAutomationsEnabled}
            onChange={() =>
              setSoftwareAutomationsEnabled(!softwareAutomationsEnabled)
            }
            inactiveText={"Vulnerability automations disabled"}
            activeText={"Vulnerability automations enabled"}
          />
        </div>
        <div className={`${baseClass}__overlay-container`}>
          <div className={`${baseClass}__software-automation-enabled`}>
            <div className={`${baseClass}__workflow`}>
              Workflow
              <Radio
                className={`${baseClass}__radio-input`}
                label={"Ticket"}
                id={"ticket-radio-btn"}
                checked={jiraEnabled}
                value={"ticket"}
                name={"ticket"}
                onChange={onRadioChange(true)}
              />
              <Radio
                className={`${baseClass}__radio-input`}
                label={"Webhook"}
                id={"webhook-radio-btn"}
                checked={!jiraEnabled}
                value={"webhook"}
                name={"webhook"}
                onChange={onRadioChange(false)}
              />
            </div>
            {jiraEnabled ? renderTicket() : renderWebhook()}
          </div>
          {!softwareAutomationsEnabled && (
            <div className={`${baseClass}__overlay`} />
          )}
        </div>
        <div className={`${baseClass}__button-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            onClick={onReturnToApp}
            variant="inverse"
          >
            Cancel
          </Button>
          <div
            data-tip
            data-for="save-automation-button"
            data-tip-disable={
              !(
                integrationsIndexed &&
                integrationsIndexed.length === 0 &&
                jiraEnabled &&
                softwareAutomationsEnabled
              )
            }
          >
            <Button
              className={`${baseClass}__btn`}
              type="submit"
              variant="brand"
              onClick={handleSaveAutomation}
              disabled={
                (softwareAutomationsEnabled &&
                  jiraEnabled &&
                  !selectedIntegration) ||
                (softwareAutomationsEnabled &&
                  !jiraEnabled &&
                  destinationUrl === "")
              }
            >
              Save
            </Button>
          </div>
          <ReactTooltip
            className={`save-automation-button-tooltip`}
            place="bottom"
            type="dark"
            effect="solid"
            backgroundColor="#3e4771"
            id="save-automation-button"
            data-html
          >
            <div
              className={`tooltip`}
              style={{ width: "152px", textAlign: "center" }}
            >
              Add an integration to create tickets for vulnerability automations
            </div>
          </ReactTooltip>
        </div>
      </div>
    </Modal>
  );
};

export default ManageAutomationsModal;

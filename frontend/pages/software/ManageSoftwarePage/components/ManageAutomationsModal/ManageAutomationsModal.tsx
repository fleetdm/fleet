import React, { useState, useEffect } from "react";
import { useQuery } from "react-query";

import { Link } from "react-router";
import PATHS from "router/paths";

import {
  IJiraIntegration,
  IZendeskIntegration,
  IIntegration,
  IIntegrations,
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
import useDeepEffect from "hooks/useDeepEffect";
import _, { size } from "lodash";

import PreviewPayloadModal from "../PreviewPayloadModal";

interface ISoftwareAutomations {
  webhook_settings: {
    vulnerabilities_webhook: IWebhookSoftwareVulnerabilities;
  };
  integrations: {
    jira: IJiraIntegration[];
    zendesk: IZendeskIntegration[];
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
  const [integrationEnabled, setIntegrationEnabled] = useState<boolean>(
    !softwareVulnerabilityWebhookEnabled
  );
  const [jiraIntegrationsIndexed, setJiraIntegrationsIndexed] = useState<
    IIntegration[]
  >();
  const [zendeskIntegrationsIndexed, setZendeskIntegrationsIndexed] = useState<
    IIntegration[]
  >();
  const [allIntegrationsIndexed, setAllIntegrationsIndexed] = useState<
    IIntegration[]
  >();
  const [
    selectedIntegration,
    setSelectedIntegration,
  ] = useState<IIntegration>();

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

  const { data: integrations } = useQuery<IConfig, Error, IIntegrations>(
    ["integrations"],
    () => configAPI.loadAll(),
    {
      select: (data: IConfig) => {
        return data.integrations;
      },
      onSuccess: (data) => {
        // Set jira and zendesk integrations
        const addJiraIndexed = data.jira
          ? data.jira.map((integration, index) => {
              return { ...integration, originalIndex: index, type: "jira" };
            })
          : [];
        setJiraIntegrationsIndexed(addJiraIndexed);
        const addZendeskIndexed = data.zendesk
          ? data.zendesk.map((integration, index) => {
              return {
                ...integration,
                originalIndex: index,
                type: "zendesk",
              };
            })
          : [];
        setZendeskIntegrationsIndexed(addZendeskIndexed);
      },
    }
  );

  useEffect(() => {
    if (jiraIntegrationsIndexed && zendeskIntegrationsIndexed) {
      const combineDataSets = jiraIntegrationsIndexed.concat(
        zendeskIntegrationsIndexed
      );
      setAllIntegrationsIndexed(
        combineDataSets?.map((integration, index) => {
          return { ...integration, dropdownIndex: index };
        })
      );
    }
  }, [
    jiraIntegrationsIndexed,
    zendeskIntegrationsIndexed,
    setAllIntegrationsIndexed,
  ]);

  useEffect(() => {
    if (allIntegrationsIndexed) {
      const currentSelectedIntegration = allIntegrationsIndexed.find(
        (integration) => {
          return integration.enable_software_vulnerabilities === true;
        }
      );
      setSelectedIntegration(currentSelectedIntegration);
    }
  }, [allIntegrationsIndexed]);

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
        jira: integrations?.jira || [],
        zendesk: integrations?.zendesk || [],
      },
    };

    const updateSoftwareAutomation = () => {
      if (!softwareAutomationsEnabled) {
        // set enable_vulnerabilities_webhook
        // jira.enable_software_vulnerabilities
        // and zendesk.enable_software_vulnerabilities to false
        configSoftwareAutomations.webhook_settings.vulnerabilities_webhook.enable_vulnerabilities_webhook = false;
        const disableAllJira = configSoftwareAutomations.integrations.jira.map(
          (integration) => {
            return { ...integration, enable_software_vulnerabilities: false };
          }
        );
        configSoftwareAutomations.integrations.jira = disableAllJira;
        const disableAllZendesk = configSoftwareAutomations.integrations.zendesk.map(
          (integration) => {
            return {
              ...integration,
              enable_software_vulnerabilities: false,
            };
          }
        );
        configSoftwareAutomations.integrations.zendesk = disableAllZendesk;
        return;
      }
      if (!integrationEnabled) {
        if (!validUrl) {
          return;
        }
        // set enable_vulnerabilities_webhook to true
        // all jira.enable_software_vulnerabilities to false
        // all zendesk.enable_software_vulnerabilities to false
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
        const disableAllZendesk = configSoftwareAutomations.integrations.zendesk.map(
          (integration) => {
            return {
              ...integration,
              enable_software_vulnerabilities: false,
            };
          }
        );
        configSoftwareAutomations.integrations.zendesk = disableAllZendesk;
        return;
      }
      // set enable_vulnerabilities_webhook to false
      // all jira.enable_software_vulnerabilities to false
      // all zendesk.enable_software_vulnerabilities to false
      // except the one integration selected
      configSoftwareAutomations.webhook_settings.vulnerabilities_webhook.enable_vulnerabilities_webhook = false;
      const enableSelectedJiraIntegrationOnly = configSoftwareAutomations.integrations.jira.map(
        (integration, index) => {
          return {
            ...integration,
            enable_software_vulnerabilities:
              selectedIntegration?.type === "jira"
                ? index === selectedIntegration?.originalIndex
                : false,
          };
        }
      );
      configSoftwareAutomations.integrations.jira = enableSelectedJiraIntegrationOnly;
      const enableSelectedZendeskIntegrationOnly = configSoftwareAutomations.integrations.zendesk.map(
        (integration, index) => {
          return {
            ...integration,
            enable_software_vulnerabilities:
              selectedIntegration?.type === "zendesk"
                ? index === selectedIntegration?.originalIndex
                : false,
          };
        }
      );
      configSoftwareAutomations.integrations.zendesk = enableSelectedZendeskIntegrationOnly;
    };

    updateSoftwareAutomation();
    onCreateWebhookSubmit(configSoftwareAutomations);
    onReturnToApp();
  };

  const createIntegrationDropdownOptions = () => {
    const integrationOptions = allIntegrationsIndexed?.map((i) => {
      return {
        value: String(i.dropdownIndex),
        label: `${i.url} - ${i.project_key || i.group_id}`,
      };
    });
    return integrationOptions;
  };

  const onChangeSelectIntegration = (selectIntegrationIndex: string) => {
    const integrationWithIndex:
      | IIntegration
      | undefined = allIntegrationsIndexed?.find(
      (integ: IIntegration) =>
        integ.dropdownIndex === parseInt(selectIntegrationIndex, 10)
    );
    setSelectedIntegration(integrationWithIndex);
  };

  const onRadioChange = (
    enableIntegration: boolean
  ): ((evt: string) => void) => {
    return () => {
      setIntegrationEnabled(enableIntegration);
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
        {(jiraIntegrationsIndexed && jiraIntegrationsIndexed.length > 0) ||
        (zendeskIntegrationsIndexed &&
          zendeskIntegrationsIndexed.length > 0) ? (
          <Dropdown
            searchable
            options={createIntegrationDropdownOptions()}
            onChange={onChangeSelectIntegration}
            placeholder={"Select integration"}
            value={selectedIntegration?.dropdownIndex}
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
                checked={integrationEnabled}
                value={"ticket"}
                name={"ticket"}
                onChange={onRadioChange(true)}
              />
              <Radio
                className={`${baseClass}__radio-input`}
                label={"Webhook"}
                id={"webhook-radio-btn"}
                checked={!integrationEnabled}
                value={"webhook"}
                name={"webhook"}
                onChange={onRadioChange(false)}
              />
            </div>
            {integrationEnabled ? renderTicket() : renderWebhook()}
          </div>
          {!softwareAutomationsEnabled && (
            <div className={`${baseClass}__overlay`} />
          )}
        </div>
        <div className="modal-cta-wrap">
          <div
            data-tip
            data-for="save-automation-button"
            data-tip-disable={
              !(
                ((jiraIntegrationsIndexed &&
                  jiraIntegrationsIndexed.length === 0) ||
                  (zendeskIntegrationsIndexed &&
                    zendeskIntegrationsIndexed.length === 0)) &&
                integrationEnabled &&
                softwareAutomationsEnabled
              )
            }
          >
            <Button
              type="submit"
              variant="brand"
              onClick={handleSaveAutomation}
              disabled={
                (softwareAutomationsEnabled &&
                  integrationEnabled &&
                  !selectedIntegration) ||
                (softwareAutomationsEnabled &&
                  !integrationEnabled &&
                  destinationUrl === "")
              }
            >
              Save
            </Button>
          </div>
          <ReactTooltip
            className={`save-automation-button-tooltip`}
            place="bottom"
            effect="solid"
            backgroundColor="#3e4771"
            id="save-automation-button"
            data-html
          >
            <>
              Add an integration to create
              <br /> tickets for vulnerability automations.
            </>
          </ReactTooltip>
          <Button onClick={onReturnToApp} variant="inverse">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ManageAutomationsModal;

import React, { useState, useEffect } from "react";
import { useQuery } from "react-query";

import { Link } from "react-router";
import PATHS from "router/paths";

import {
  IJiraIntegration,
  IZendeskIntegration,
  IIntegration,
  IIntegrations,
  IIntegrationType,
} from "interfaces/integration";
import {
  IConfig,
  CONFIG_DEFAULT_RECENT_VULNERABILITY_MAX_AGE_IN_DAYS,
} from "interfaces/config";
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
import CustomLink from "components/CustomLink";
import validUrl from "components/forms/validators/valid_url";

import { IWebhookSoftwareVulnerabilities } from "interfaces/webhook";
import useDeepEffect from "hooks/useDeepEffect";
import { isEmpty, omit } from "lodash";
import { COLORS } from "styles/var/colors";

import PreviewPayloadModal from "../PreviewPayloadModal";
import PreviewTicketModal from "../PreviewTicketModal";

interface ISoftwareAutomations {
  webhook_settings: {
    vulnerabilities_webhook: IWebhookSoftwareVulnerabilities;
  };
  integrations: {
    jira: IJiraIntegration[];
    zendesk: IZendeskIntegration[];
  };
}

interface IManageSoftwareAutomationsModalProps {
  onCancel: () => void;
  onCreateWebhookSubmit: (formData: ISoftwareAutomations) => void;
  togglePreviewPayloadModal: () => void;
  togglePreviewTicketModal: () => void;
  showPreviewPayloadModal: boolean;
  showPreviewTicketModal: boolean;
  softwareVulnerabilityAutomationEnabled?: boolean;
  softwareVulnerabilityWebhookEnabled?: boolean;
  currentDestinationUrl?: string;
  recentVulnerabilityMaxAge?: number;
}

const validateWebhookURL = (url: string) => {
  const errors: { [key: string]: string } = {};

  if (!url) {
    errors.url = "Please add a destination URL";
  } else if (!validUrl({ url })) {
    errors.url = `${url} is not a valid URL`;
  } else {
    delete errors.url;
  }

  return { valid: isEmpty(errors), errors };
};

const baseClass = "manage-software-automations-modal";

const ManageAutomationsModal = ({
  onCancel: onReturnToApp,
  onCreateWebhookSubmit,
  togglePreviewPayloadModal,
  togglePreviewTicketModal,
  showPreviewPayloadModal,
  showPreviewTicketModal,
  softwareVulnerabilityAutomationEnabled,
  softwareVulnerabilityWebhookEnabled,
  currentDestinationUrl,
  recentVulnerabilityMaxAge,
}: IManageSoftwareAutomationsModalProps): JSX.Element => {
  const [destinationUrl, setDestinationUrl] = useState(
    currentDestinationUrl || ""
  );
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  const [softwareAutomationsEnabled, setSoftwareAutomationsEnabled] = useState(
    softwareVulnerabilityAutomationEnabled || false
  );
  const [integrationEnabled, setIntegrationEnabled] = useState(
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
              return {
                ...integration,
                originalIndex: index,
                type: "jira" as IIntegrationType,
              };
            })
          : [];
        setJiraIntegrationsIndexed(addJiraIndexed);
        const addZendeskIndexed = data.zendesk
          ? data.zendesk.map((integration, index) => {
              return {
                ...integration,
                originalIndex: index,
                type: "zendesk" as IIntegrationType,
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

    const {
      valid: validWebhookUrl,
      errors: errorsWebhookUrl,
    } = validateWebhookURL(destinationUrl);
    if (!validWebhookUrl) {
      setErrors((prevErrs) => ({ ...prevErrs, ...errorsWebhookUrl }));
    } else {
      setErrors((prevErrs) => omit(prevErrs, "url"));
    }

    // Original config keys for software automation (webhook_settings, integrations)
    const configSoftwareAutomations: ISoftwareAutomations = {
      webhook_settings: {
        vulnerabilities_webhook: {
          destination_url: validWebhookUrl
            ? destinationUrl
            : currentDestinationUrl, // if new destination url is not valid, revert to current destination url
          enable_vulnerabilities_webhook: softwareVulnerabilityWebhookEnabled,
        },
      },
      integrations: {
        jira: integrations?.jira || [],
        zendesk: integrations?.zendesk || [],
      },
    };

    const readyForSubmission = (): boolean => {
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
        return true;
      }
      if (!integrationEnabled) {
        if (!isEmpty(errorsWebhookUrl)) {
          return false;
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
        return true;
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
      return true;
    };

    if (!readyForSubmission()) {
      return;
    }
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
      <>
        <div className={`${baseClass}__software-automation-description`}>
          A ticket will be created in your <b>Integration</b> if a detected
          vulnerability (CVE) was published in the last{" "}
          {recentVulnerabilityMaxAge ||
            CONFIG_DEFAULT_RECENT_VULNERABILITY_MAX_AGE_IN_DAYS}{" "}
          days.
        </div>
        {(jiraIntegrationsIndexed && jiraIntegrationsIndexed.length > 0) ||
        (zendeskIntegrationsIndexed &&
          zendeskIntegrationsIndexed.length > 0) ? (
          <Dropdown
            searchable
            options={createIntegrationDropdownOptions()}
            onChange={onChangeSelectIntegration}
            placeholder="Select integration"
            value={selectedIntegration?.dropdownIndex}
            label="Integration"
            wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--frequency`}
            helpText="For each new vulnerability detected, Fleet will create a ticket with a list of the affected hosts."
          />
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
        )}
        {!!selectedIntegration && (
          <Button
            type="button"
            variant="text-link"
            onClick={togglePreviewTicketModal}
          >
            Preview ticket
          </Button>
        )}
      </>
    );
  };

  const renderWebhook = () => {
    return (
      <>
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
          label="Destination URL"
          type="text"
          value={destinationUrl}
          onChange={onURLChange}
          error={errors.url}
          helpText={
            "For each new vulnerability detected, Fleet will send a JSON payload to this URL with a list of the affected hosts."
          }
          placeholder="https://server.com/example"
          tooltip="Provide a URL to deliver a webhook request to."
        />
        <Button
          type="button"
          variant="text-link"
          onClick={togglePreviewPayloadModal}
        >
          Preview payload
        </Button>
      </>
    );
  };

  if (showPreviewTicketModal && selectedIntegration?.type) {
    return (
      <PreviewTicketModal
        integrationType={selectedIntegration.type}
        onCancel={togglePreviewTicketModal}
      />
    );
  }

  if (showPreviewPayloadModal) {
    return <PreviewPayloadModal onCancel={togglePreviewPayloadModal} />;
  }

  return (
    <Modal
      onExit={onReturnToApp}
      title="Manage automations"
      className={baseClass}
      width="large"
    >
      <div className={`${baseClass} form`}>
        <Slider
          value={softwareAutomationsEnabled}
          onChange={() =>
            setSoftwareAutomationsEnabled(!softwareAutomationsEnabled)
          }
          inactiveText="Vulnerability automations disabled"
          activeText="Vulnerability automations enabled"
        />
        <div
          className={`form ${baseClass}__software-automations${
            softwareAutomationsEnabled ? "" : "__disabled"
          }`}
        >
          <div className="form-field">
            <div className="form-field__label">Workflow</div>
            <Radio
              className={`${baseClass}__radio-input`}
              label="Ticket"
              id="ticket-radio-btn"
              checked={integrationEnabled}
              value="ticket"
              name="ticket"
              onChange={onRadioChange(true)}
            />
            <Radio
              className={`${baseClass}__radio-input`}
              label="Webhook"
              id="webhook-radio-btn"
              checked={!integrationEnabled}
              value="webhook"
              name="webhook"
              onChange={onRadioChange(false)}
            />
          </div>
          {integrationEnabled ? renderTicket() : renderWebhook()}
          <p>
            Vulnerability automations currently run for software
            vulnerabilities. Interested in automations for OS vulnerabilities?{" "}
            <CustomLink
              url="https://www.fleetdm.com/support"
              text="Let us know"
              newTab
            />
          </p>
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
            backgroundColor={COLORS["tooltip-bg"]}
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

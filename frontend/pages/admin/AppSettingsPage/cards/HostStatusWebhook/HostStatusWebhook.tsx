import React, { useState, useEffect } from "react";
import { syntaxHighlight } from "utilities/helpers";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Modal from "components/Modal";
import {
  IAppConfigFormProps,
  IFormField,
  IAppConfigFormErrors,
  percentageOfHosts,
  numberOfDays,
  hostStatusPreview,
} from "../constants";

const baseClass = "app-config-form";

const HostStatusWebhook = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [
    showHostStatusWebhookPreviewModal,
    setShowHostStatusWebhookPreviewModal,
  ] = useState<boolean>(false);
  const [formData, setFormData] = useState<any>({
    enableHostStatusWebhook:
      appConfig.webhook_settings.host_status_webhook
        .enable_host_status_webhook || false,
    hostStatusWebhookDestinationURL:
      appConfig.webhook_settings.host_status_webhook.destination_url || "",
    hostStatusWebhookHostPercentage:
      appConfig.webhook_settings.host_status_webhook.host_percentage ||
      undefined,
    hostStatusWebhookDaysCount:
      appConfig.webhook_settings.host_status_webhook.days_count || undefined,
  });

  const {
    enableHostStatusWebhook,
    hostStatusWebhookDestinationURL,
    hostStatusWebhookHostPercentage,
    hostStatusWebhookDaysCount,
  } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const validateForm = () => {
    const errors: IAppConfigFormErrors = {};

    if (enableHostStatusWebhook && !hostStatusWebhookDestinationURL) {
      errors.destination_url = "Destination URL must be present";
    }

    setFormErrors(errors);
  };

  useEffect(() => {
    validateForm();
  }, [enableHostStatusWebhook]);

  const toggleHostStatusWebhookPreviewModal = () => {
    setShowHostStatusWebhookPreviewModal(!showHostStatusWebhookPreviewModal);
    return false;
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI
    const formDataToSubmit = {
      webhook_settings: {
        host_status_webhook: {
          enable_host_status_webhook: enableHostStatusWebhook,
          destination_url: hostStatusWebhookDestinationURL,
          host_percentage: hostStatusWebhookHostPercentage,
          days_count: hostStatusWebhookDaysCount,
        },
      },
    };

    handleSubmit(formDataToSubmit);
  };

  const renderHostStatusWebhookPreviewModal = () => {
    if (!showHostStatusWebhookPreviewModal) {
      return null;
    }

    return (
      <Modal
        title="Host status webhook"
        onExit={toggleHostStatusWebhookPreviewModal}
        className={`${baseClass}__host-status-webhook-preview-modal`}
      >
        <>
          <p>
            An example request sent to your configured <b>Destination URL</b>.
          </p>
          <div className={`${baseClass}__host-status-webhook-preview`}>
            <pre
              dangerouslySetInnerHTML={{
                __html: syntaxHighlight(hostStatusPreview),
              }}
            />
          </div>
          <div className="flex-end">
            <Button type="button" onClick={toggleHostStatusWebhookPreviewModal}>
              Done
            </Button>
          </div>
        </>
      </Modal>
    );
  };

  return (
    <>
      <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
        <div className={`${baseClass}__section`}>
          <h2>Host status webhook</h2>
          <div className={`${baseClass}__host-status-webhook`}>
            <p className={`${baseClass}__section-description`}>
              Send an alert if a portion of your hosts go offline.
            </p>
            <Checkbox
              onChange={handleInputChange}
              name="enableHostStatusWebhook"
              value={enableHostStatusWebhook}
              parseTarget
            >
              Enable host status webhook
            </Checkbox>
            <p className={`${baseClass}__section-description`}>
              A request will be sent to your configured <b>Destination URL</b>{" "}
              if the configured <b>Percentage of hosts</b> have not checked into
              Fleet for the configured <b>Number of days</b>.
            </p>
          </div>
          <div className={`${baseClass}__inputs ${baseClass}__inputs--webhook`}>
            <Button
              type="button"
              variant="inverse"
              onClick={toggleHostStatusWebhookPreviewModal}
            >
              Preview request
            </Button>
          </div>
          <div className={`${baseClass}__inputs`}>
            <InputField
              placeholder="https://server.com/example"
              label="Destination URL"
              onChange={handleInputChange}
              name="hostStatusWebhookDestinationURL"
              value={hostStatusWebhookDestinationURL}
              parseTarget
              onBlur={validateForm}
              error={formErrors.destination_url}
              tooltip={
                "\
                  <p>Provide a URL to deliver <br/>the webhook request to.</p>\
                "
              }
            />
          </div>
          <div className={`${baseClass}__inputs ${baseClass}__host-percentage`}>
            <Dropdown
              label="Percentage of hosts"
              options={percentageOfHosts}
              onChange={handleInputChange}
              name="hostStatusWebhookHostPercentage"
              value={hostStatusWebhookHostPercentage}
              parseTarget
              tooltip={
                "\
                  <p>Select the minimum percentage of hosts that<br/>must fail to check into Fleet in order to trigger<br/>the webhook request.</p>\
                "
              }
            />
          </div>
          <div className={`${baseClass}__inputs ${baseClass}__days-count`}>
            <Dropdown
              label="Number of days"
              options={numberOfDays}
              onChange={handleInputChange}
              name="hostStatusWebhookDaysCount"
              value={hostStatusWebhookDaysCount}
              parseTarget
              tooltip={
                "\
                  <p>Select the minimum number of days that the<br/>configured <b>Percentage of hosts</b> must fail to<br/>check into Fleet in order to trigger the<br/>webhook request.</p>\
                "
              }
            />
          </div>
        </div>
        <Button
          type="submit"
          variant="brand"
          disabled={Object.keys(formErrors).length > 0}
          className="save-loading"
          spinner={isUpdatingSettings}
        >
          Save
        </Button>
      </form>
      {showHostStatusWebhookPreviewModal &&
        renderHostStatusWebhookPreviewModal()}
    </>
  );
};

export default HostStatusWebhook;

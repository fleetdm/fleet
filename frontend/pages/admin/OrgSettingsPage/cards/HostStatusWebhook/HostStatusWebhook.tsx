import React, { useState, useEffect } from "react";

import HostStatusWebhookPreviewModal from "pages/admin/components/HostStatusWebhookPreviewModal";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";
import SectionHeader from "components/SectionHeader";

import {
  IAppConfigFormProps,
  IFormField,
  IAppConfigFormErrors,
  percentageOfHosts,
  numberOfDays,
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
  ] = useState(false);
  const [formData, setFormData] = useState<any>({
    enableHostStatusWebhook:
      appConfig.webhook_settings.host_status_webhook
        .enable_host_status_webhook || false,
    hostStatusWebhookDestinationUrl:
      appConfig.webhook_settings.host_status_webhook.destination_url || "",
    hostStatusWebhookHostPercentage:
      appConfig.webhook_settings.host_status_webhook.host_percentage ||
      undefined,
    hostStatusWebhookDaysCount:
      appConfig.webhook_settings.host_status_webhook.days_count || undefined,
  });

  const {
    enableHostStatusWebhook,
    hostStatusWebhookDestinationUrl,
    hostStatusWebhookHostPercentage,
    hostStatusWebhookDaysCount,
  } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
    setFormErrors({});
  };

  const validateForm = () => {
    const errors: IAppConfigFormErrors = {};

    if (enableHostStatusWebhook) {
      if (!hostStatusWebhookDestinationUrl) {
        errors.destination_url = "Destination URL must be present";
      } else if (!validUrl({ url: hostStatusWebhookDestinationUrl })) {
        errors.server_url = `${hostStatusWebhookDestinationUrl} is not a valid URL`;
      }

      if (!hostStatusWebhookDaysCount) {
        errors.days_count = "Number of days must be present";
      }

      if (!hostStatusWebhookDaysCount) {
        errors.host_percentage = "Percentage of hosts must be present";
      }
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
          destination_url: hostStatusWebhookDestinationUrl,
          host_percentage: hostStatusWebhookHostPercentage,
          days_count: hostStatusWebhookDaysCount,
        },
      },
    };

    handleSubmit(formDataToSubmit);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <SectionHeader title="Host status webhook" />
        <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
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
          <div>
            <p className={`${baseClass}__section-description`}>
              A request will be sent to your configured <b>Destination URL</b>{" "}
              if the configured <b>Percentage of hosts</b> have not checked into
              Fleet for the configured <b>Number of days</b>.
            </p>
            <Button
              type="button"
              variant="inverse"
              onClick={toggleHostStatusWebhookPreviewModal}
            >
              Preview request
            </Button>
          </div>
          <InputField
            placeholder="https://server.com/example"
            label="Destination URL"
            onChange={handleInputChange}
            name="hostStatusWebhookDestinationUrl"
            value={hostStatusWebhookDestinationUrl}
            parseTarget
            onBlur={validateForm}
            error={formErrors.destination_url}
            tooltip={
              <>
                Provide a URL to deliver <br />
                the webhook request to.
              </>
            }
          />
          <Dropdown
            label="Percentage of hosts"
            options={percentageOfHosts}
            onChange={handleInputChange}
            name="hostStatusWebhookHostPercentage"
            value={hostStatusWebhookHostPercentage}
            parseTarget
            searchable={false}
            onBlur={validateForm}
            tooltip={
              <>
                Select the minimum percentage of hosts that
                <br />
                must fail to check into Fleet in order to trigger
                <br />
                the webhook request.
              </>
            }
          />
          <Dropdown
            label="Number of days"
            options={numberOfDays}
            onChange={handleInputChange}
            name="hostStatusWebhookDaysCount"
            value={hostStatusWebhookDaysCount}
            parseTarget
            searchable={false}
            onBlur={validateForm}
            tooltip={
              <>
                Select the minimum number of days that the
                <br />
                configured <b>Percentage of hosts</b> must fail to
                <br />
                check into Fleet in order to trigger the
                <br />
                webhook request.
              </>
            }
          />
          <Button
            type="submit"
            variant="brand"
            disabled={Object.keys(formErrors).length > 0}
            className="button-wrap"
            isLoading={isUpdatingSettings}
          >
            Save
          </Button>
        </form>
      </div>
      {showHostStatusWebhookPreviewModal && (
        <HostStatusWebhookPreviewModal
          toggleModal={toggleHostStatusWebhookPreviewModal}
        />
      )}
    </div>
  );
};

export default HostStatusWebhook;

import React, { useState, useEffect, useMemo } from "react";

import {
  HOST_STATUS_WEBHOOK_HOST_PERCENTAGE_DROPDOWN_OPTIONS,
  HOST_STATUS_WEBHOOK_WINDOW_DROPDOWN_OPTIONS,
} from "utilities/constants";

import { getCustomDropdownOptions } from "utilities/helpers";

import HostStatusWebhookPreviewModal from "pages/admin/components/HostStatusWebhookPreviewModal";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";
import SectionHeader from "components/SectionHeader";

import { IAppConfigFormProps, IFormField } from "../constants";

interface IGlobalHostStatusWebhookFormData {
  enableHostStatusWebhook: boolean;
  destination_url?: string;
  hostStatusWebhookHostPercentage: number;
  hostStatusWebhookWindow: number;
}

interface IGlobalHostStatusWebhookFormErrors {
  destination_url?: string;
}

const baseClass = "app-config-form";

const GlobalHostStatusWebhook = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [
    showHostStatusWebhookPreviewModal,
    setShowHostStatusWebhookPreviewModal,
  ] = useState(false);
  const [formData, setFormData] = useState<IGlobalHostStatusWebhookFormData>({
    enableHostStatusWebhook:
      appConfig.webhook_settings.host_status_webhook
        ?.enable_host_status_webhook || false,
    destination_url:
      appConfig.webhook_settings.host_status_webhook?.destination_url || "",
    hostStatusWebhookHostPercentage:
      appConfig.webhook_settings.host_status_webhook?.host_percentage || 1,
    hostStatusWebhookWindow:
      appConfig.webhook_settings.host_status_webhook?.days_count || 1,
  });

  const {
    enableHostStatusWebhook,
    destination_url,
    hostStatusWebhookHostPercentage,
    hostStatusWebhookWindow,
  } = formData;

  const [
    formErrors,
    setFormErrors,
  ] = useState<IGlobalHostStatusWebhookFormErrors>({});

  const onInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
    setFormErrors({});
  };

  const validateForm = () => {
    const errors: IGlobalHostStatusWebhookFormErrors = {};

    if (enableHostStatusWebhook) {
      if (!destination_url) {
        errors.destination_url = "Destination URL must be present";
      } else if (!validUrl({ url: destination_url })) {
        errors.destination_url = `${destination_url} is not a valid URL`;
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
          destination_url,
          host_percentage: hostStatusWebhookHostPercentage,
          days_count: hostStatusWebhookWindow,
        },
        failing_policies_webhook:
          appConfig.webhook_settings.failing_policies_webhook,
        vulnerabilities_webhook:
          appConfig.webhook_settings.vulnerabilities_webhook,
      },
    };

    handleSubmit(formDataToSubmit);
  };

  const percentageHostsOptions = useMemo(
    () =>
      getCustomDropdownOptions(
        HOST_STATUS_WEBHOOK_HOST_PERCENTAGE_DROPDOWN_OPTIONS,
        hostStatusWebhookHostPercentage,
        (val) => `${val}%`
      ),
    // intentionally omit dependency so options only computed initially
    []
  );

  const windowOptions = useMemo(
    () =>
      getCustomDropdownOptions(
        HOST_STATUS_WEBHOOK_WINDOW_DROPDOWN_OPTIONS,
        hostStatusWebhookWindow,
        (val) => `${val} day${val !== 1 ? "s" : ""}`
      ),
    // intentionally omit dependency so options only computed initially
    []
  );
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <SectionHeader title="Host status webhook" />
        <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
          <p className={`${baseClass}__section-description`}>
            Send an alert if a portion of your hosts go offline.
          </p>
          <Checkbox
            onChange={onInputChange}
            name="enableHostStatusWebhook"
            value={enableHostStatusWebhook}
            parseTarget
          >
            Enable host status webhook
          </Checkbox>
          <p className={`${baseClass}__section-description`}>
            A request will be sent to your configured <b>Destination URL</b> if
            the configured <b>Percentage of hosts</b> have not checked into
            Fleet for the configured <b>Number of days</b>.
          </p>
          <Button
            type="button"
            variant="text-link"
            onClick={toggleHostStatusWebhookPreviewModal}
          >
            Preview request
          </Button>
          {enableHostStatusWebhook && (
            <>
              <InputField
                placeholder="https://server.com/example"
                label="Destination URL"
                onChange={onInputChange}
                name="destination_url"
                value={destination_url}
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
                options={percentageHostsOptions}
                onChange={onInputChange}
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
                options={windowOptions}
                onChange={onInputChange}
                name="hostStatusWebhookWindow"
                value={hostStatusWebhookWindow}
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
            </>
          )}
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

export default GlobalHostStatusWebhook;

import React, { useState } from "react";

import Modal from "components/Modal";
import validURL from "components/forms/validators/valid_url";
import { IWebhookActivities } from "interfaces/webhook";
import Slider from "components/forms/fields/Slider";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";
import { syntaxHighlight } from "utilities/helpers";

const baseClass = "activity-feed-automations-modal";

export interface IAFAMFormData {
  enabled: boolean;
  url: string;
}
interface IActivityFeedAutomationsModal {
  automationSettings: IWebhookActivities;
  onSubmit: (formData: IAFAMFormData) => void;
  onExit: () => void;
  isUpdating: boolean;
}

const ActivityFeedAutomationsModal = ({
  automationSettings,
  onSubmit,
  onExit,
  isUpdating,
}: IActivityFeedAutomationsModal) => {
  const { enable_activities_webhook: enabled, destination_url: url } =
    automationSettings || {};

  const [formData, setFormData] = useState<IAFAMFormData>({
    enabled,
    url,
  });

  const [formErrors, setFormErrors] = useState<Record<string, string | null>>(
    {}
  );
  const [showExamplePayload, setShowExamplePayload] = useState(false);

  // Used on URL change only when URL error exists and always on attempting to save
  const validateForm = (newFormData: IAFAMFormData) => {
    const errors: Record<string, string> = {};
    const { url: newUrl } = newFormData;
    if (
      formData.enabled &&
      !validURL({ url: newUrl || "", protocols: ["http", "https"] })
    ) {
      const errorPrefix = newUrl ? `${newUrl} is not` : "Please enter";
      errors.url = `${errorPrefix} a valid destination URL`;
    }

    return errors;
  };

  const onFeatureEnabledChange = () => {
    const newFormData = { ...formData, enabled: !formData.enabled };

    const isDisabling = newFormData.enabled === false;

    // On disabling feature, validate URL and if an error clear input and error
    if (isDisabling) {
      const errors = validateForm(newFormData);

      if (errors.url) {
        newFormData.url = "";
        delete formErrors.url;
        setFormErrors(formErrors);
      }
      setShowExamplePayload(false);
    }

    setFormData(newFormData);
  };

  const onUrlChange = (value: string) => {
    const newFormData = { ...formData, url: value };
    // On URL change with erroneous URL, validate form
    if (formErrors.url) {
      setFormErrors(validateForm(newFormData));
    }

    setFormData(newFormData);
  };

  const renderExamplePayload = () => {
    return (
      <>
        <pre>POST https://server.com/example</pre>
        <pre
          dangerouslySetInnerHTML={{
            __html: syntaxHighlight({
              timestamp: "0000-00-00T00:00:00Z",
              host_id: 1,
              host_display_name: "Anna's MacBook Pro",
              host_serial_number: "ABCD1234567890",
              failing_policies: [
                {
                  id: 123,
                  name: "macOS - Disable guest account",
                },
              ],
            }),
          }}
        />
      </>
    );
  };

  return (
    <Modal
      className={baseClass}
      title="Manage automations"
      width="large"
      onExit={onExit}
      onEnter={() => onSubmit(formData)}
    >
      <div className={`${baseClass} form`}>
        <Slider
          value={formData.enabled}
          onChange={onFeatureEnabledChange}
          inactiveText="Disabled"
          activeText="Enabled"
        />
        <div
          className={`form ${formData.enabled ? "" : "form-fields--disabled"}`}
        >
          <InputField
            placeholder="https://server.com/example"
            label="Destination URL"
            onChange={onUrlChange}
            name="url"
            value={formData.url}
            error={formErrors.url}
            helpText="Fleet will send a JSON payload to this URL whenever a new audit activity is generated."
          />
          <RevealButton
            isShowing={showExamplePayload}
            className={`${baseClass}__show-example-payload-toggle`}
            hideText="Hide example payload"
            showText="Show example payload"
            caretPosition="after"
            onClick={() => {
              setShowExamplePayload(!showExamplePayload);
            }}
          />
          {showExamplePayload && renderExamplePayload()}
        </div>
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            variant="brand"
            onClick={onSubmit}
            className="save-loading"
            isLoading={isUpdating}
            disabled={Object.keys(formErrors).length > 0}
          >
            Save
          </Button>
          <Button onClick={onExit} variant="inverse">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ActivityFeedAutomationsModal;

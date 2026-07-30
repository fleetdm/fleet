import React, { useState } from "react";

import { IWebhookHostActivities } from "interfaces/webhook";

import Modal from "components/Modal";
import validURL from "components/forms/validators/valid_url";
import Slider from "components/forms/fields/Slider";
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import RevealButton from "components/buttons/RevealButton";

import useGitOpsMode from "hooks/useGitOpsMode";
import { syntaxHighlight } from "utilities/helpers";
import CustomLink from "components/CustomLink";

const baseClass = "host-activity-automations-modal";

export interface IHostActivityAutomationsFormData {
  enabled: boolean;
  url: string;
}

interface IHostActivityAutomationsModal {
  automationSettings?: IWebhookHostActivities | null;
  onSubmit: (formData: IHostActivityAutomationsFormData) => void;
  onExit: () => void;
  isUpdating: boolean;
}

const HostActivityAutomationsModal = ({
  automationSettings,
  onSubmit,
  onExit,
  isUpdating,
}: IHostActivityAutomationsModal) => {
  const {
    enable_host_activities_webhook: enabled = false,
    destination_url: url = "",
  } = automationSettings || {};

  const [formData, setFormData] = useState<IHostActivityAutomationsFormData>({
    enabled,
    url,
  });

  const [formErrors, setFormErrors] = useState<Record<string, string | null>>(
    {}
  );
  const [showExamplePayload, setShowExamplePayload] = useState(false);

  const { gitOpsModeEnabled } = useGitOpsMode();

  const validateForm = (newFormData: IHostActivityAutomationsFormData) => {
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
    if (formErrors.url) {
      setFormErrors(validateForm(newFormData));
    }

    setFormData(newFormData);
  };

  const onModalSubmit = () => {
    const newErrors = validateForm(formData);
    setFormErrors(newErrors);
    if (Object.keys(newErrors).length === 0) {
      onSubmit(formData);
    }
  };

  const renderExamplePayload = () => {
    return (
      <>
        <pre>POST https://server.com/example</pre>
        <pre
          dangerouslySetInnerHTML={{
            __html: syntaxHighlight({
              timestamp: "0000-00-00T00:00:00Z",
              actor_full_name: "Anna Chao",
              actor_id: 321,
              actor_email: "anna.chao@example.com",
              type: "ran_script",
              details: {
                host_id: 42,
                host_display_name: "Anna's MacBook Pro",
                script_name: "remediate.sh",
                script_execution_id: "e797d6c6-3aae-11ee-be56-0242ac120002",
                async: true,
              },
            }),
          }}
        />
        <div className="form-field__help-text">
          To see the data included in each activity, check out the documentation
          for{" "}
          {
            <CustomLink
              url="https://fleetdm.com/learn-more-about/audit-logs"
              text="audit logs"
              newTab
            />
          }
        </div>
      </>
    );
  };

  return (
    <Modal
      className={baseClass}
      title="Manage automations"
      width="large"
      onExit={onExit}
      onEnter={onModalSubmit}
    >
      <div className={`${baseClass} form`}>
        <Slider
          value={formData.enabled}
          onChange={onFeatureEnabledChange}
          inactiveText="Disabled"
          activeText="Enabled"
          disabled={gitOpsModeEnabled}
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
            helpText="Fleet will send a JSON payload to this URL whenever an activity is generated for one of this fleet's hosts."
            disabled={!formData.enabled || gitOpsModeEnabled}
          />
        </div>
        <RevealButton
          isShowing={showExamplePayload}
          className={`${baseClass}__show-example-payload-toggle`}
          hideText="Example payload"
          showText="Example payload"
          caretPosition="after"
          onClick={() => {
            setShowExamplePayload(!showExamplePayload);
          }}
        />
        {showExamplePayload && renderExamplePayload()}
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            onClick={onModalSubmit}
            className="save-loading"
            isLoading={isUpdating}
            disabled={Object.keys(formErrors).length > 0}
          >
            Save
          </Button>
          <Button onClick={onExit} variant="secondary">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default HostActivityAutomationsModal;

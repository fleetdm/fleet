import React, { useContext, useState } from "react";
import classnames from "classnames";

import configAPI from "services/entities/config";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Radio from "components/forms/fields/Radio/Radio";
import Slider from "components/forms/fields/Slider/Slider";
import Button from "components/buttons/Button/Button";
import validateUrl from "components/forms/validators/valid_url";
import ExampleWebhookUrlPayloadModal from "../ExampleWebhookUrlPayloadModal/ExampleWebhookUrlPayloadModal";

const baseClass = "end-user-migration-section";

const VOLUNTARY_MODE_DESCRIPTION =
  "The end user sees the above window when they select Migrate to Fleet in the Fleet Desktop menu. If they’re unenrolled from your old MDM, the window appears every 15 minutes.";
const FORCED_MODE_DESCRIPTION =
  "The end user sees the above window every 15 minutes.";

interface IEndUserMigrationFormData {
  isEnabled: boolean;
  mode: "voluntary" | "forced";
  webhookUrl: string;
}

const validateWebhookUrl = (val: string) => {
  return validateUrl({ url: val });
};

const EndUserMigrationSection = () => {
  const { config } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  // const [formData, setFormData] = useState<IEndUserMigrationFormData>({
  //   isEnabled: config?.mdm.macos_migration.enable || false,
  //   mode: config?.mdm.macos_migration.mode || "voluntary",
  //   webhookUrl: config?.mdm.macos_migration.webhook_url || "",
  // });
  const [formData, setFormData] = useState<IEndUserMigrationFormData>({
    isEnabled: false,
    mode: "voluntary",
    webhookUrl: "",
  });
  const [showExamplePayload, setShowExamplePayload] = useState(false);

  // we only validate this one input so just going to use simple boolean to
  // track validation. If we need to validate more inputs in the future we can
  // use a formErrors object.
  const [isValidWebhookUrl, setIsValidWebhookUrl] = useState(true);

  const toggleMigrationEnabled = () => {
    setFormData({ ...formData, isEnabled: !formData.isEnabled });
  };

  const onChangeMode = (mode: string) => {
    // typecast to "voluntary" | "forced" as we know the value is either one of those.
    // TODO: typing the radio component onChange value argument to be specific string defined
    // by the `value` prop instead of a generic string.
    const newMode = mode as "voluntary" | "forced";
    setFormData({ ...formData, mode: newMode });
  };

  const toggleExamplePayloadModal = () => {
    setShowExamplePayload(!showExamplePayload);
  };

  const onChangeWebhookUrl = (webhookUrl: string) => {
    setFormData({ ...formData, webhookUrl });
  };

  const onSubmit = async (e: React.FormEvent<SubmitEvent>) => {
    e.preventDefault();

    if (formData.isEnabled && !validateWebhookUrl(formData.webhookUrl)) {
      setIsValidWebhookUrl(false);
      return;
    }

    try {
      await configAPI.update({
        mdm: {
          macos_migration: {
            enable: formData.isEnabled,
            mode: formData.mode,
            webhook_url: formData.webhookUrl,
          },
        },
      });
      renderFlash("success", "Successfully updated end user migration!");
    } catch (err) {
      renderFlash("error", "Could not update. Please try again.");
    }
  };

  const formClasses = classnames(`${baseClass}__end-user-migration-form`, {
    disabled: !formData.isEnabled,
  });

  return (
    <div className={baseClass}>
      <h2>End user migration workflow</h2>
      <p>
        Control the end user migration workflow for hosts that automatically
        enrolled to your old MDM solution.
      </p>

      <img src="" alt="end user migration preview" />

      <form>
        <Slider
          value={formData.isEnabled}
          onChange={toggleMigrationEnabled}
          activeText="Enabled"
          inactiveText="Diabled"
          className={`${baseClass}__enabled-slider`}
        />
        <div className={formClasses}>
          <div className={`${baseClass}__mode-field`}>
            <span>Mode</span>
            <Radio
              disabled={!formData.isEnabled}
              checked={formData.mode === "voluntary"}
              value="voluntary"
              id="voluntary"
              label="Voluntary"
              onChange={onChangeMode}
              className={`${baseClass}__voluntary-radio`}
            />
            <Radio
              disabled={!formData.isEnabled}
              checked={formData.mode === "forced"}
              value="forced"
              id="forced"
              label="Forced"
              onChange={onChangeMode}
              className={`${baseClass}__forced-radio`}
            />
            <p>
              {formData.mode === "voluntary"
                ? VOLUNTARY_MODE_DESCRIPTION
                : FORCED_MODE_DESCRIPTION}
            </p>
            <p>
              To edit the organization name, avatar (logo), and contact link,
              head to the <b>Organization settings</b> &gt;{" "}
              <b>Organization info</b> page.
            </p>
          </div>

          <InputField
            disabled={!formData.isEnabled}
            name="webhook_url"
            label="Webhook URL"
            value={formData.webhookUrl}
            onChange={onChangeWebhookUrl}
            error={!isValidWebhookUrl && "Must be a valid URL."}
            hint={
              <>
                When the end users clicks <b>Start</b>, a JSON payload is sent
                to this URL if the end user is enrolled to your old MDM. Receive
                this webhook using your automation tool (ex. Tines) to unenroll
                your end users from your old MDM solution.
              </>
            }
          />
        </div>
        <Button variant="text-link" onClick={onSubmit}>
          Preview Payload
        </Button>
        <Button onClick={onSubmit}>Save</Button>
      </form>
      {showExamplePayload && (
        <ExampleWebhookUrlPayloadModal onCancel={toggleExamplePayloadModal} />
      )}
    </div>
  );
};

export default EndUserMigrationSection;

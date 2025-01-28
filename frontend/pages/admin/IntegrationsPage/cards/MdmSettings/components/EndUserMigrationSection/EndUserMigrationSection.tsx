import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import classnames from "classnames";

import isURL from "validator/lib/isURL";

import PATHS from "router/paths";

import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

import { getErrorReason } from "interfaces/errors";

import configAPI from "services/entities/config";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Radio from "components/forms/fields/Radio/Radio";
import Slider from "components/forms/fields/Slider/Slider";
import Button from "components/buttons/Button/Button";
import SectionHeader from "components/SectionHeader";
import PremiumFeatureMessage from "components/PremiumFeatureMessage/PremiumFeatureMessage";
import EmptyTable from "components/EmptyTable/EmptyTable";
import SettingsSection from "pages/admin/components/SettingsSection";

import ExampleWebhookUrlPayloadModal from "../ExampleWebhookUrlPayloadModal/ExampleWebhookUrlPayloadModal";

import MdmMigrationPreview from "../../../../../../../../assets/images/mdm-migration-preview.gif";

const baseClass = "end-user-migration-section";

const VOLUNTARY_MODE_DESCRIPTION =
  "The end user sees the above window when they select Migrate to Fleet in the Fleet Desktop menu. If they’re unenrolled from your old MDM, the window appears every 15-20 minutes.";
const FORCED_MODE_DESCRIPTION =
  "The end user sees the above window every 15-20 minutes.";

interface IEndUserMigrationFormData {
  isEnabled: boolean;
  mode: "voluntary" | "forced";
  webhookUrl: string;
}

interface IEndUserMigrationSectionProps {
  router: InjectedRouter;
}

const validateWebhookUrl = (val: string) => {
  return isURL(val, {
    protocols: ["http", "https"],
    require_protocol: true,
    require_valid_protocol: true,
  });
};

const EndUserMigrationSection = ({ router }: IEndUserMigrationSectionProps) => {
  const { config, isPremiumTier, setConfig } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const [formData, setFormData] = useState<IEndUserMigrationFormData>({
    isEnabled: config?.mdm.macos_migration.enable || false,
    mode: config?.mdm.macos_migration.mode || "voluntary",
    webhookUrl: config?.mdm.macos_migration.webhook_url || "",
  });
  const [showExamplePayload, setShowExamplePayload] = useState(false);

  // we only validate this one input so just going to use simple boolean to
  // track validation. If we need to validate more inputs in the future we can
  // use a formErrors object.
  const [isValidWebhookUrl, setIsValidWebhookUrl] = useState(true);

  const toggleExamplePayloadModal = () => {
    setShowExamplePayload(!showExamplePayload);
  };

  const toggleMigrationEnabled = () => {
    setFormData((prevFormData) => ({
      ...prevFormData,
      isEnabled: !prevFormData.isEnabled,
    }));
  };

  const onClickConnect = () => {
    router.push(PATHS.ADMIN_INTEGRATIONS_MDM);
  };

  const onChangeMode = (mode: string) => {
    // typecast to "voluntary" | "forced" as we know the value is either one of those.
    // TODO: typing the radio component onChange value argument to be specific string defined
    // by the `value` prop instead of a generic string.
    const newMode = mode as "voluntary" | "forced";
    setFormData((prevFormData) => ({ ...prevFormData, mode: newMode }));
  };

  const onChangeWebhookUrl = (webhookUrl: string) => {
    setFormData((prevFormData) => ({ ...prevFormData, webhookUrl }));
    setIsValidWebhookUrl(validateWebhookUrl(webhookUrl));
  };

  const onSubmit = async (e: React.FormEvent<SubmitEvent>) => {
    e.preventDefault();

    if (formData.isEnabled && !validateWebhookUrl(formData.webhookUrl)) {
      setIsValidWebhookUrl(false);
      return;
    }

    try {
      const updatedConfig = await configAPI.update({
        mdm: {
          macos_migration: {
            enable: formData.isEnabled,
            mode: formData.mode,
            webhook_url: formData.webhookUrl,
          },
        },
      });
      renderFlash("success", "Successfully updated end user migration!");
      setConfig(updatedConfig);
    } catch (err) {
      if (
        getErrorReason(err, {
          nameEquals: "macos_migration.webhook_url",
        })
      ) {
        setIsValidWebhookUrl(false);
        return;
      }
      renderFlash("error", "Could not update. Please try again.");
    }
  };

  const formClasses = classnames(`${baseClass}__end-user-migration-form`, {
    disabled: !formData.isEnabled,
  });

  if (!isPremiumTier) {
    return (
      <div className={baseClass}>
        <SectionHeader title="End user migration workflow" />
        <PremiumFeatureMessage />
      </div>
    );
  }

  if (!config?.mdm.apple_bm_enabled_and_configured) {
    return (
      <div className={baseClass}>
        <SectionHeader title="End user migration workflow" />
        <EmptyTable
          className={`${baseClass}__abm-connect-message`}
          header="Migration workflow for macOS hosts"
          info="Connect to Apple Business Manager to get started."
          primaryButton={
            <Button variant="brand" onClick={onClickConnect}>
              Connect
            </Button>
          }
        />
      </div>
    );
  }

  return (
    <SettingsSection className={baseClass} title="End user migration workflow">
      <form>
        <p>Control the end user migration workflow for macOS hosts.</p>
        <img
          src={MdmMigrationPreview}
          alt="end user migration preview"
          className={`${baseClass}__migration-preview`}
        />
        <Slider
          value={formData.isEnabled}
          onChange={toggleMigrationEnabled}
          activeText="Enabled"
          inactiveText="Disabled"
          className={`${baseClass}__enabled-slider`}
        />
        <div className={`form ${formClasses}`}>
          <div className={`form-field ${baseClass}__mode-field`}>
            <div className="form-field__label">Mode</div>
            <Radio
              disabled={!formData.isEnabled}
              checked={formData.mode === "voluntary"}
              value="voluntary"
              id="voluntary"
              label="Voluntary"
              onChange={onChangeMode}
              className={`${baseClass}__voluntary-radio`}
              name="mode-type"
            />
            <Radio
              disabled={!formData.isEnabled}
              checked={formData.mode === "forced"}
              value="forced"
              id="forced"
              label="Forced"
              onChange={onChangeMode}
              className={`${baseClass}__forced-radio`}
              name="mode-type"
            />
          </div>
          <p>
            {formData.mode === "voluntary"
              ? VOLUNTARY_MODE_DESCRIPTION
              : FORCED_MODE_DESCRIPTION}
          </p>
          <p>
            To edit the organization name, avatar (logo), and contact link, head
            to the <b>Organization settings</b> &gt; <b>Organization info</b>{" "}
            page.
          </p>
          <InputField
            readOnly={!formData.isEnabled}
            name="webhook_url"
            label="Webhook URL"
            value={formData.webhookUrl}
            onChange={onChangeWebhookUrl}
            error={!isValidWebhookUrl && "Must be a valid URL."}
            helpText={
              <>
                When the end users clicks <b>Start</b>, a JSON payload is sent
                to this URL if the end user is enrolled to your old MDM. Receive
                this webhook using your automation tool (ex. Tines) to unenroll
                your end users from your old MDM solution.
              </>
            }
          />
        </div>
        <Button
          className={`${baseClass}__preview-button`}
          variant="text-link"
          onClick={toggleExamplePayloadModal}
        >
          Preview payload
        </Button>
        <Button onClick={onSubmit}>Save</Button>
      </form>
      {showExamplePayload && (
        <ExampleWebhookUrlPayloadModal onCancel={toggleExamplePayloadModal} />
      )}
    </SettingsSection>
  );
};

export default EndUserMigrationSection;

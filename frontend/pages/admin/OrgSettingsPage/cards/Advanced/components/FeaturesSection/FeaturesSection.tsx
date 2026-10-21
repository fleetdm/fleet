import React from "react";

import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import SettingsSection from "pages/admin/components/SettingsSection";

import { IAdvancedSectionProps } from "../../Advanced";

const FeaturesSection = ({
  formData,
  onInputChange,
}: IAdvancedSectionProps) => {
  const { disableLiveQuery, disableScripts, disableAIFeatures } = formData;

  return (
    <SettingsSection title="Features">
      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <Checkbox
            disabled={disableChildren}
            onChange={() =>
              onInputChange({
                name: "disableLiveQuery",
                value: !disableLiveQuery,
              })
            }
            name="disableLiveQuery"
            value={!disableLiveQuery}
            labelTooltipContent={
              !disableChildren && (
                <>
                  When disabled, removes the ability to run live reports (ad hoc
                  reports executed via the UI or fleetctl).
                  <br />
                  <i>
                    (Default: <strong>On</strong>)
                  </i>
                </>
              )
            }
          >
            Live reports
          </Checkbox>
        )}
      />

      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <Checkbox
            disabled={disableChildren}
            onChange={() =>
              onInputChange({ name: "disableScripts", value: !disableScripts })
            }
            name="disableScripts"
            value={!disableScripts}
            labelTooltipContent={
              !disableChildren && (
                <>
                  Disabling script execution will block access to run scripts.
                  Scripts may still be added and removed in the UI and API.
                  Features that run scripts under-the-hood (e.g. software
                  install, lock/wipe, script-only packages) will still be
                  available.
                  <br />
                  <i>
                    (Default: <b>On</b>)
                  </i>
                </>
              )
            }
          >
            Script execution
          </Checkbox>
        )}
      />
      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <Checkbox
            disabled={disableChildren}
            onChange={() =>
              onInputChange({
                name: "disableAIFeatures",
                value: !disableAIFeatures,
              })
            }
            name="disableAIFeatures"
            value={!disableAIFeatures}
            labelTooltipContent={
              !disableChildren && (
                <>
                  When disabled, removes AI features that generate policy
                  descriptions, policy resolutions, and configuration profiles.{" "}
                  <br />
                  <i>
                    (Default: <strong>On</strong>)
                  </i>
                </>
              )
            }
            helpText="Only policy queries (SQL) and profile prompts are sent to the LLM. Fleet doesn't use this data to train models."
          >
            Generative AI
          </Checkbox>
        )}
      />
    </SettingsSection>
  );
};

export default FeaturesSection;

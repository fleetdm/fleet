import React, { useState, useEffect } from "react";

// @ts-ignore
import constructErrorString from "utilities/yaml";
import yaml from "js-yaml";

import Button from "components/buttons/Button";
// @ts-ignore
import validateYaml from "components/forms/validators/validate_yaml";

import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import YamlAce from "components/YamlAce";
import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";
import { IAppConfigFormProps, IAppConfigFormErrors } from "../constants";

const baseClass = "app-config-form";

const AgentOptions = ({
  appConfig,
  handleSubmit,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<any>({
    agentOptions: yaml.dump(appConfig.agent_options) || {},
  });

  const { agentOptions } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleAceInputChange = (value: string) => {
    setFormData({ ...formData, agentOptions: value });
  };

  const validateForm = () => {
    const errors: IAppConfigFormErrors = {};

    if (agentOptions) {
      const { error: yamlError, valid: yamlValid } = validateYaml(agentOptions);
      if (!yamlValid) {
        errors.agent_options = constructErrorString(yamlError);
      }
    }

    setFormErrors(errors);
  };

  // Validates forms when certain information is changed
  useEffect(() => {
    validateForm();
  }, [agentOptions]);

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI
    const formDataToSubmit = {
      agent_options: yaml.load(agentOptions),
    };

    handleSubmit(formDataToSubmit);
  };

  return (
    <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
      <div className={`${baseClass}__section`}>
        <h2>
          <a id="agent-options">Global agent options</a>
        </h2>
        <div className={`${baseClass}__yaml`}>
          <p className={`${baseClass}__section-description`}>
            This code will be used by osquery when it checks for configuration
            options.
            <br />
            <b>
              Changes to these configuration options will be applied to all
              hosts in your organization that do not belong to any team.
            </b>
          </p>
          <InfoBanner className={`${baseClass}__config-docs`}>
            How do global agent options interact with team-level agent
            options?&nbsp;
            <a
              href="https://fleetdm.com/docs/using-fleet/fleet-ui#configuring-agent-options"
              className={`${baseClass}__learn-more ${baseClass}__learn-more--inline`}
              target="_blank"
              rel="noopener noreferrer"
            >
              Learn more about agent options&nbsp;
              <img className="icon" src={OpenNewTabIcon} alt="open new tab" />
            </a>
          </InfoBanner>
          <p className={`${baseClass}__component-label`}>
            <b>YAML</b>
          </p>
          <YamlAce
            wrapperClassName={`${baseClass}__text-editor-wrapper`}
            onChange={handleAceInputChange}
            name="agentOptions"
            value={agentOptions}
            parseTarget
            error={formErrors.agent_options}
          />
        </div>
      </div>
      <Button
        type="submit"
        variant="brand"
        disabled={Object.keys(formErrors).length > 0}
      >
        Save
      </Button>
    </form>
  );
};

export default AgentOptions;

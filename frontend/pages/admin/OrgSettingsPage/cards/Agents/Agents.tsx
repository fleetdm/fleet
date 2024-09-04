import React, { useState, useEffect } from "react";
import yaml from "js-yaml";
import paths from "router/paths";
import { constructErrorString, agentOptionsToYaml } from "utilities/yaml";
import { EMPTY_AGENT_OPTIONS } from "utilities/constants";

import Button from "components/buttons/Button";
// @ts-ignore
import validateYaml from "components/forms/validators/validate_yaml";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import YamlAce from "components/YamlAce";
import CustomLink from "components/CustomLink";
import SectionHeader from "components/SectionHeader";

import { IAppConfigFormProps } from "../constants";

const baseClass = "app-config-form";

interface IAgentOptionsFormData {
  agentOptions?: string;
}

interface IAgentOptionsFormErrors {
  agent_options?: string | null;
}

const Agents = ({
  appConfig,
  handleSubmit,
  isPremiumTier,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const { ADMIN_TEAMS } = paths;

  const [formData, setFormData] = useState<IAgentOptionsFormData>({
    agentOptions: agentOptionsToYaml(appConfig.agent_options),
  });
  const [formErrors, setFormErrors] = useState<IAgentOptionsFormErrors>({});

  const { agentOptions } = formData;

  const handleAgentOptionsChange = (value: string) => {
    setFormData({ ...formData, agentOptions: value });
  };

  const validateForm = () => {
    const errors: IAgentOptionsFormErrors = {};

    if (agentOptions) {
      const { error: yamlError, valid: yamlValid } = validateYaml(agentOptions);
      if (!yamlValid) {
        errors.agent_options = constructErrorString(yamlError);
      }
    }

    setFormErrors(errors);
  };

  // onChange basic yaml validation only
  useEffect(() => {
    validateForm();
  }, [agentOptions]);

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI and allows empty agent options
    const formDataToSubmit: any = agentOptions
      ? {
          agent_options: yaml.load(agentOptions),
        }
      : { agent_options: EMPTY_AGENT_OPTIONS };

    handleSubmit(formDataToSubmit);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <SectionHeader title="Agent options" />
        <form onSubmit={onFormSubmit} autoComplete="off">
          <p className={`${baseClass}__section-description`}>
            Agent options configure Fleet&apos;s agent (fleetd). When you update
            agent options, they will be applied the next time a host checks in
            to Fleet.{" "}
            <CustomLink
              url="https://fleetdm.com/docs/configuration/agent-configuration"
              text="Learn more about agent options"
              newTab
              multiline
            />
          </p>
          {isPremiumTier ? (
            <InfoBanner>
              These options are not applied to hosts on a team. To update agent
              options for hosts on a team, head to the&nbsp;
              <a href={ADMIN_TEAMS}>Teams page</a>&nbsp;and select a team.
            </InfoBanner>
          ) : (
            <InfoBanner>
              Want some hosts to have different options?&nbsp;
              <CustomLink
                url="https://fleetdm.com/docs/using-fleet/teams"
                text="Learn more about teams"
                newTab
              />
            </InfoBanner>
          )}
          <YamlAce
            onChange={handleAgentOptionsChange}
            name="agentOptions"
            value={agentOptions}
            parseTarget
            error={formErrors.agent_options}
            label="YAML"
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
    </div>
  );
};

export default Agents;

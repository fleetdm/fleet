import React, { useContext, useState, useEffect } from "react";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import { InjectedRouter } from "react-router";
import yaml from "js-yaml";
import { constructErrorString, agentOptionsToYaml } from "utilities/yaml";
import { EMPTY_AGENT_OPTIONS } from "utilities/constants";

import { NotificationContext } from "context/notification";
import useTeamIdParam from "hooks/useTeamIdParam";
import { IApiError } from "interfaces/errors";
import { ITeam } from "interfaces/team";

import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import osqueryOptionsAPI from "services/entities/osquery_options";

// @ts-ignore
import validateYaml from "components/forms/validators/validate_yaml";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";
// @ts-ignore
import YamlAce from "components/YamlAce";

const baseClass = "agent-options";

interface IAgentOptionsPageProps {
  location: {
    pathname: string;
    search: string;
    hash?: string;
    query: { team_id?: string };
  };
  router: InjectedRouter;
}

const AgentOptionsPage = ({
  location,
  router,
}: IAgentOptionsPageProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const { isRouteOk, teamIdForApi } = useTeamIdParam({
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: false,
    permittedAccessByTeamRole: {
      admin: true,
      maintainer: false,
      observer: false,
      observer_plus: false,
    },
  });

  const [teamName, setTeamName] = useState("");
  const [formData, setFormData] = useState<{ agentOptions?: string }>({});
  const [formErrors, setFormErrors] = useState<any>({});
  const [isUpdatingAgentOptions, setIsUpdatingAgentOptions] = useState(false);

  const { agentOptions } = formData;

  const handlePageError = useErrorHandler();

  const {
    isFetching: isFetchingTeamOptions,
    refetch: refetchTeamOptions,
  } = useQuery<ILoadTeamResponse, Error, ITeam>(
    ["team_details", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      enabled: isRouteOk && !!teamIdForApi,
      select: (data: ILoadTeamResponse) => data.team,
      onSuccess: (data) => {
        setFormData({
          agentOptions: agentOptionsToYaml(data.agent_options),
        });
        setTeamName(data.name);
      },
      onError: (error) => handlePageError(error),
    }
  );

  const validateForm = () => {
    const errors: any = {};

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
  }, [formData]);

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    setIsUpdatingAgentOptions(true);

    // Formatting of API not UI and allows empty agent options
    const formDataToSubmit = agentOptions
      ? yaml.load(agentOptions)
      : EMPTY_AGENT_OPTIONS;

    osqueryOptionsAPI
      .updateTeam(teamIdForApi, formDataToSubmit)
      .then(() => {
        renderFlash(
          "success",
          `Successfully updated ${teamName} team agent options.`
        );
        refetchTeamOptions();
      })
      .catch((response: { data: IApiError }) => {
        console.error(response);

        const agentOptionsInvalid =
          response.data.errors[0].reason.includes("unsupported key provided") ||
          response.data.errors[0].reason.includes("invalid value type");

        return renderFlash(
          "error",
          <>
            Could not update {teamName} team agent options.{" "}
            {response.data.errors[0].reason}
            {agentOptionsInvalid && (
              <>
                <br />
                If you’re not using the latest osquery, use the fleetctl apply
                --force command to override validation.
              </>
            )}
          </>
        );
      })
      .finally(() => {
        setIsUpdatingAgentOptions(false);
      });
  };

  const handleAgentOptionsChange = (value: string) => {
    setFormData({ ...formData, agentOptions: value });
  };

  return (
    <div className={`${baseClass}`}>
      <p className={`${baseClass}__page-description`}>
        Agent options configure the osquery agent. When you update agent
        options, they will be applied the next time a host checks in to Fleet.
        <br />
        <CustomLink
          url="https://fleetdm.com/docs/configuration/configuration-files#team-agent-options"
          text="Learn more about agent options"
          newTab
          multiline
        />
      </p>
      {isFetchingTeamOptions ? (
        <Spinner />
      ) : (
        <div className={`${baseClass}__form-wrapper`}>
          <form
            className={`${baseClass}__form`}
            onSubmit={onFormSubmit}
            autoComplete="off"
          >
            <div className={`${baseClass}__btn-wrap`}>
              <p>YAML</p>
              <Button
                type="submit"
                variant="brand"
                className="save-loading"
                isLoading={isUpdatingAgentOptions}
              >
                Save options
              </Button>
            </div>
            <YamlAce
              wrapperClassName={`${baseClass}__text-editor-wrapper`}
              onChange={handleAgentOptionsChange}
              name="agentOptions"
              value={agentOptions}
              parseTarget
              error={formErrors.agent_options}
            />
          </form>
        </div>
      )}
    </div>
  );
};

export default AgentOptionsPage;

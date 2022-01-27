import React, { useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";
import yaml from "js-yaml";
import { ITeam } from "interfaces/team";
import endpoints from "fleet/endpoints";
import teamsAPI from "services/entities/teams";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
// @ts-ignore
import osqueryOptionsActions from "redux/nodes/osquery/actions";
// @ts-ignore
import validateYaml from "components/forms/validators/validate_yaml";
// @ts-ignore
import OsqueryOptionsForm from "components/forms/admin/OsqueryOptionsForm";
import InfoBanner from "components/InfoBanner/InfoBanner";
import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";

const baseClass = "agent-options";

interface IAgentOptionsPageProps {
  params: {
    team_id: string;
  };
}

interface ITeamsResponse {
  teams: ITeam[];
}

const AgentOptionsPage = ({
  params: { team_id },
}: IAgentOptionsPageProps): JSX.Element => {
  const teamIdFromURL = parseInt(team_id, 10);
  const dispatch = useDispatch();

  const [formData, setFormData] = useState<any>({});
  const handlePageError = useErrorHandler();

  useQuery<ITeamsResponse, Error, ITeam[]>(
    ["teams"],
    () => teamsAPI.loadAll(),
    {
      select: (data: ITeamsResponse) => data.teams,
      onSuccess: (data) => {
        const selected = data.find((team) => team.id === teamIdFromURL);

        if (selected) {
          setFormData({
            osquery_options: yaml.dump(selected.agent_options),
          });
        } else {
          handlePageError({ status: 404 });
        }
      },
      onError: (error) => handlePageError(error),
    }
  );

  const onSaveOsqueryOptionsFormSubmit = (updatedForm: any): void | false => {
    const { TEAMS_AGENT_OPTIONS } = endpoints;
    const { error } = validateYaml(updatedForm.osquery_options);
    if (error) {
      dispatch(renderFlash("error", error.reason));
      return false;
    }
    dispatch(
      osqueryOptionsActions.updateOsqueryOptions(
        updatedForm,
        TEAMS_AGENT_OPTIONS(teamIdFromURL)
      )
    )
      .then(() => {
        dispatch(renderFlash("success", "Successfully saved agent options"));
      })
      .catch((errors: { [key: string]: any }) => {
        dispatch(renderFlash("error", errors.stack));
      });
  };

  return (
    <div className={`${baseClass} body-wrap`}>
      <p className={`${baseClass}__page-description`}>
        This file describes options returned to osquery when it checks for
        configuration.
      </p>
      <InfoBanner className={`${baseClass}__config-docs`}>
        See Fleet documentation for an example file that includes the overrides
        option.{" "}
        <a
          href="https://github.com/fleetdm/fleet/tree/2f42c281f98e39a72ab4a5125ecd26d303a16a6b/docs/1-Using-Fleet/configuration-files#overrides-option"
          target="_blank"
          rel="noopener noreferrer"
        >
          Go to Fleet docs{" "}
          <img className="icon" src={OpenNewTabIcon} alt="open new tab" />
        </a>
      </InfoBanner>
      <div className={`${baseClass}__form-wrapper`}>
        <OsqueryOptionsForm
          formData={formData}
          handleSubmit={onSaveOsqueryOptionsFormSubmit}
        />
      </div>
    </div>
  );
};

export default AgentOptionsPage;

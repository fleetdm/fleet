import React from "react";
import { useDispatch, useSelector } from "react-redux";
import yaml from "js-yaml";
import { ITeam } from "interfaces/team";
import endpoints from "fleet/endpoints";
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

interface IRootState {
  entities: {
    teams: {
      loading: boolean;
      data: { [id: number]: ITeam };
    };
  };
}

const AgentOptionsPage = (props: IAgentOptionsPageProps): JSX.Element => {
  const {
    params: { team_id },
  } = props;
  const teamId = parseInt(team_id, 10);
  const dispatch = useDispatch();
  const team = useSelector((state: IRootState) => {
    return state.entities.teams.data[teamId];
  });

  const formData = {
    osquery_options: yaml.dump(team.agent_options),
  };

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
        TEAMS_AGENT_OPTIONS(teamId)
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

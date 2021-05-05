import React from "react";
import PropTypes from "prop-types";
import { useSelector } from "react-redux";
// import { noop } from "lodash";

import { ITeam } from "interfaces/team";
// @ts-ignore
import yaml from "js-yaml";
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
  options: any;
  dispatch: any;
  params: any;
}

const AgentOptionsPage = (props: IAgentOptionsPageProps): JSX.Element => {
  const {
    options,
    dispatch,
    params: { team_id },
  } = props;

  const formData = {
    osquery_options: yaml.dump(options),
  };

  // do we grab this or do we pass it in from teamdetailswrapper
  // commented out because it was bugging the code
  // const team = useSelector((state: IRootState) => {
  //   return state.entities.teams.data[team_id];
  // });

  console.log(team);

  const onSaveOsqueryOptionsFormSubmit = () => {
    const { error } = validateYaml(formData.osquery_options);

    if (error) {
      dispatch(renderFlash("error", error));

      return false;
    }

    dispatch(osqueryOptionsActions.updateOsqueryOptions(formData))
      .then(() => {
        dispatch(renderFlash("success", "Successfully saved agent options"));

        return false;
      })
      .catch((errors: any) => {
        if (errors.base) {
          dispatch(renderFlash("error", errors.base));
        }

        return false;
      });

    return false;
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
          href="https://github.com/fleetdm/fleet/blob/master/docs/1-Using-Fleet/2-fleetctl-CLI.md#osquery-configuration-options"
          target="_blank"
          rel="noreferrer"
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

import config from "./config";
import { ITeam } from "../../../../interfaces/team";
import { addMembersFailure } from "../teams/actions";

const { actions } = config;
const { loadRequest, successAction, updateSuccess } = actions;

const transferHosts = (teamId: number, hostIds: number[]): any => {
  return (dispatch: any) => {
    dispatch(loadRequest()); // TODO: ensure works when API is implemented
    return Kolide.teams
      .transferHosts(teamId, hostIds)
      .then((res: { team: ITeam }) => {
        return dispatch(successAction(res.team, updateSuccess));
      })
      .catch((res: any) => {
        const errorsObject = formatErrorResponse(res);
        dispatch(addMembersFailure(errorsObject));
        throw errorsObject;
      });
  };
};

export default {
  ...actions,
};

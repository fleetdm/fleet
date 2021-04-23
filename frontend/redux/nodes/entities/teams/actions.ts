import { INewMembersBody, ITeam } from "interfaces/team";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Kolide from "kolide";
// @ts-ignore
import { formatErrorResponse } from "redux/nodes/entities/base/helpers";

import config from "./config";

const { actions } = config;
const { loadRequest, successAction, updateSuccess, updateFailure } = actions;

export const ADD_MEMBERS_FAILURE = "ADD_MEMBERS_FAILURE";

export const addMembersFailure = (errors: any) => {
  return {
    type: ADD_MEMBERS_FAILURE,
    payload: { errors },
  };
};

export const addMembers = (
  teamId: number,
  newMembers: INewMembersBody
): any => {
  return (dispatch: any) => {
    dispatch(loadRequest());
    return Kolide.teams
      .addMembers(teamId, newMembers)
      .then((res: { team: ITeam }) => {
        // return dispatch(addMembersSuccess(res)); // TODO: come back and figure out updating team entity.
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
  addMembers,
};

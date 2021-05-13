import { INewMembersBody, IRemoveMembersBody, ITeam } from "interfaces/team";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Kolide from "kolide";
// @ts-ignore
import { formatErrorResponse } from "redux/nodes/entities/base/helpers";

import config from "./config";

const { actions } = config;
const { loadRequest, successAction, updateSuccess } = actions;

export const ADD_MEMBERS_FAILURE = "ADD_MEMBERS_FAILURE";
export const addMembersFailure = (errors: any) => {
  return {
    type: ADD_MEMBERS_FAILURE,
    payload: { errors },
  };
};

export const REMOVE_MEMBERS_FAILURE = "REMOVE_MEMBERS_FAILURE";
export const removeMembersFailure = (errors: any) => {
  return {
    type: REMOVE_MEMBERS_FAILURE,
    payload: { errors },
  };
};

export const addMembers = (
  teamId: number,
  newMembers: INewMembersBody
): any => {
  return (dispatch: any) => {
    dispatch(loadRequest()); // TODO: figure out better way to do this. This causes page flash
    return Kolide.teams
      .addMembers(teamId, newMembers)
      .then((res: { team: ITeam }) => {
        return dispatch(successAction(res.team, updateSuccess)); // TODO: come back and figure out updating team entity.
      })
      .catch((res: any) => {
        const errorsObject = formatErrorResponse(res);
        dispatch(addMembersFailure(errorsObject));
        throw errorsObject;
      });
  };
};

export const removeMembers = (
  teamId: number,
  removedMembers: IRemoveMembersBody
) => {
  return (dispatch: any) => {
    dispatch(loadRequest()); // TODO: ensure works when API is implemented
    return Kolide.teams
      .removeMembers(teamId, removedMembers)
      .then((res: { team: ITeam }) => {
        return dispatch(successAction(res.team, updateSuccess));
      })
      .catch((res: any) => {
        const errorsObject = formatErrorResponse(res);
        dispatch(removeMembersFailure(errorsObject));
        throw errorsObject;
      });
  };
};

export const transferHosts = (teamId: number, hostIds: number[]): any => {
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

export const getEnrollSecrets = (teamId: number): any => {
  return (dispatch: any) => {
    return Kolide.teams
      .getEnrolSecrets(teamId)
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
  addMembers,
  removeMembers,
  transferHosts,
  getEnrolSecrets: getEnrollSecrets,
};

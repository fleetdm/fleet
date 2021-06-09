import { INewMembersBody, IRemoveMembersBody, ITeam } from "interfaces/team";
import { IEnrollSecret } from "interfaces/enroll_secret";
// ignore TS error for now until these are rewritten in ts.
// @ts-ignore
import Fleet from "fleet";
// @ts-ignore
import { formatErrorResponse } from "redux/nodes/entities/base/helpers";
import {
  enrollSecretSuccess,
  enrollSecretFailure,
  getEnrollSecret,
  // @ts-ignore
} from "redux/nodes/app/actions";
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
    dispatch(loadRequest());
    return Fleet.teams
      .addMembers(teamId, newMembers)
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

export const removeMembers = (
  teamId: number,
  removedMembers: IRemoveMembersBody
) => {
  return (dispatch: any) => {
    dispatch(loadRequest());
    return Fleet.teams
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
    return Fleet.teams
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

export const getEnrollSecrets = (team?: ITeam | null): any => {
  // This case happens when the 'No Team' options is selected. We want to
  // just call the default getEnrollSecret in this case
  if (team === null || team === undefined) {
    return (dispatch: any) => {
      return dispatch(getEnrollSecret());
    };
  }

  return (dispatch: any) => {
    return Fleet.teams
      .getEnrollSecrets(team.id)
      .then((secrets: IEnrollSecret[]) => {
        return dispatch(enrollSecretSuccess(secrets));
      })
      .catch((err: any) => {
        const errorsObject = formatErrorResponse(err);
        dispatch(enrollSecretFailure(err));
        throw errorsObject;
      });
  };
};

export default {
  ...actions,
  addMembers,
  removeMembers,
  transferHosts,
  getEnrollSecrets,
};

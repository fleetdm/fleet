import { ITeam } from "interfaces/team";
// @ts-ignore
// ignore TS error for now until these are rewritten in ts.
import Kolide from "kolide";
// @ts-ignore
import { formatErrorResponse } from "redux/nodes/entities/base/helpers";
import config from "./config";
import { addMembersFailure } from "../teams/actions";

const { actions } = config;
const { loadRequest, successAction, updateSuccess } = actions;

export const TRANSFER_HOSTS_FAILURE = "TRANSFER_HOSTS_FAILURE";
export const transferHostsFailure = (errors: any) => {
  return {
    type: TRANSFER_HOSTS_FAILURE,
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

const transferHosts = (teamId: number, hostIds: number[]): any => {
  return (dispatch: any) => {
    dispatch(loadRequest()); // TODO: ensure works when API is implemented
    return Kolide.hosts
      .transfer(teamId, hostIds)
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
  transferHosts,
};

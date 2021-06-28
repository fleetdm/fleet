import { ITeam } from "interfaces/team";
// @ts-ignore
// ignore TS error for now until these are rewritten in ts.
import Fleet from "fleet";
// @ts-ignore
import { formatErrorResponse } from "redux/nodes/entities/base/helpers";
import { IApiError } from "interfaces/errors";
import config from "./config";
import { addMembersFailure } from "../teams/actions";

const { actions } = config;
const { loadRequest, successAction, updateSuccess } = actions;

export const TRANSFER_HOSTS_SUCCESS = "TRANSFER_HOSTS_SUCCESS";
export const transferHostsSuccess = () => {
  return {
    type: TRANSFER_HOSTS_SUCCESS,
  };
};

export const TRANSFER_HOSTS_FAILURE = "TRANSFER_HOSTS_FAILURE";
export const transferHostsFailure = (errors: any) => {
  return {
    type: TRANSFER_HOSTS_FAILURE,
    payload: { errors },
  };
};

const transferToTeam = (teamId: number | null, hostIds: number[]): any => {
  return (dispatch: any) => {
    dispatch(loadRequest());
    return Fleet.hosts
      .transferToTeam(teamId, hostIds)
      .then(() => {
        dispatch(transferHostsSuccess());
      })
      .catch((res: IApiError) => {
        const errorsObject = formatErrorResponse(res);
        dispatch(transferHostsFailure(errorsObject));
        throw errorsObject;
      });
  };
};

const transferToTeamByFilter = (
  teamId: number | null,
  query: string,
  status: string,
  labelId: number | null
): any => {
  return (dispatch: any) => {
    dispatch(loadRequest());
    return Fleet.hosts
      .transferToTeamByFilter(teamId, query, status, labelId)
      .then(() => {
        dispatch(transferHostsSuccess());
      })
      .catch((res: IApiError) => {
        const errorsObject = formatErrorResponse(res);
        dispatch(transferHostsFailure(errorsObject));
        throw errorsObject;
      });
  };
};

export default {
  ...actions,
  transferToTeam,
  transferToTeamByFilter,
};

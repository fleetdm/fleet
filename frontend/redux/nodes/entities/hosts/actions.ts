import { IHost } from "interfaces/host";
// @ts-ignore
// ignore TS error for now until these are rewritten in ts.
import Fleet from "fleet";
// @ts-ignore
import { formatErrorResponse } from "redux/nodes/entities/base/helpers";
import { IApiError } from "interfaces/errors";
import config from "./config";

const { actions } = config;
const { loadRequest } = actions;

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

export const LOAD_PAGINATED = "LOAD_PAGINATED";
export const loadPaginated = (): any => {
  return (dispatch: any) => {
    dispatch(loadRequest());

    return Fleet.hosts;
  };
};

export const REFETCH_HOST_SUCCESS = "REFETCH_HOST_SUCCESS";
export const refetchHostSuccess = (data: any) => {
  return { type: REFETCH_HOST_SUCCESS, payload: { data } };
};

export const REFETCH_HOST_FAILURE = "REFETCH_HOST";
export const refetchHostFailure = (errors: any) => {
  return { type: REFETCH_HOST_FAILURE, payload: { errors } };
};

export const REFETCH_HOST_START = "REFETCH_HOST_START";
export const refetchHostStart = (host: IHost): any => {
  return (dispatch: any) => {
    return Fleet.hosts
      .refetch(host)
      .then((data: any) => {
        dispatch(refetchHostSuccess(data));
        return data;
      })
      .catch((errors: any) => {
        dispatch(refetchHostFailure(errors));

        throw errors;
      });
  };
};

export default {
  ...actions,
  transferToTeam,
  transferToTeamByFilter,
  refetchHostSuccess,
  refetchHostFailure,
  refetchHostStart,
};

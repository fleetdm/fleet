import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { ICommand, ICommandResult } from "interfaces/command";
import { getPathWithQueryParams } from "utilities/url";

import { PaginationParams } from "./common";

export interface IGetCommandsRequest extends PaginationParams {
  order_key?: string;
  order_direction?: "asc" | "desc";
  host_identifier?: string;
  request_type?: string;
  command_status?: string;
}

export interface IGetCommandsResponse {
  count: number | null;
  results: ICommand[];
}

export interface IGetCommandResultsResponse {
  results: ICommandResult[];
}

export interface IGetCommandResultsParams {
  command_uuid: string;
}

export interface IGetHostCommandResultsParams extends IGetCommandResultsParams {
  host_identifier: string;
}

export interface IGetHostCommandResultsQueryKey
  extends IGetHostCommandResultsParams {
  scope: "command_results";
}

export default {
  getCommands: (
    requestParams: IGetCommandsRequest
  ): Promise<IGetCommandsResponse> => {
    const { COMMANDS } = endpoints;
    const url = getPathWithQueryParams(COMMANDS, requestParams);

    return sendRequest("GET", url);
  },

  getCommandResults: (
    command_uuid: string
  ): Promise<IGetCommandResultsResponse> => {
    const { COMMANDS_RESULTS } = endpoints;
    const url = `${COMMANDS_RESULTS}?command_uuid=${command_uuid}`;
    return sendRequest("GET", url);
  },

  getHostCommandResults: ({
    host_identifier,
    command_uuid,
  }: IGetHostCommandResultsParams): Promise<IGetCommandResultsResponse> => {
    const { COMMANDS_RESULTS } = endpoints;
    const url = `${COMMANDS_RESULTS}?command_uuid=${command_uuid}&host_identifier=${host_identifier}`;
    return sendRequest("GET", url);
  },
};

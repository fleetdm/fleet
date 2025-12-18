import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { ICommand } from "interfaces/command";
import { getPathWithQueryParams } from "utilities/url";
import { createMockGetCommandsResponse } from "__mocks__/commandMock";

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

export default {
  getCommands: (
    requestParams: IGetCommandsRequest
  ): Promise<IGetCommandsResponse> => {
    const { COMMANDS } = endpoints;
    const url = getPathWithQueryParams(COMMANDS, requestParams);

    return sendRequest("GET", url);
  },

  // getCommandResults: (
  //   command_uuid: string
  // ): Promise<IGetMdmCommandResultsResponse> => {
  //   const { COMMANDS_RESULTS: MDM_COMMANDS_RESULTS } = endpoints;
  //   const url = `${MDM_COMMANDS_RESULTS}?command_uuid=${command_uuid}`;
  //   return sendRequest("GET", url);
  // },
};

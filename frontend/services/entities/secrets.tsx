import { ISecret } from "interfaces/secrets";

export interface IListSecretsRequestApiParams {
  page?: number;
  per_page?: number;
}

export interface IListSecretsResponse {
  secrets: ISecret[] | null;
  meta: {
    has_next_results: boolean;
    has_previous_results: boolean;
  };
}

export default {
  getSecrets(
    params: IListSecretsRequestApiParams
  ): Promise<IListSecretsResponse> {
    // Stubbed out for now, as the secrets endpoint is not yet implemented.
    console.log("getSecrets called with params:", params);
    return Promise.resolve({
      secrets: [
        {
          id: 1,
          name: "SOME_API_TOKEN",
          created_at: "2023-10-01T00:00:00Z",
          updated_at: "2025-08-10T00:00:00Z",
        },
        {
          id: 2,
          name: "CROWDSTRIKE_LICENSE_KEY",
          created_at: "2021-09-04T00:00:00Z",
          updated_at: "2024-10-02T00:00:00Z",
        },
      ],
      meta: {
        has_next_results: false,
        has_previous_results: false,
      },
    });
    // const { SECRETS } = endpoints;
    // const path = `${SECRETS}?${buildQueryStringFromParams({
    //   page,
    //   per_page,
    // })}`

    // return sendRequest("GET", path);
  },
};

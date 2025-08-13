import { ISecret, ISecretPayload } from "interfaces/secrets";
import { add } from "lodash";

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
  count: number;
}

let mockSecrets: ISecret[] = [
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
];

export default {
  getSecrets(
    params: IListSecretsRequestApiParams
  ): Promise<IListSecretsResponse> {
    // Stubbed out for now, as the secrets endpoint is not yet implemented.
    console.log("getSecrets called with params:", params);
    const startIndex = (params.page || 0) * (params.per_page || 1);
    const endIndex = startIndex + (params.per_page || 1);
    const hasNextResults = endIndex < mockSecrets.length;
    const hasPreviousResults = startIndex > 0;
    return Promise.resolve({
      secrets: [...mockSecrets].slice(startIndex, endIndex),
      meta: {
        has_next_results: hasNextResults,
        has_previous_results: hasPreviousResults,
      },
      count: mockSecrets.length,
    });
    // const { SECRETS } = endpoints;
    // const path = `${SECRETS}?${buildQueryStringFromParams({
    //   page,
    //   per_page,
    // })}`

    // return sendRequest("GET", path);
  },

  addSecret(secret: ISecretPayload) {
    // Stubbed out for now, as the secrets endpoint is not yet implemented.
    if (secret.name === "DUPE") {
      return Promise.reject({ status: 409, message: "Conflict" });
    }
    if (secret.name === "ERR") {
      return Promise.reject({ status: 500, message: "Internal Server Error" });
    }

    mockSecrets = [
      ...mockSecrets,
      {
        name: secret.name,
        id: mockSecrets.length + 1,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      } as ISecret,
    ];
    return Promise.resolve({});
    // const { SECRETS } = endpoints;
    // const path = `${SECRETS}?${buildQueryStringFromParams({
    //   page,
    //   per_page,
    // })}`

    // return sendRequest("GET", path);
  },

  deleteSecret(secretId: number) {
    // Stubbed out for now, as the secrets endpoint is not yet implemented.
    console.log("deleteSecret called with secretId:", secretId);
    mockSecrets = mockSecrets.filter((s) => s.id !== secretId);
    return Promise.resolve({});
  },
};

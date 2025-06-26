/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import helpers from "utilities/helpers";
import { ILabel, ILabelSummary } from "interfaces/label";
import { IDynamicLabelFormData } from "pages/labels/components/DynamicLabelForm/DynamicLabelForm";
import { IManualLabelFormData } from "pages/labels/components/ManualLabelForm/ManualLabelForm";
import { IHost } from "interfaces/host";
import { INewLabelFormData } from "pages/labels/NewLabelPage/NewLabelPage";

export interface ILabelsResponse {
  labels: ILabel[];
}

export interface ILabelsSummaryResponse {
  labels: ILabelSummary[];
}

export interface ICreateLabelResponse {
  label: ILabel;
}
export type IUpdateLabelResponse = ICreateLabelResponse;
export type IGetLabelResponse = ICreateLabelResponse;

export interface IGetHostsInLabelResponse {
  hosts: IHost[];
}

const isManualLabelFormData = (
  formData: IDynamicLabelFormData | IManualLabelFormData
): formData is IManualLabelFormData => {
  return "targetedHosts" in formData;
};

const generateUpdateLabelBody = (
  formData: IDynamicLabelFormData | IManualLabelFormData
) => {
  // TODO - handle all of this in a switch case inside the create API method
  // we need to prepare the post body for only manual labels.
  if (isManualLabelFormData(formData)) {
    return {
      name: formData.name,
      description: formData.description,
      host_ids: formData.targetedHosts.map((host) => host.id),
    };
  }
  return formData;
};

const generateCreateLabelBody = (formData: INewLabelFormData) => {
  switch (formData.type) {
    case "manual":
      return {
        name: formData.name,
        description: formData.description,
        host_ids: formData.targetedHosts.map((host) => host.id),
      };
    case "dynamic":
      return {
        name: formData.name,
        description: formData.description,
        query: formData.labelQuery,
        platform: formData.platform,
      };
    case "host-vitals":
      return {
        name: formData.name,
        description: formData.description,
        criteria: {
          vital: formData.vital,
          value: formData.vitalValue,
        },
      };
    default:
      throw new Error(`Unknown label type: ${formData.type}`);
  }
};

/** gets the custom label and returns them in case-insensitive alphabetical
 * ascending order by label name. (e.g. [A, B, C, a, b, c] => [A, a, B, b, C, c])
 */
export const getCustomLabels = <T extends { label_type: string; name: string }>(
  labels: T[]
) => {
  if (labels.length === 0) {
    return [];
  }

  return labels
    .filter((label) => label.label_type === "regular")
    .sort((a, b) => {
      // Found this technique here
      // https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/localeCompare
      // This is a case insensitive sort
      return a.name.localeCompare(b.name, undefined, {
        sensitivity: "base",
      });
    });
};

export default {
  create: (formData: INewLabelFormData): Promise<ICreateLabelResponse> => {
    const { LABELS } = endpoints;
    return sendRequest("POST", LABELS, generateCreateLabelBody(formData));
  },

  destroy: (label: ILabel) => {
    const { LABELS } = endpoints;
    const path = `${LABELS}/id/${label.id}`;

    return sendRequest("DELETE", path);
  },
  // TODO: confirm this still works
  loadAll: async (): Promise<ILabelsResponse> => {
    const { LABELS } = endpoints;

    try {
      const response = await sendRequest("GET", LABELS);
      return Promise.resolve({ labels: helpers.formatLabelResponse(response) });
    } catch (error) {
      console.error(error);
      return Promise.reject(error);
    }
  },
  summary: (): Promise<ILabelsSummaryResponse> => {
    const { LABELS_SUMMARY } = endpoints;

    return sendRequest("GET", LABELS_SUMMARY);
  },

  update: async (
    labelId: number,
    formData: IDynamicLabelFormData | IManualLabelFormData
  ): Promise<IUpdateLabelResponse> => {
    const { LABEL } = endpoints;
    const updateAttrs = generateUpdateLabelBody(formData);
    return sendRequest("PATCH", LABEL(labelId), updateAttrs);
  },

  specByName: (labelName: string) => {
    const { LABEL_SPEC_BY_NAME } = endpoints;
    const path = LABEL_SPEC_BY_NAME(labelName);
    return sendRequest("GET", path);
  },

  getLabel: (labelId: number): Promise<IGetLabelResponse> => {
    const { LABEL } = endpoints;
    return sendRequest("GET", LABEL(labelId));
  },

  getHostsInLabel: (labelId: number): Promise<IGetHostsInLabelResponse> => {
    const { LABEL_HOSTS } = endpoints;
    return sendRequest("GET", LABEL_HOSTS(labelId));
  },
};

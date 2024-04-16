/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import helpers from "utilities/helpers";
import { ILabel, ILabelSummary } from "interfaces/label";
import { IDynamicLabelFormData } from "pages/labels/components/DynamicLabelForm/DynamicLabelForm";
import { IManualLabelFormData } from "pages/labels/components/ManualLabelForm/ManualLabelForm";
import { IHost } from "interfaces/host";

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
export type IGetLabelResonse = ICreateLabelResponse;

const isManualLabelFormData = (
  formData: IDynamicLabelFormData | IManualLabelFormData
): formData is IManualLabelFormData => {
  return "targetedHosts" in formData;
};

const getUniqueHostIdentifier = (host: IHost) => {
  return host.hardware_serial || host.uuid || host.hostname;
};

const generateCreateLabelBody = (
  formData: IDynamicLabelFormData | IManualLabelFormData
) => {
  // we need to prepare the post body for only manual labels.
  if (isManualLabelFormData(formData)) {
    return {
      name: formData.name,
      description: formData.description,
      hosts: formData.targetedHosts.map((host) =>
        getUniqueHostIdentifier(host)
      ),
    };
  }
  return formData;
};

const generateUpdateLabelBody = generateCreateLabelBody;

export default {
  create: (
    formData: IDynamicLabelFormData | IManualLabelFormData
  ): Promise<ICreateLabelResponse> => {
    const { LABELS } = endpoints;
    const postBody = generateCreateLabelBody(formData);
    return sendRequest("POST", LABELS, postBody);
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

  getLabel: (labelId: number): Promise<IGetLabelResonse> => {
    const { LABEL } = endpoints;
    return sendRequest("GET", LABEL(labelId));
  },
};

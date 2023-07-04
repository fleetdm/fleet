/* eslint-disable  @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import helpers from "utilities/helpers";
import { ILabel, ILabelFormData, ILabelSummary } from "interfaces/label";

export interface ILabelsResponse {
  labels: ILabel[];
}

export interface ILabelsSummaryResponse {
  labels: ILabelSummary[];
}

export default {
  create: async (formData: ILabelFormData) => {
    const { LABELS } = endpoints;

    try {
      const { label: createdLabel } = await sendRequest(
        "POST",
        LABELS,
        formData
      );

      return {
        ...createdLabel,
        slug: helpers.labelSlug(createdLabel),
        type: "custom",
      };
    } catch (error) {
      console.error(error);
      throw error;
    }
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
    const { LABELS } = endpoints;
    const path = `${LABELS}/summary`;

    return sendRequest("GET", path);
  },
  update: async (label: ILabel, updatedAttrs: ILabel) => {
    const { LABELS } = endpoints;
    const path = `${LABELS}/${label.id}`;

    try {
      const { label: updatedLabel } = await sendRequest(
        "PATCH",
        path,
        updatedAttrs
      );
      return {
        ...updatedLabel,
        slug: helpers.labelSlug(updatedLabel),
        type: "custom",
      };
    } catch (error) {
      console.error(error);
      throw error;
    }
  },
  specByName: (labelName: string) => {
    const { LABEL_SPEC_BY_NAME } = endpoints;
    const path = LABEL_SPEC_BY_NAME(labelName);
    return sendRequest("GET", path);
  },
};

import sendRequest from "services";
import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";
import { ILabel, ILabelFormData } from "interfaces/label";

export default {
  create: async (formData: ILabelFormData) => {
    const { LABELS } = endpoints;

    const { label: createdLabel } = await sendRequest("POST", LABELS, formData);
    return {
      ...createdLabel,
      slug: helpers.labelSlug(createdLabel),
      type: "custom",
    };
  },
  destroy: (label: ILabel) => {
    const { LABELS } = endpoints;
    const path = `${LABELS}/id/${label.id}`;

    return sendRequest("DELETE", path);
  },
  loadAll: async () => {
    const { LABELS } = endpoints;

    const response = await sendRequest("GET", LABELS);
    return {labels: helpers.formatLabelResponse(response)};
  },
  update: async (label: ILabel, updatedAttrs: ILabel) => {
    const { LABELS } = endpoints;
    const path = `${LABELS}/${label.id}`;

    const { label: updatedLabel } = await sendRequest("PATCH", path, updatedAttrs);
    return {
      ...updatedLabel,
      slug: helpers.labelSlug(updatedLabel),
      type: "custom",
    };
  },
};
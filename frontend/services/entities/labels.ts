import sendRequest from "services";
import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";
<<<<<<< HEAD
import { ILabel, ILabelFormData } from "interfaces/label";

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
      throw new Error("Could not create label.");
    }
=======
import { ILabel } from "interfaces/label";

export default {
  create: async (label: ILabel) => {
    const { LABELS } = endpoints;
    const { label: createdLabel } = await sendRequest("POST", LABELS, label);
    return {
      ...createdLabel,
      slug: helpers.labelSlug(createdLabel),
      type: "custom",
    };
>>>>>>> eacfaf3d (Move pack queries before action buttons)
  },
  destroy: (label: ILabel) => {
    const { LABELS } = endpoints;
    const path = `${LABELS}/id/${label.id}`;
    return sendRequest("DELETE", path);
  },
  loadAll: async () => {
    const { LABELS } = endpoints;
<<<<<<< HEAD

    try {
      const response = await sendRequest("GET", LABELS);
      return { labels: helpers.formatLabelResponse(response) };
    } catch (error) {
      console.error(error);
      throw new Error("Could not load all labels.");
    }
  },
<<<<<<< HEAD
  update: async (label: ILabel, updatedAttrs: ILabel) => {
=======
  update: (label: ILabel, updatedAttrs: ILabel) => {
>>>>>>> 87b67750 (services/entities updated with all needed EAPIs for EditPacksPage)
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
      throw new Error("Could not update label.");
    }
=======
    const response = await sendRequest("GET", LABELS);
    return { labels: helpers.formatLabelResponse(response) };
  },
  update: async (label: ILabel, updatedAttrs: ILabel) => {
    const { LABELS } = endpoints;
    const path = `${LABELS}/${label.id}`;
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
>>>>>>> eacfaf3d (Move pack queries before action buttons)
  },
};

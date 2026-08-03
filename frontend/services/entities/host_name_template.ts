import sendRequest from "services";

import endpoints from "utilities/endpoints";

const hostNameTemplateService = {
  updateHostNameTemplate: (nameTemplate: string, teamId: number) => {
    const { HOST_NAME_TEMPLATE } = endpoints;
    return sendRequest("POST", HOST_NAME_TEMPLATE, {
      fleet_id: teamId,
      name_template: nameTemplate,
    });
  },
};

export default hostNameTemplateService;

import endpoints from "fleet/endpoints";
import helpers from "fleet/helpers";
import Base from "fleet/base";

export default (client) => {
  return {
    create: (formData) => {
      const { SETUP } = endpoints;
      const setupData = helpers.setupData(formData);

      return Base.post(client._endpoint(SETUP), JSON.stringify(setupData));
    },
  };
};

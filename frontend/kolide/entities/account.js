import endpoints from "kolide/endpoints";
import helpers from "kolide/helpers";
import Base from "kolide/base";

export default (client) => {
  return {
    create: (formData) => {
      const { SETUP } = endpoints;
      const setupData = helpers.setupData(formData);

      return Base.post(client._endpoint(SETUP), JSON.stringify(setupData));
    },
  };
};

import configAPI from "./config";

export default {
  deleteCertificateAuthority: (updateData: any): Promise<void> => {
    return configAPI.update(updateData);
  },
};

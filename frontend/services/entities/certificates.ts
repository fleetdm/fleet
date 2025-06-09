import configAPI from "./config";

export default {
  addCertificateAuthority: (updateData: any): Promise<any> => {
    return configAPI.update(updateData);
  },

  editCertAuthorityModal: (updateData: any): Promise<any> => {
    return configAPI.update(updateData);
  },

  deleteCertificateAuthority: (updateData: any): Promise<any> => {
    return configAPI.update(updateData);
  },
};

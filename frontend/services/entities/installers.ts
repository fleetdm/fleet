/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import { IInstallerType } from "interfaces/installer";
import sendRequest from "services";
import ENDPOINTS from "utilities/endpoints";

export interface ICheckInstallerExistenceRequestParams {
  enrollSecret: string;
  includeDesktop: boolean;
  installerType: IInstallerType;
}

export default {
  checkInstallerExistence: ({
    enrollSecret,
    includeDesktop,
    installerType,
  }: ICheckInstallerExistenceRequestParams): Promise<BlobPart> => {
    const path = `${
      ENDPOINTS.DOWNLOAD_INSTALLER
    }/${installerType}?desktop=${includeDesktop}&enroll_secret=${encodeURIComponent(
      enrollSecret
    )}`;

    return sendRequest("HEAD", path, undefined);
  },
};

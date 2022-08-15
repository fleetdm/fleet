/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import { IInstallerType } from "interfaces/installer";
import ENDPOINTS from "utilities/endpoints";
import URL_PREFIX from "router/url_prefix";

export interface IDownloadInstallerRequestParams {
  enrollSecret: string;
  includeDesktop: boolean;
  installerType: IInstallerType;
}

export default {
  downloadInstaller: ({
    enrollSecret,
    includeDesktop,
    installerType,
  }: IDownloadInstallerRequestParams) => {
    const { origin } = global.window.location;
    const url = `${origin}${URL_PREFIX}/api/${
      ENDPOINTS.DOWNLOAD_INSTALLER
    }/${installerType}?desktop=${includeDesktop}&enroll_secret=${encodeURIComponent(
      enrollSecret
    )}`;

    window.open(url);
  },
};

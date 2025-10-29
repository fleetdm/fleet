import { ISoftwareTitle } from "interfaces/software";

export const hasNoSoftwareUploaded = (
  softwareTitles?: ISoftwareTitle[] | null
) => {
  return !softwareTitles || softwareTitles.length === 0;
};

export const getInstallSoftwareDuringSetupCount = (
  softwareTitles?: ISoftwareTitle[] | null
) => {
  if (!softwareTitles) {
    return 0;
  }

  return softwareTitles.filter(
    (software) =>
      software.software_package?.install_during_setup ||
      software.app_store_app?.install_during_setup
  ).length;
};

import React, { useCallback, useMemo, useRef } from "react";

import { useQuery } from "react-query";
import { omit } from "lodash";

import paths from "router/paths";
import { Platform, PLATFORM_DISPLAY_NAMES } from "interfaces/platform";
import softwareAPI, {
  ISoftwareTitlesQueryKey,
  ISoftwareTitlesResponse,
} from "services/entities/software";
import { IPaginatedListHandle } from "components/PaginatedList";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import { getExtensionFromFileName } from "utilities/file/fileUtils";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import CustomLink from "components/CustomLink";
import {
  INSTALLABLE_SOURCE_PLATFORM_CONVERSION,
  InstallableSoftwareSource,
  ISoftwareTitle,
} from "interfaces/software";

import PoliciesPaginatedList, {
  IFormPolicy,
} from "../PoliciesPaginatedList/PoliciesPaginatedList";

const SOFTWARE_TITLE_LIST_LENGTH = 1000;

const baseClass = "install-software-modal";

const formatSoftwarePlatform = (source: InstallableSoftwareSource) => {
  return INSTALLABLE_SOURCE_PLATFORM_CONVERSION[source] || null;
};

interface ISwDropdownField {
  name: string;
  value: number;
}

export type IInstallSoftwareFormData = IFormPolicy[];

interface IEnhancedSoftwareTitle extends ISoftwareTitle {
  platform: Platform | null;
  extension?: string;
}

interface IInstallSoftwareModal {
  onExit: () => void;
  onSubmit: (formData: IInstallSoftwareFormData) => void;
  isUpdating: boolean;
  teamId: number;
  gitOpsModeEnabled?: boolean;
}

const generateSoftwareOptionHelpText = (title: IEnhancedSoftwareTitle) => {
  const vppOption = title.source === "apps" && !!title.app_store_app;
  let platformString = "";
  let versionString = "";

  if (vppOption) {
    platformString = "macOS (App Store)";
    versionString = title.app_store_app?.version || "";
  } else {
    if (title.platform && title.extension) {
      platformString = `${PLATFORM_DISPLAY_NAMES[title.platform]} (.${
        title.extension
      })`;
    }
    versionString = title.software_package?.version
      ? ` • ${title.software_package?.version}`
      : "";
  }

  return `${platformString}${versionString}`;
};

const InstallSoftwareModal = ({
  onExit,
  onSubmit,
  isUpdating,
  teamId,
  gitOpsModeEnabled = false,
}: IInstallSoftwareModal) => {
  const paginatedListRef = useRef<IPaginatedListHandle<IFormPolicy>>(null);

  const {
    data: titlesAvailableForInstall,
    isLoading: isTitlesAvailableForInstallLoading,
    isError: isTitlesAvailableForInstallError,
  } = useQuery<
    ISoftwareTitlesResponse,
    Error,
    IEnhancedSoftwareTitle[],
    [ISoftwareTitlesQueryKey]
  >(
    [
      {
        scope: "software-titles",
        page: 0,
        perPage: SOFTWARE_TITLE_LIST_LENGTH,
        query: "",
        orderDirection: "desc",
        orderKey: "hosts_count",
        teamId,
        availableForInstall: true,
        platform: "darwin,windows,linux",
      },
    ],
    ({ queryKey: [queryKey] }) =>
      softwareAPI.getSoftwareTitles(omit(queryKey, "scope")),
    {
      select: (data): IEnhancedSoftwareTitle[] =>
        data.software_titles.map((title) => {
          const extension =
            (title.software_package &&
              getExtensionFromFileName(title.software_package?.name)) ||
            undefined;

          return {
            ...title,
            platform: formatSoftwarePlatform(title.source),
            extension,
          };
        }),
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );

  const onUpdateInstallSoftware = useCallback(() => {
    if (paginatedListRef.current) {
      onSubmit(paginatedListRef.current.getDirtyItems());
    }
  }, [onSubmit]);

  const onSelectPolicySoftware = (
    item: IFormPolicy,
    { value }: ISwDropdownField
  ) => {
    // Software name needed for error message rendering
    const findSwNameById = () => {
      const foundTitle = titlesAvailableForInstall?.find(
        (title) => title.id === value
      );
      return foundTitle ? foundTitle.name : "";
    };

    return {
      ...item,
      swIdToInstall: value,
      swNameToInstall: findSwNameById(),
    };
  };

  // Filters and transforms software titles into dropdown options
  // to include only software compatible with the policy's platform(s)
  const availableSoftwareOptions = useCallback(
    (policy: IFormPolicy) => {
      const policyPlatforms = policy.platform.split(",");
      return titlesAvailableForInstall
        ?.filter(
          (title) => title.platform && policyPlatforms.includes(title.platform)
        )
        .map((title) => {
          return {
            label: title.name,
            value: title.id,
            helpText: generateSoftwareOptionHelpText(title),
          };
        });
    },
    [titlesAvailableForInstall]
  );

  // Cache availableSoftwareOptions for each unique platform
  const memoizedAvailableSoftwareOptions = useMemo(() => {
    const cache = new Map();
    return (policy: IFormPolicy) => {
      let options = availableSoftwareOptions(policy) || [];
      const installOptionsByPlatformMismatchSelectedInstaller =
        policy.swIdToInstall &&
        !options.some((opt) => opt.value === policy.swIdToInstall);

      // More unique cache key if installOptionsByPlatformMismatchSelectedInstaller
      const key = `${policy.platform}${
        installOptionsByPlatformMismatchSelectedInstaller
          ? `-${policy.swIdToInstall}`
          : ""
      }`;
      if (!cache.has(key)) {
        // Add the current software if it's not in the options
        // due to user-created a platform mismatch
        if (installOptionsByPlatformMismatchSelectedInstaller) {
          const currentSoftware = titlesAvailableForInstall?.find(
            (title) => title.id === policy.swIdToInstall
          );
          if (currentSoftware) {
            options = [
              {
                label: currentSoftware.name,
                value: currentSoftware.id,
                helpText: generateSoftwareOptionHelpText(currentSoftware),
              },
              ...options,
            ];
          }
        }

        cache.set(key, options);
      }
      return cache.get(key);
    };
  }, [availableSoftwareOptions, titlesAvailableForInstall]);

  const renderContent = () => {
    if (isTitlesAvailableForInstallError) {
      return <DataError />;
    }
    if (isTitlesAvailableForInstallLoading) {
      return <Spinner />;
    }
    if (!titlesAvailableForInstall?.length) {
      return (
        <div className={`${baseClass}__no-software`}>
          <b>No software available for install</b>
          <div>
            Go to{" "}
            <CustomLink
              url={getPathWithQueryParams(paths.SOFTWARE_TITLES, {
                team_id: teamId,
              })}
              text="Software"
            />{" "}
            to add software to this team.
          </div>
        </div>
      );
    }

    return (
      <div className={`${baseClass} form`}>
        <div className="form-field">
          <div>
            <PoliciesPaginatedList
              ref={paginatedListRef}
              isSelected="installSoftwareEnabled"
              disableSave={(changedItems) => {
                return changedItems.some(
                  (item) => item.installSoftwareEnabled && !item.swIdToInstall
                )
                  ? "Add software to all selected policies to save."
                  : false;
              }}
              onToggleItem={(item) => {
                item.installSoftwareEnabled = !item.installSoftwareEnabled;
                if (!item.installSoftwareEnabled) {
                  delete item.swIdToInstall;
                }
                return item;
              }}
              renderItemRow={(item, onChange) => {
                const formPolicy = {
                  ...item,
                  installSoftwareEnabled: !!item.swIdToInstall,
                };
                return item.installSoftwareEnabled ? (
                  <span
                    onClick={(e) => {
                      e.stopPropagation();
                    }}
                  >
                    <Dropdown
                      options={memoizedAvailableSoftwareOptions(formPolicy)} // Options filtered for policy's platform(s)
                      value={formPolicy.swIdToInstall}
                      onChange={({ value }: ISwDropdownField) =>
                        onChange(
                          onSelectPolicySoftware(item, {
                            name: formPolicy.name,
                            value,
                          })
                        )
                      }
                      placeholder="Select software"
                      className={`${baseClass}__software-dropdown`}
                      name={formPolicy.name}
                      parseTarget
                    />
                  </span>
                ) : null;
              }}
              helpText={
                <>
                  If compatible with the host, the selected software will be
                  installed when hosts fail the policy. Host counts will reset
                  when new software is selected.{" "}
                  <CustomLink
                    url="https://fleetdm.com/learn-more-about/policy-automation-install-software"
                    text="Learn more"
                    newTab
                  />
                </>
              }
              isUpdating={isUpdating}
              onSubmit={onUpdateInstallSoftware}
              onCancel={onExit}
              teamId={teamId}
            />
          </div>
        </div>
      </div>
    );
  };

  return (
    <Modal
      title="Install software"
      className={baseClass}
      onExit={onExit}
      onEnter={onUpdateInstallSoftware}
      width="large"
      isContentDisabled={isUpdating}
    >
      {renderContent()}
    </Modal>
  );
};

export default InstallSoftwareModal;

import React, { useCallback, useMemo, useState } from "react";

import { useQuery } from "react-query";
import { omit } from "lodash";

import { IPolicyStats } from "interfaces/policy";
import {
  CommaSeparatedPlatformString,
  Platform,
  PLATFORM_DISPLAY_NAMES,
} from "interfaces/platform";
import softwareAPI, {
  ISoftwareTitlesQueryKey,
  ISoftwareTitlesResponse,
} from "services/entities/software";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Modal from "components/Modal";
import DataError from "components/DataError";
import Spinner from "components/Spinner";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import CustomLink from "components/CustomLink";
import Button from "components/buttons/Button";
import {
  INSTALLABLE_SOURCE_PLATFORM_CONVERSION,
  InstallableSoftwareSource,
  ISoftwareTitle,
} from "interfaces/software";
import TooltipWrapper from "components/TooltipWrapper";

const SOFTWARE_TITLE_LIST_LENGTH = 1000;

const baseClass = "install-software-modal";

const formatSoftwarePlatform = (source: InstallableSoftwareSource) => {
  return INSTALLABLE_SOURCE_PLATFORM_CONVERSION[source] || null;
};

interface ISwDropdownField {
  name: string;
  value: number;
}
interface IFormPolicy {
  name: string;
  id: number;
  installSoftwareEnabled: boolean;
  swIdToInstall?: number;
  platform: CommaSeparatedPlatformString;
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
  policies: IPolicyStats[];
  teamId: number;
}

const InstallSoftwareModal = ({
  onExit,
  onSubmit,
  isUpdating,
  policies,
  teamId,
}: IInstallSoftwareModal) => {
  const [formData, setFormData] = useState<IInstallSoftwareFormData>(
    policies.map((policy) => ({
      name: policy.name,
      id: policy.id,
      installSoftwareEnabled: !!policy.install_software,
      swIdToInstall: policy.install_software?.software_title_id,
      platform: policy.platform,
    }))
  );

  const anyPolicyEnabledWithoutSelectedSoftware = formData.some(
    (policy) => policy.installSoftwareEnabled && !policy.swIdToInstall
  );

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
          const extension = title.software_package?.name.split(".").pop();
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
    onSubmit(formData);
  }, [formData, onSubmit]);

  const onChangeEnableInstallSoftware = useCallback(
    (newVal: { policyName: string; value: boolean }) => {
      const { policyName, value } = newVal;
      setFormData(
        formData.map((policy) => {
          if (policy.name === policyName) {
            return {
              ...policy,
              installSoftwareEnabled: value,
              swIdToInstall: value ? policy.swIdToInstall : undefined,
            };
          }
          return policy;
        })
      );
    },
    [formData]
  );

  const onSelectPolicySoftware = useCallback(
    ({ name, value }: ISwDropdownField) => {
      const [policyName, softwareId] = [name, value];
      setFormData(
        formData.map((policy) => {
          if (policy.name === policyName) {
            return { ...policy, swIdToInstall: softwareId };
          }
          return policy;
        })
      );
    },
    [formData]
  );

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
          const vppOption = title.source === "apps" && !!title.app_store_app;
          const platformString = () => {
            if (vppOption) {
              return "macOS (App Store) • ";
            }

            return title.extension
              ? `${
                  title.platform && PLATFORM_DISPLAY_NAMES[title.platform]
                } (.${title.extension}) • `
              : "";
          };
          const versionString = () => {
            return vppOption
              ? title.app_store_app?.version
              : title.software_package?.version ?? "";
          };

          return {
            label: title.name,
            value: title.id,
            helpText: `${platformString()}${versionString()}`,
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
                helpText: `${
                  currentSoftware.platform
                    ? PLATFORM_DISPLAY_NAMES[currentSoftware.platform]
                    : ""
                } • ${currentSoftware.software_package?.version || ""}`,
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

  const renderPolicySwInstallOption = (policy: IFormPolicy) => {
    const {
      name: policyName,
      id: policyId,
      installSoftwareEnabled: enabled,
      swIdToInstall,
    } = policy;

    return (
      <li
        className={`${baseClass}__policy-row policy-row`}
        id={`policy-row--${policyId}`}
        key={`${policyId}-${enabled}`} // Re-renders when modifying enabled for truncation check
      >
        <Checkbox
          value={enabled}
          name={policyName}
          onChange={() => {
            onChangeEnableInstallSoftware({
              policyName,
              value: !enabled,
            });
          }}
        >
          <TooltipTruncatedText value={policyName} />
        </Checkbox>
        {enabled && (
          <Dropdown
            options={memoizedAvailableSoftwareOptions(policy)} // Options filtered for policy's platform(s)
            value={swIdToInstall}
            onChange={onSelectPolicySoftware}
            placeholder="Select software"
            className={`${baseClass}__software-dropdown`}
            name={policyName}
            parseTarget
          />
        )}
      </li>
    );
  };

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
            Go to <a href={`/software/titles?team_id=${teamId}`}>Software</a> to
            add software to this team.
          </div>
        </div>
      );
    }

    return (
      <div className={`${baseClass} form`}>
        <div className="form-field">
          <div className="form-field__label">Policies:</div>
          <ul className="automated-policies-section">
            {formData.map((policyData) =>
              renderPolicySwInstallOption(policyData)
            )}
          </ul>
          <span className="form-field__help-text">
            If compatible with the host, the selected software will be installed
            when hosts fail the policy. App Store apps will not be installed if
            hosts are not enrolled in MDM, or if no VPP licenses are available
            for the app. If custom targets are enabled and a host with a failing
            policy is <b>not</b> targeted, installation will be skipped.{" "}
            <CustomLink
              url="https://fleetdm.com/learn-more-about/policy-automation-install-software"
              text="Learn more"
              newTab
            />
          </span>
        </div>
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            variant="brand"
            onClick={onUpdateInstallSoftware}
            className="save-loading"
            isLoading={isUpdating}
            disabled={anyPolicyEnabledWithoutSelectedSoftware}
          >
            Save
          </Button>
          <Button onClick={onExit} variant="inverse">
            Cancel
          </Button>
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

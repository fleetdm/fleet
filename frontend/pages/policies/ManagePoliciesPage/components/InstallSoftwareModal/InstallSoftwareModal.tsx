import React, { useCallback, useState } from "react";

import { useQuery } from "react-query";
import { omit } from "lodash";

import { IPolicyStats } from "interfaces/policy";
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
import { ISoftwareTitle } from "interfaces/software";
import TooltipWrapper from "components/TooltipWrapper";

const getPlatformDisplayFromPackageExtension = (ext: string | undefined) => {
  switch (ext) {
    case "pkg":
    case "zip":
    case "dmg":
      return "macOS";
    case "deb":
    case "rpm":
      return "Linux";
    case "exe":
    case "msi":
      return "Windows";
    default:
      return null;
  }
};

const AFI_SOFTWARE_BATCH_SIZE = 1000;

const baseClass = "install-software-modal";

interface ISwDropdownField {
  name: string;
  value: number;
}
interface IFormPolicy {
  name: string;
  id: number;
  installSoftwareEnabled: boolean;
  swIdToInstall?: number;
}

export type IInstallSoftwareFormData = IFormPolicy[];

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
    }))
  );

  const anyPolicyEnabledWithoutSelectedSoftware = formData.some(
    (policy) => policy.installSoftwareEnabled && !policy.swIdToInstall
  );

  const {
    data: titlesAFI,
    isLoading: isTitlesAFILoading,
    isError: isTitlesAFIError,
  } = useQuery<
    ISoftwareTitlesResponse,
    Error,
    ISoftwareTitle[],
    [ISoftwareTitlesQueryKey]
  >(
    [
      {
        scope: "software-titles",
        page: 0,
        perPage: AFI_SOFTWARE_BATCH_SIZE,
        query: "",
        orderDirection: "desc",
        orderKey: "hosts_count",
        teamId,
        availableForInstall: true,
        packagesOnly: true,
      },
    ],
    ({ queryKey: [queryKey] }) =>
      softwareAPI.getSoftwareTitles(omit(queryKey, "scope")),
    {
      select: (data) => data.software_titles,
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

  const availableSoftwareOptions = titlesAFI?.map((title) => {
    const splitName = title.software_package?.name.split(".") ?? "";
    const ext =
      splitName.length > 1 ? splitName[splitName.length - 1] : undefined;
    const platformString = ext
      ? `${getPlatformDisplayFromPackageExtension(ext)} (.${ext}) • `
      : "";
    return {
      label: title.name,
      value: title.id,
      helpText: `${platformString}${title.software_package?.version ?? ""}`,
    };
  });

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
            options={availableSoftwareOptions}
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
    if (isTitlesAFIError) {
      return <DataError />;
    }
    if (isTitlesAFILoading) {
      return <Spinner />;
    }
    if (!titlesAFI?.length) {
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

    const compatibleTipContent = (
      <>
        .pkg for macOS.
        <br />
        .msi or .exe for Windows.
        <br />
        .deb for Linux.
      </>
    );

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
            Selected software, if{" "}
            <TooltipWrapper tipContent={compatibleTipContent}>
              compatible
            </TooltipWrapper>{" "}
            with the host, will be installed when hosts fail the chosen policy.{" "}
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

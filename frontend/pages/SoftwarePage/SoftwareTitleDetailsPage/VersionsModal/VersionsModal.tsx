import React, { useMemo, useState } from "react";

import { ISoftwareTitleDetails } from "interfaces/software";
import softwareAPI from "services/entities/software";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

import { notify } from "components/ToastNotification";
import Modal from "components/Modal";
import ModalFooter from "components/ModalFooter";
import Button from "components/buttons/Button";
import Radio from "components/forms/fields/Radio";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import { deriveVersionOptions, getPreselectedVersionValue } from "./helpers";

const baseClass = "versions-modal";

/** Version-pin branch of `editSoftwarePackage`. `pinnedVersion` is sent as the
 * `version` field: "" clears the pin (Latest), else an exact or caret value. */
export interface IVersionPinFormData {
  pinnedVersion: string;
}

interface IVersionsModalProps {
  softwareTitle: ISoftwareTitleDetails;
  softwareId: number;
  teamId: number;
  refetchSoftwareTitle: () => void;
  onExit: () => void;
}

const VersionsModal = ({
  softwareTitle,
  softwareId,
  teamId,
  refetchSoftwareTitle,
  onExit,
}: IVersionsModalProps) => {
  const pkg = softwareTitle.software_package;
  const initialValue = getPreselectedVersionValue(pkg?.pinned_version);

  const options = useMemo(() => {
    const opts = deriveVersionOptions(pkg?.fleet_maintained_versions ?? []);
    // A pin no longer among the cached versions still needs an option so the
    // modal opens on the right radio. Format a caret major the same way as the
    // derived options rather than leaking "^N" into the label.
    if (initialValue && !opts.some((o) => o.value === initialValue)) {
      const label = initialValue.startsWith("^")
        ? `Pin to major version (${initialValue.slice(1)})`
        : `Pin to ${initialValue}`;
      opts.push({ value: initialValue, label });
    }
    return opts;
  }, [pkg?.fleet_maintained_versions, initialValue]);

  const [selectedValue, setSelectedValue] = useState(initialValue);
  const [isSaving, setIsSaving] = useState(false);

  const hasChanges = selectedValue !== initialValue;

  const onSave = async (evt: React.MouseEvent<HTMLButtonElement>) => {
    evt.preventDefault();
    setIsSaving(true);
    try {
      await softwareAPI.editSoftwarePackage({
        data: { pinnedVersion: selectedValue },
        softwareId,
        teamId,
      });
      notify.success(
        <>
          Successfully updated{" "}
          <b>
            {getDisplayedSoftwareName(
              softwareTitle.name,
              softwareTitle.display_name
            )}
          </b>{" "}
          version.
        </>
      );
      refetchSoftwareTitle();
      onExit();
    } catch (error) {
      notify.error("Couldn't update version. Please try again.");
      setIsSaving(false);
    }
  };

  return (
    <Modal className={baseClass} title="Versions" onExit={onExit}>
      <>
        <fieldset className={`${baseClass}__form form-field`}>
          {options.map((option) => {
            const optionId = option.value || "latest";
            return (
              <Radio
                key={optionId}
                name="versionPin"
                id={`version-pin-${optionId}`}
                label={option.label}
                value={option.value}
                checked={selectedValue === option.value}
                onChange={setSelectedValue}
              />
            );
          })}
        </fieldset>
        <ModalFooter
          primaryButtons={
            <>
              <Button onClick={onExit} variant="secondary">
                Cancel
              </Button>
              <GitOpsModeTooltipWrapper
                entityType="software"
                position="top"
                tipOffset={8}
                renderChildren={(disableChildren) => (
                  <Button
                    type="submit"
                    onClick={onSave}
                    isLoading={isSaving}
                    disabled={!hasChanges || isSaving || !!disableChildren}
                  >
                    Save
                  </Button>
                )}
              />
            </>
          }
        />
      </>
    </Modal>
  );
};

export default VersionsModal;

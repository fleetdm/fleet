import React from "react";

import RevealButton from "components/buttons/RevealButton";
import Checkbox from "components/forms/fields/Checkbox";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import FormField from "components/forms/FormField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

const baseClass = "profile-advanced-options";

const DEPLOY_OPTIONS = [
  { label: "Force install", value: "force" },
  { label: "End user initiated (manual)", value: "self_service" },
];

export interface IProfileDeployOptions {
  selfService: boolean;
  hidden: boolean;
}

interface IProfileAdvancedOptionsProps extends IProfileDeployOptions {
  isOpen: boolean;
  onToggle: () => void;
  /** Only a .mobileconfig can be self-service; anything else is force
   * installed. */
  canSelfService: boolean;
  onChange: (changes: Partial<IProfileDeployOptions>) => void;
  disabled?: boolean;
}

const ProfileAdvancedOptions = ({
  isOpen,
  onToggle,
  canSelfService,
  selfService,
  hidden,
  onChange,
  disabled = false,
}: IProfileAdvancedOptionsProps) => (
  <div className={baseClass}>
    <RevealButton
      className={`${baseClass}__toggle`}
      isShowing={isOpen}
      showText="Advanced options"
      hideText="Advanced options"
      caretPosition="after"
      onClick={onToggle}
    />
    {isOpen && (
      // One tooltip per control, to its right, so it doesn't cover the
      // fields above.
      <div className={`${baseClass}__options`}>
        {canSelfService ? (
          <GitOpsModeTooltipWrapper
            position="right"
            tipOffset={8}
            renderChildren={(disableChildren) => (
              <DropdownWrapper
                label="Deploy"
                name="deploy"
                className={`${baseClass}__deploy-dropdown`}
                options={DEPLOY_OPTIONS}
                value={selfService ? "self_service" : "force"}
                onChange={(option) => {
                  const isSelfService = option?.value === "self_service";
                  // self-service profiles can't be hidden
                  onChange({
                    selfService: isSelfService,
                    ...(isSelfService && { hidden: false }),
                  });
                }}
                isDisabled={disabled || disableChildren}
              />
            )}
          />
        ) : (
          <FormField label="Deploy" name="deploy">
            <span className={`${baseClass}__deploy-static`}>Force install</span>
          </FormField>
        )}
        <GitOpsModeTooltipWrapper
          position="right"
          tipOffset={8}
          renderChildren={(disableChildren) => (
            <Checkbox
              name="hidden"
              ariaLabel="Hide from end user"
              value={hidden}
              onChange={(value: boolean) => onChange({ hidden: value })}
              labelTooltipContent="When checked, this profile is hidden from the end user's default list of profiles in Fleet Desktop."
              disabled={disabled || disableChildren || selfService}
            >
              Hide from end user
            </Checkbox>
          )}
        />
      </div>
    )}
  </div>
);

export default ProfileAdvancedOptions;

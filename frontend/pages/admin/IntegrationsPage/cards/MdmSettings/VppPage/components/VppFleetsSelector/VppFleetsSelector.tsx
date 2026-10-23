import React, { useContext, useMemo, useState } from "react";

import CustomLink from "components/CustomLink";
import Checkbox from "components/forms/fields/Checkbox";
import Radio from "components/forms/fields/Radio";
import SearchField from "components/forms/fields/SearchField";
import Tag from "components/Tag";
import TooltipWrapper from "components/TooltipWrapper";
import { AppContext } from "context/app";
import { APP_CONTEXT_ALL_TEAMS_ID, getTeamDisplayName } from "interfaces/team";

import { IVppFleetsValue } from "./helpers";

const baseClass = "vpp-fleets-selector";

interface IVppFleetsSelectorProps {
  value: IVppFleetsValue;
  onChange: (value: IVppFleetsValue) => void;
  tooltipPosition: "top-start" | "bottom-start";
  /** Adds a "Learn more" link to the Fleets tooltip. */
  tooltipLearnMoreUrl?: string;
  /** Shows the helper text about deleting apps from unchecked fleets. */
  showUncheckHelpText?: boolean;
  disabled?: boolean;
}

const VppFleetsSelector = ({
  value,
  onChange,
  tooltipPosition,
  tooltipLearnMoreUrl,
  showUncheckHelpText = false,
  disabled = false,
}: IVppFleetsSelectorProps) => {
  const { availableTeams } = useContext(AppContext);
  const [searchQuery, setSearchQuery] = useState("");

  const fleets = useMemo(
    () =>
      (availableTeams ?? [])
        .filter((t) => t.id !== APP_CONTEXT_ALL_TEAMS_ID)
        .map((t) => ({
          id: t.id,
          label: getTeamDisplayName({ team_id: t.id, name: t.name }),
        })),
    [availableTeams]
  );

  const filteredFleets = fleets.filter((f) =>
    f.label.toLowerCase().includes(searchQuery.toLowerCase())
  );
  const selectedFleets = fleets.filter((f) => value.fleetIds.includes(f.id));

  const onToggleFleet = (id: number, checked: boolean) => {
    onChange({
      ...value,
      fleetIds: checked
        ? [...value.fleetIds, id]
        : value.fleetIds.filter((fleetId) => fleetId !== id),
    });
  };

  return (
    <div className={`${baseClass} form-field`}>
      {/* FormField's label tooltip isn't clickable, so the Learn more link
          inside it couldn't be reached. */}
      <span className="form-field__label" id={`${baseClass}-label`}>
        <TooltipWrapper
          position={tooltipPosition}
          clickable={!!tooltipLearnMoreUrl}
          tipContent={
            <>
              Choose <b>Custom</b> if you want your App Store apps from this VPP
              token to only be available to install on hosts in specific fleets.
              {tooltipLearnMoreUrl && (
                <>
                  {" "}
                  <CustomLink
                    url={tooltipLearnMoreUrl}
                    text="Learn more"
                    newTab
                  />
                </>
              )}
            </>
          }
        >
          Fleets
        </TooltipWrapper>
      </span>
      <div
        className={`${baseClass}__radios`}
        role="radiogroup"
        aria-labelledby={`${baseClass}-label`}
      >
        <Radio
          className={`${baseClass}__radio-input`}
          label="All fleets"
          id="vpp-fleets-all-radio"
          name="vpp-fleets-target"
          value="all"
          checked={value.allFleets}
          disabled={disabled}
          onChange={() => onChange({ ...value, allFleets: true })}
        />
        <Radio
          className={`${baseClass}__radio-input`}
          label="Custom"
          id="vpp-fleets-custom-radio"
          name="vpp-fleets-target"
          value="custom"
          checked={!value.allFleets}
          disabled={disabled}
          onChange={() => onChange({ ...value, allFleets: false })}
        />
      </div>
      {!value.allFleets && (
        <div className={`${baseClass}__custom`}>
          <SearchField
            placeholder="Search fleets"
            defaultValue={searchQuery}
            onChange={setSearchQuery}
            disabled={disabled}
          />
          {selectedFleets.length > 0 && (
            <div className={`${baseClass}__selected-badges`}>
              {selectedFleets.map((f) => (
                <Tag
                  key={f.id}
                  type="dismissible"
                  size="small"
                  disabled={disabled}
                  dismissLabel={`Remove ${f.label}`}
                  onDismiss={() => onToggleFleet(f.id, false)}
                >
                  {f.label}
                </Tag>
              ))}
            </div>
          )}
          <div className={`${baseClass}__checkboxes`}>
            {filteredFleets.map((f) => (
              <div className={`${baseClass}__fleet`} key={f.id}>
                <Checkbox
                  name={`vpp-fleet-${f.id}`}
                  ariaLabel={f.label}
                  value={value.fleetIds.includes(f.id)}
                  disabled={disabled}
                  onChange={({ value: checked }: { value: boolean }) =>
                    onToggleFleet(f.id, checked)
                  }
                  parseTarget
                >
                  {f.label}
                </Checkbox>
              </div>
            ))}
          </div>
          {showUncheckHelpText && (
            <span className="form-field__help-text">
              If you uncheck a fleet, apps from this VPP token will be deleted
              from that fleet. Installed apps won&apos;t be uninstalled from
              hosts.
            </span>
          )}
        </div>
      )}
    </div>
  );
};

export default VppFleetsSelector;

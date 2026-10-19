import React, { ReactNode, useState } from "react";

import Checkbox from "components/forms/fields/Checkbox";
import Radio from "components/forms/fields/Radio";
import SearchField from "components/forms/fields/SearchField";
import Tag from "components/Tag";
import TooltipWrapper from "components/TooltipWrapper";
import { APP_CONTEXT_ALL_TEAMS_ID, ITeamSummary } from "interfaces/team";

import { IVppFleetsValue, VppFleetsMode } from "./helpers";

const baseClass = "vpp-fleets-field";

interface IVppFleetsFieldProps {
  value: IVppFleetsValue;
  onChange: (value: IVppFleetsValue) => void;
  availableTeams: ITeamSummary[];
  tooltip: ReactNode;
  tooltipPosition: "top" | "bottom";
  helpText?: ReactNode;
  disabled?: boolean;
}

const VppFleetsField = ({
  value,
  onChange,
  availableTeams,
  tooltip,
  tooltipPosition,
  helpText,
  disabled = false,
}: IVppFleetsFieldProps) => {
  const [searchQuery, setSearchQuery] = useState("");

  // "Unassigned" (ID 0) is a real assignment target, "All fleets" is chosen with the radio instead.
  const teams = availableTeams.filter((t) => t.id !== APP_CONTEXT_ALL_TEAMS_ID);
  const filteredTeams = teams.filter((t) =>
    t.name.toLowerCase().includes(searchQuery.toLowerCase())
  );
  const selectedTeams = teams.filter((t) => value.teamIds.includes(t.id));

  const onSelectMode = (mode: string) => {
    onChange({ ...value, mode: mode as VppFleetsMode });
  };

  const onToggleTeam = (teamId: number, checked: boolean) => {
    const teamIds = checked
      ? [...value.teamIds, teamId]
      : value.teamIds.filter((id) => id !== teamId);
    onChange({ ...value, teamIds });
  };

  return (
    <div className={`${baseClass} form-field`}>
      <div className="form-field__label">
        <TooltipWrapper tipContent={tooltip} position={tooltipPosition}>
          Fleets
        </TooltipWrapper>
      </div>
      <div className={`${baseClass}__modes`}>
        <Radio
          label="All fleets"
          id="vpp-fleets-all"
          name="vpp-fleets-mode"
          value="all"
          checked={value.mode === "all"}
          onChange={onSelectMode}
          disabled={disabled}
        />
        <Radio
          label="Custom"
          id="vpp-fleets-custom"
          name="vpp-fleets-mode"
          value="custom"
          checked={value.mode === "custom"}
          onChange={onSelectMode}
          disabled={disabled}
        />
      </div>
      {value.mode === "custom" && (
        <div className={`${baseClass}__custom`}>
          <SearchField placeholder="Search fleets" onChange={setSearchQuery} />
          {!!selectedTeams.length && (
            <div className={`${baseClass}__selected`}>
              {selectedTeams.map((team) => (
                <Tag
                  key={team.id}
                  type="dismissible"
                  size="small"
                  disabled={disabled}
                  dismissLabel={`Remove ${team.name}`}
                  onDismiss={() => onToggleTeam(team.id, false)}
                >
                  {team.name}
                </Tag>
              ))}
            </div>
          )}
          <div className={`${baseClass}__list-wrapper`}>
            <div className={`${baseClass}__list`}>
              {filteredTeams.map((team) => (
                <div className={`${baseClass}__list-item`} key={team.id}>
                  <Checkbox
                    name={`vpp-fleet-${team.id}`}
                    ariaLabel={team.name}
                    value={value.teamIds.includes(team.id)}
                    onChange={(checked: boolean) =>
                      onToggleTeam(team.id, checked)
                    }
                    disabled={disabled}
                  >
                    {team.name}
                  </Checkbox>
                </div>
              ))}
            </div>
            {helpText && (
              <div className={`${baseClass}__help-text form-field__help-text`}>
                {helpText}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default VppFleetsField;

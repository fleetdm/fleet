import React from "react";
import { Command } from "cmdk";

import { ITeamSummary } from "interfaces/team";

import HighlightedLabel from "./HighlightedLabel";

const baseClass = "command-palette";

interface IFleetPickerProps {
  availableTeams?: ITeamSummary[];
  currentTeam?: ITeamSummary;
  search: string;
  onSelect: (fleetId: number) => void;
}

const FleetPicker = ({
  availableTeams,
  currentTeam,
  search,
  onSelect,
}: IFleetPickerProps): JSX.Element => {
  return (
    <>
      {/* cmdk's Command.Empty auto-renders when every Item is filtered out.
          Mirrors the root palette's "No results found." so the picker doesn't
          go blank on a typo. CommandPalette suppresses its own Command.Empty
          on picker pages — pickers own their empty copy. */}
      <Command.Empty className={`${baseClass}__empty`}>
        {search ? `No fleets match "${search}".` : "No fleets found."}
      </Command.Empty>
      <Command.Group className={`${baseClass}__group`}>
        {availableTeams?.map((fleet) => {
          const isSelected = fleet.id === currentTeam?.id;
          return (
            <Command.Item
              key={`fleet-${fleet.id}`}
              value={fleet.name}
              onSelect={() => onSelect(fleet.id)}
              className={`${baseClass}__item`}
            >
              <span
                className={`${baseClass}__item-label${
                  isSelected ? ` ${baseClass}__item-label--selected` : ""
                }`}
              >
                <HighlightedLabel text={fleet.name} query={search} />
              </span>
            </Command.Item>
          );
        })}
      </Command.Group>
    </>
  );
};

export default FleetPicker;

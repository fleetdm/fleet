import { xor } from "lodash";
import React, { useState } from "react";

import Chip from "components/Chip";
import Checkbox from "components/forms/fields/Checkbox";
import SearchField from "components/forms/fields/SearchField";
import { ISoftwareType } from "interfaces/software";

const baseClass = "software-types-picker";

interface ISoftwareTypesPickerProps {
  availableTypes: readonly ISoftwareType[];
  selectedKeys: string[];
  onChange: (keys: string[]) => void;
}

const SoftwareTypesPicker = ({
  availableTypes,
  selectedKeys,
  onChange,
}: ISoftwareTypesPickerProps) => {
  const [searchQuery, setSearchQuery] = useState("");

  const selected = new Set(selectedKeys);
  const selectedTypes = availableTypes.filter((t) => selected.has(t.key));
  const normalizedQuery = searchQuery.trim().toLowerCase();
  const visibleTypes = availableTypes.filter((t) =>
    t.displayName.toLowerCase().includes(normalizedQuery)
  );

  const toggleType = (key: string) => onChange(xor(selectedKeys, [key]));

  return (
    <div className={baseClass}>
      <h3 className={`${baseClass}__title`}>Types</h3>
      <SearchField placeholder="Search types" onChange={setSearchQuery} />
      {selectedTypes.length > 0 && (
        <div className={`${baseClass}__chips`}>
          {selectedTypes.map((t) => (
            <Chip
              key={t.key}
              text={t.displayName}
              trailingIcon="close"
              onClick={() => toggleType(t.key)}
            />
          ))}
        </div>
      )}
      <div className={`${baseClass}__list`}>
        {visibleTypes.map((t) => (
          <div key={t.key} className={`${baseClass}__row`}>
            <Checkbox
              name={t.displayName}
              value={selected.has(t.key)}
              onChange={() => toggleType(t.key)}
            >
              {t.displayName}
            </Checkbox>
          </div>
        ))}
        {visibleTypes.length === 0 && (
          <div className={`${baseClass}__status`}>No matching types.</div>
        )}
      </div>
    </div>
  );
};

export default SoftwareTypesPicker;

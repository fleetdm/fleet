import React, { useState } from "react";

import { IMDMPolicyCheck } from "interfaces/policy";
import {
  DEFAULT_MDM_POLICIES,
  IMDMPolicyTemplate,
} from "pages/policies/constants";

import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Icon from "components/Icon";
import Modal from "components/Modal";

const baseClass = "mdm-check-builder";

interface IMDMCheckBuilderProps {
  checks: IMDMPolicyCheck[];
  onChange: (checks: IMDMPolicyCheck[]) => void;
  disabled?: boolean;
}

interface IFieldCatalogEntry {
  field: string;
  label: string;
  type: string;
  category: string;
}

const MDM_FIELD_CATALOG: IFieldCatalogEntry[] = [
  {
    field: "OSVersion",
    label: "OS Version",
    type: "version",
    category: "Device Info",
  },
  {
    field: "DeviceName",
    label: "Device Name",
    type: "string",
    category: "Device Info",
  },
  {
    field: "IsSupervised",
    label: "Device Supervised",
    type: "boolean",
    category: "Device Info",
  },
  {
    field: "DeviceCapacity",
    label: "Total Storage (GB)",
    type: "number",
    category: "Device Info",
  },
  {
    field: "AvailableDeviceCapacity",
    label: "Available Storage (GB)",
    type: "number",
    category: "Device Info",
  },
  {
    field: "BatteryLevel",
    label: "Battery Level",
    type: "number",
    category: "Device Info",
  },
  {
    field: "IsActivationLockEnabled",
    label: "Activation Lock",
    type: "boolean",
    category: "Security",
  },
  {
    field: "IsDeviceLocatorServiceEnabled",
    label: "Find My Enabled",
    type: "boolean",
    category: "Security",
  },
  {
    field: "IsCloudBackupEnabled",
    label: "iCloud Backup",
    type: "boolean",
    category: "Security",
  },
  {
    field: "PasscodePresent",
    label: "Passcode Set",
    type: "boolean",
    category: "Security",
  },
  {
    field: "PasscodeCompliant",
    label: "Passcode Compliant",
    type: "boolean",
    category: "Security",
  },
  {
    field: "IsRoaming",
    label: "Device Roaming",
    type: "boolean",
    category: "Network",
  },
  {
    field: "PersonalHotspotEnabled",
    label: "Personal Hotspot",
    type: "boolean",
    category: "Network",
  },
  {
    field: "IsMDMLostModeEnabled",
    label: "Lost Mode Active",
    type: "boolean",
    category: "MDM",
  },
];

const OPERATORS_BY_TYPE: Record<string, { value: string; label: string }[]> = {
  string: [
    { value: "eq", label: "equals" },
    { value: "neq", label: "does not equal" },
    { value: "contains", label: "contains" },
    { value: "not_contains", label: "does not contain" },
  ],
  boolean: [{ value: "eq", label: "equals" }],
  number: [
    { value: "eq", label: "equals" },
    { value: "gt", label: "greater than" },
    { value: "gte", label: "greater than or equal" },
    { value: "lt", label: "less than" },
    { value: "lte", label: "less than or equal" },
  ],
  version: [
    { value: "version_gte", label: "at least" },
    { value: "version_lte", label: "at most" },
    { value: "eq", label: "exactly" },
  ],
};

const BOOLEAN_VALUE_OPTIONS = [
  { value: "true", label: "true" },
  { value: "false", label: "false" },
];

const fieldOptions = MDM_FIELD_CATALOG.map((f) => ({
  value: f.field,
  label: `${f.label} (${f.category})`,
}));

const getFieldType = (fieldName: string): string => {
  return MDM_FIELD_CATALOG.find((f) => f.field === fieldName)?.type || "string";
};

const getOperatorsForField = (fieldName: string) => {
  const type = getFieldType(fieldName);
  return OPERATORS_BY_TYPE[type] || OPERATORS_BY_TYPE.string;
};

const getOperatorLabel = (operator: string): string => {
  let label = operator;
  Object.values(OPERATORS_BY_TYPE).forEach((ops) => {
    const found = ops.find((o) => o.value === operator);
    if (found) {
      label = found.label;
    }
  });
  return label;
};

const getFieldLabel = (fieldName: string): string => {
  return (
    MDM_FIELD_CATALOG.find((f) => f.field === fieldName)?.label || fieldName
  );
};

const MDMCheckBuilder = ({
  checks,
  onChange,
  disabled,
}: IMDMCheckBuilderProps) => {
  const [showTemplates, setShowTemplates] = useState(false);

  const updateCheck = (index: number, updates: Partial<IMDMPolicyCheck>) => {
    const newChecks = [...checks];
    newChecks[index] = { ...newChecks[index], ...updates };

    // Reset operator and expected when field changes
    if (updates.field) {
      const type = getFieldType(updates.field);
      const operators = OPERATORS_BY_TYPE[type] || OPERATORS_BY_TYPE.string;
      newChecks[index].operator = operators[0]?.value || "eq";
      if (type === "boolean") {
        newChecks[index].expected = "true";
      } else {
        newChecks[index].expected = "";
      }
    }

    onChange(newChecks);
  };

  const addCheck = () => {
    onChange([...checks, { field: "", operator: "eq", expected: "" }]);
  };

  const removeCheck = (index: number) => {
    const newChecks = checks.filter((_, i) => i !== index);
    onChange(newChecks);
  };

  const applyTemplate = (template: IMDMPolicyTemplate) => {
    onChange(template.checks.map((c) => ({ ...c })));
    setShowTemplates(false);
  };

  const renderCheckRow = (check: IMDMPolicyCheck, index: number) => {
    const fieldType = check.field ? getFieldType(check.field) : "string";
    const operators = check.field
      ? getOperatorsForField(check.field)
      : OPERATORS_BY_TYPE.string;
    const isBooleanField = fieldType === "boolean";

    return (
      <div
        key={`check-${check.field || "empty"}-${index}`}
        className={`${baseClass}__check-row`}
      >
        <Dropdown
          name={`field-${index}`}
          value={check.field}
          options={fieldOptions}
          onChange={(value: string) => updateCheck(index, { field: value })}
          placeholder="Select field..."
          className={`${baseClass}__field-dropdown`}
          disabled={disabled}
          searchable
        />
        <Dropdown
          name={`operator-${index}`}
          value={check.operator}
          options={operators}
          onChange={(value: string) => updateCheck(index, { operator: value })}
          placeholder="Operator"
          className={`${baseClass}__operator-dropdown`}
          disabled={disabled || !check.field}
          searchable={false}
        />
        {isBooleanField ? (
          <Dropdown
            name={`expected-${index}`}
            value={check.expected}
            options={BOOLEAN_VALUE_OPTIONS}
            onChange={(value: string) =>
              updateCheck(index, { expected: value })
            }
            className={`${baseClass}__value-dropdown`}
            disabled={disabled}
            searchable={false}
          />
        ) : (
          <InputField
            name={`expected-${index}`}
            value={check.expected}
            onChange={(value: string) =>
              updateCheck(index, { expected: value })
            }
            placeholder="Expected value"
            inputClassName={`${baseClass}__value-input`}
            disabled={disabled || !check.field}
          />
        )}
        {!disabled && (
          <Button
            className={`${baseClass}__remove-btn`}
            variant="text-icon"
            onClick={() => removeCheck(index)}
          >
            <Icon name="close" color="ui-fleet-black-50" size="small" />
          </Button>
        )}
      </div>
    );
  };

  const renderPreview = () => {
    const validChecks = checks.filter((c) => c.field && c.expected);
    if (validChecks.length === 0) return null;

    return (
      <div className={`${baseClass}__preview`}>
        <span className={`${baseClass}__preview-label`}>Preview:</span>
        <span className={`${baseClass}__preview-text`}>
          {validChecks.map((c) => (
            <span key={`${c.field}-${c.operator}-${c.expected}`}>
              {i > 0 && <strong> AND </strong>}
              {getFieldLabel(c.field)} {getOperatorLabel(c.operator)}{" "}
              <em>{c.expected}</em>
            </span>
          ))}
        </span>
      </div>
    );
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header`}>
        <span className="form-field__label">MDM device checks</span>
        {!disabled && (
          <Button
            variant="text-link"
            onClick={() => setShowTemplates(true)}
            className={`${baseClass}__templates-btn`}
          >
            Templates
          </Button>
        )}
      </div>
      <div className={`${baseClass}__checks`}>
        {checks.map((check, index) => renderCheckRow(check, index))}
      </div>
      {!disabled && (
        <Button
          variant="text-link"
          onClick={addCheck}
          className={`${baseClass}__add-btn`}
        >
          <Icon name="plus" /> Add check
        </Button>
      )}
      {renderPreview()}
      {showTemplates && (
        <Modal
          title="MDM policy templates"
          onExit={() => setShowTemplates(false)}
          className={`${baseClass}__templates-modal`}
        >
          <div className={`${baseClass}__templates-list`}>
            {DEFAULT_MDM_POLICIES.map((template) => (
              <div key={template.key} className={`${baseClass}__template-item`}>
                <div className={`${baseClass}__template-info`}>
                  <strong>{template.name}</strong>
                  <p>{template.description}</p>
                </div>
                <Button
                  variant="default"
                  onClick={() => applyTemplate(template)}
                  className={`${baseClass}__template-use-btn`}
                >
                  Use
                </Button>
              </div>
            ))}
          </div>
        </Modal>
      )}
    </div>
  );
};

export default MDMCheckBuilder;

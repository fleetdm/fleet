import { Ace } from "ace-builds";
import React, { useState } from "react";

import Button from "components/buttons/Button";
import { validateQuery } from "components/forms/validators/validate_query";
import SQLEditor from "components/SQLEditor";
import { LabelPlatform } from "interfaces/label";

import LabelForm from "../LabelForm";
import { ILabelFormData } from "../LabelForm/LabelForm";
import PlatformField from "../PlatformField";

const baseClass = "dynamic-label-form";

export interface IDynamicLabelFormData {
  name: string;
  description: string;
  query: string;
  platform: LabelPlatform;
}

interface IDynamicLabelFormProps {
  defaultName?: string;
  defaultDescription?: string;
  defaultQuery?: string;
  defaultPlatform?: LabelPlatform;
  showOpenSidebarButton?: boolean;
  isEditing?: boolean;
  onOpenSidebar?: () => void;
  onOsqueryTableSelect?: (tableName: string) => void;
  teamName: string | null;
  onSave: (formData: IDynamicLabelFormData) => void;
  onCancel: () => void;
}

const DynamicLabelForm = ({
  defaultName = "",
  defaultDescription = "",
  defaultQuery = "",
  defaultPlatform = "",
  isEditing = false,
  showOpenSidebarButton = false,
  onOpenSidebar,
  onOsqueryTableSelect,
  teamName,
  onSave,
  onCancel,
}: IDynamicLabelFormProps) => {
  const [query, setQuery] = useState(defaultQuery);
  const [platform, setPlatform] = useState(defaultPlatform);
  const [queryError, setQueryError] = useState<string | null>(null);

  const onQueryChange = (newQuery: string) => {
    setQuery(newQuery);
    // Clear a stale syntax error the moment the query parses again. Never
    // set a new error here: new errors wait for blur / submit.
    if (queryError && !validateQuery(newQuery).error) {
      setQueryError(null);
    }
  };

  // Validate SQL on blur so incomplete queries don't flash a syntax error
  // mid-type. Skipped when editing: the query is read-only, so the user
  // can't change it to resolve any error we'd surface.
  const onQueryBlur = () => {
    if (isEditing) return;
    const { error } = validateQuery(query);
    setQueryError(query === "" || error === "" ? null : error);
  };

  const onSaveForm = (
    labelFormData: ILabelFormData,
    labelFormDataValid: boolean
  ) => {
    // Query is read-only when editing, so skip SQL validation on save too.
    if (isEditing) {
      if (labelFormDataValid) {
        onSave({ ...labelFormData, query, platform });
      }
      return;
    }
    const { error } = validateQuery(query);
    if (error) {
      setQueryError(error);
    } else if (labelFormDataValid) {
      // values from LabelForm component must be valid too
      onSave({ ...labelFormData, query, platform });
    }
  };

  const renderLabelComponent = (): JSX.Element | null => {
    if (!showOpenSidebarButton) {
      return null;
    }

    return (
      <Button
        variant="subdued"
        onClick={onOpenSidebar}
        icon="info"
        iconPosition="right"
      >
        Schema
      </Button>
    );
  };

  const onLoad = (editor: Ace.Editor) => {
    editor.setOptions({
      enableMultiselect: false, // Disables command + click creating multiple cursors
    });

    // @ts-expect-error
    // the string "linkClick" is not officially in the lib but we need it
    editor.on("linkClick", (data) => {
      const { type, value } = data.token;

      if (type === "osquery-token" && onOsqueryTableSelect) {
        return onOsqueryTableSelect(value);
      }

      return false;
    });
  };

  const onChangePlatform = (value: LabelPlatform) => {
    setPlatform(value);
  };

  return (
    <div className={baseClass}>
      <LabelForm
        defaultName={defaultName}
        defaultDescription={defaultDescription}
        teamName={teamName}
        onSave={onSaveForm}
        onCancel={onCancel}
        immutableFields={
          teamName
            ? ["fleets", "queries", "platforms"]
            : ["queries", "platforms"]
        }
        additionalFields={
          <>
            <SQLEditor
              error={isEditing ? null : queryError}
              name="query"
              onChange={onQueryChange}
              onBlur={onQueryBlur}
              value={query}
              label="Query"
              labelActionComponent={renderLabelComponent()}
              readOnly={isEditing}
              onLoad={onLoad}
              wrapperClassName={`${baseClass}__text-editor-wrapper form-field`}
              wrapEnabled
              enableCopy={isEditing}
            />
            <PlatformField
              platform={platform}
              isEditing={isEditing}
              onChange={onChangePlatform}
            />
          </>
        }
      />
    </div>
  );
};

export default DynamicLabelForm;

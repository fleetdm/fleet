import React, { useState } from "react";
import { useDebouncedCallback } from "use-debounce";
import { IAceEditor } from "react-ace/lib/types";

// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";
import FleetAce from "components/FleetAce";
import Button from "components/buttons/Button";
import Icon from "components/Icon";

import LabelForm from "../LabelForm";
import { ILabelFormData } from "../LabelForm/LabelForm";
import PlatformField from "../PlatformField";

const baseClass = "dynamic-label-form";

const IMMUTABLE_QUERY_HELP_TEXT =
  "Label queries are immutable. To change the query, delete this label and create a new one.";

export interface IDynamicLabelFormData {
  name: string;
  description: string;
  query: string;
  platform: string;
}

interface IDynamicLabelFormProps {
  defaultName?: string;
  defaultDescription?: string;
  defaultQuery?: string;
  defaultPlatform?: string;
  showOpenSidebarButton?: boolean;
  isEditing?: boolean;
  onOpenSidebar?: () => void;
  onOsqueryTableSelect?: (tableName: string) => void;
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
  onSave,
  onCancel,
}: IDynamicLabelFormProps) => {
  const [query, setQuery] = useState(defaultQuery);
  const [platform, setPlatform] = useState(defaultPlatform);
  const [queryError, setQueryError] = useState<string | null>(null);

  const debounceValidateSQL = useDebouncedCallback((queryString: string) => {
    const { error } = validateQuery(queryString);
    if (query === "" || error === "") {
      setQueryError(null);
    } else {
      setQueryError(error);
    }
  }, 500);

  const onQueryChange = (newQuery: string) => {
    setQuery(newQuery);
    debounceValidateSQL(newQuery);
  };

  const onSaveForm = (
    labelFormData: ILabelFormData,
    labelFormDataValid: boolean
  ) => {
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
      <Button variant="text-icon" onClick={onOpenSidebar}>
        <Icon name="info" size="small" />
        <span>Show schema</span>
      </Button>
    );
  };

  const onLoad = (editor: IAceEditor) => {
    editor.setOptions({
      enableLinking: true,
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

  const onChangePlatform = (value: string) => {
    setPlatform(value);
  };

  return (
    <div className={baseClass}>
      <LabelForm
        defaultName={defaultName}
        defaultDescription={defaultDescription}
        onSave={onSaveForm}
        onCancel={onCancel}
        additionalFields={
          <>
            <FleetAce
              error={queryError}
              name="query"
              onChange={onQueryChange}
              value={query}
              label="Query"
              labelActionComponent={renderLabelComponent()}
              readOnly={isEditing}
              onLoad={onLoad}
              wrapperClassName={`${baseClass}__text-editor-wrapper form-field`}
              helpText={isEditing ? IMMUTABLE_QUERY_HELP_TEXT : ""}
              wrapEnabled
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

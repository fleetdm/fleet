import React, { useContext, useEffect, useState } from "react";

import { useQuery } from "react-query";
import { useDebouncedCallback } from "use-debounce";
import { IAceEditor } from "react-ace/lib/types";
import { Row } from "react-table";

import targetsAPI, { ITargetsSearchResponse } from "services/entities/targets";

// TODO - move this table config near here once expanded this logic to encompass editing and
// therefore not longer needed anywhere else
import { generateTableHeaders } from "pages/labels/components/ManualLabelForm/LabelHostTargetTableConfig";

// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

import { QueryContext } from "context/query";
import useToggleSidePanel from "hooks/useToggleSidePanel";

import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Button from "components/buttons/Button";

import { RouteComponentProps } from "react-router";
import {
  LabelHostVitalsCriterion,
  LabelMembershipType,
} from "interfaces/label";
import { IHost } from "interfaces/host";
import { IFormField } from "interfaces/form_field";
import SQLEditor from "components/SQLEditor";
import Icon from "components/Icon";
import TargetsInput from "components/TargetsInput";
import Radio from "components/forms/fields/Radio";

import PlatformField from "../components/PlatformField";

const availableCriteria: {
  label: string;
  value: LabelHostVitalsCriterion;
}[] = [
  { label: "Identity provider (IdP) group", value: "end_user_idp_group" },
  { label: "IdP department", value: "end_user_idp_department" },
];

const baseClass = "new-label-page";

export const LABEL_TARGET_HOSTS_INPUT_LABEL = "Select hosts";
const LABEL_TARGET_HOSTS_INPUT_PLACEHOLDER =
  "Search name, hostname, or serial number";
const DEBOUNCE_DELAY = 500;

const validate = (newData: INewLabelFormData) => {
  const errors: INewLabelFormErrors = {};
  // TODO - validation logic
  return errors;
};

interface ITargetsQueryKey {
  scope: string;
  query?: string | null;
  excludedHostIds?: number[];
}

export interface INewLabelFormData {
  name: string;
  description: string; // optional
  type: LabelMembershipType;
  // dynamic
  labelQuery: string;
  platform: string;

  // host vitals
  vital: LabelHostVitalsCriterion; // TODO - make use of recursive `LabelHostVitalsCriteria` type in future iterations to support logical combinations of different criteria
  vitalValue: string;

  // manual
  targetedHosts: IHost[];
}

interface INewLabelFormErrors {
  name?: string | null;
  query?: string | null;
  value?: string | null;
}

const NewLabelPage = ({
  router,
  location,
}: RouteComponentProps<never, never>) => {
  // page-level state
  const { selectedOsqueryTable, setSelectedOsqueryTable } = useContext(
    QueryContext
  );
  const { isSidePanelOpen, setSidePanelOpen } = useToggleSidePanel(true);
  const [showOpenSidebarButton, setShowOpenSidebarButton] = useState(false);

  // page-level handlers
  const onCloseSidebar = () => {
    setSidePanelOpen(false);
    setShowOpenSidebarButton(true);
  };

  const onOpenSidebar = () => {
    setSidePanelOpen(true);
    setShowOpenSidebarButton(false);
  };

  const onOsqueryTableSelect = (tableName: string) => {
    setSelectedOsqueryTable(tableName);
  };

  // form state
  const [isUpdating, setIsUpdating] = useState(false);
  const [formData, setFormData] = useState<INewLabelFormData>({
    name: "",
    description: "",
    type: "dynamic", // default type
    // dynamic-specific
    labelQuery: "",
    platform: "",
    // host-vitals-specific
    vital: "end_user_idp_group",
    vitalValue: "",
    // manual-specific
    targetedHosts: [],
  });
  const {
    name,
    description,
    type,
    labelQuery,
    platform,
    vital,
    vitalValue,
    targetedHosts,
  } = formData;

  const [formErrors, setFormErrors] = useState<INewLabelFormErrors>({});

  const [targetsSearchQuery, setTargetsSearchQuery] = useState("");
  const [
    debouncedTargetsSearchQuery,
    setDebouncedTargetsSearchQuery,
  ] = useState("");
  const [isDebouncingTargetsSearch, setIsDebouncingTargetsSearch] = useState(
    false
  );

  // "manual" label target search logic
  const debounceSearch = useDebouncedCallback(
    (search: string) => {
      setDebouncedTargetsSearchQuery(search);
      setIsDebouncingTargetsSearch(false);
    },
    DEBOUNCE_DELAY,
    { trailing: true }
  );

  useEffect(() => {
    setIsDebouncingTargetsSearch(true);
    debounceSearch(targetsSearchQuery);
  }, [debounceSearch, targetsSearchQuery]);

  const {
    data: targetsSearchResults,
    isLoading: isLoadingTargetsSearchResults,
    isError: isErrorTargetsSearchResults,
  } = useQuery<ITargetsSearchResponse, Error, IHost[], ITargetsQueryKey[]>(
    [
      {
        scope: "labels-targets-search",
        query: debouncedTargetsSearchQuery,
        excludedHostIds: targetedHosts.map((host) => host.id),
      },
    ],
    ({ queryKey }) => {
      const { query, excludedHostIds } = queryKey[0];
      return targetsAPI.search({
        query: query ?? "",
        excluded_host_ids: excludedHostIds ?? null,
      });
    },
    {
      select: (data) => data.hosts,
      enabled: type === "manual" && !!targetsSearchQuery,
    }
  );

  // form handlers

  const onInputChange = ({ name: fieldName, value }: IFormField) => {
    const newFormData = { ...formData, [fieldName]: value };
    setFormData(newFormData);
    const newErrs = validate(newFormData);
    // only set errors that are updates of existing errors
    // new errors are only set onBlur or submit
    const errsToSet: Record<string, string> = {};
    Object.keys(formErrors).forEach((k) => {
      // @ts-ignore
      if (newErrs[k]) {
        // @ts-ignore
        errsToSet[k] = newErrs[k];
      }
    });
    setFormErrors(errsToSet);
  };

  const onTypeChange = (value: string): void => {
    setFormData({
      ...formData,
      type: value as LabelMembershipType, // reconcile type differences between form data and radio component handler
    });
  };

  const onInputBlur = () => {
    setFormErrors(validate(formData));
  };

  const onSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const errs = validate(formData);
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }
    setIsUpdating(true);
    // TODO - create label
    setIsUpdating(false);
  };

  const debounceValidateSQL = useDebouncedCallback((queryString: string) => {
    const { error } = validateQuery(queryString);
    return error || null;
  }, 500);

  const onQueryChange = (newQuery: string) => {
    setFormData((prevData) => ({
      ...prevData,
      labelQuery: newQuery,
    }));
    debounceValidateSQL(newQuery);
  };

  // form rendering helpers
  const onLoadSQLEditor = (editor: IAceEditor) => {
    editor.setOptions({
      enableLinking: true,
      enableMultiselect: false, // Disables command + click creating multiple cursors
    });

    // @ts-expect-error
    // the string "linkClick" is not officially in the lib but we need it
    editor.on("linkClick", (data) => {
      const { type: type_, value } = data.token;

      if (type_ === "osquery-token" && onOsqueryTableSelect) {
        return onOsqueryTableSelect(value);
      }

      return false;
    });
  };

  const onChangeSearchQuery = (value: string) => {
    setTargetsSearchQuery(value);
  };
  const onHostSelect = (row: Row<IHost>) => {
    setFormData((prevData) => ({
      ...prevData,
      targetedHosts: targetedHosts.concat(row.original),
    }));
    setTargetsSearchQuery("");
  };

  const onHostRemove = (row: Row<IHost>) => {
    setFormData((prevData) => ({
      ...prevData,
      targetedHosts: targetedHosts.filter((h) => h.id !== row.original.id),
    }));
  };
  const resultsTableConfig = generateTableHeaders();
  const selectedHostsTableConfig = generateTableHeaders(onHostRemove);

  const renderVariableFields = () => {
    switch (type) {
      case "dynamic":
        return (
          <>
            <SQLEditor
              error={formErrors.query}
              name="query"
              onChange={onQueryChange}
              value={labelQuery}
              label="Query"
              labelActionComponent={
                showOpenSidebarButton ? (
                  <Button variant="text-icon" onClick={onOpenSidebar}>
                    Schema
                    <Icon name="info" size="small" />
                  </Button>
                ) : null
              }
              // readOnly={isEditing} TODO when extending to handle edits
              onLoad={onLoadSQLEditor}
              wrapperClassName={`${baseClass}__text-editor-wrapper form-field`}
              // helpText={isEditing ? IMMUTABLE_QUERY_HELP_TEXT : ""} TODO when extending to handle edits
              wrapEnabled
            />
            <PlatformField
              platform={platform}
              // isEditing={isEditing} TODO when extending to handle edits

              // onChange={onInputChange} TODO - once this form covers edits, can use the commmon
              // `onInputChange` along with updating PlatformField's Dropdown to `parseTarget`
              onChange={(newPlatform) => {
                setFormData((prevData) => ({
                  ...prevData,
                  platform: newPlatform,
                }));
              }}
            />
          </>
        );

      case "host-vitals":
        return (
          <div className={`${baseClass}__host-vitals-fields`}>
            <Dropdown
              label="Label criteria"
              name="vital"
              onChange={onInputChange}
              parseTarget
              value={vital}
              options={availableCriteria}
              classname={`${baseClass}__criteria-dropdown`}
              wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--criteria`}
              helptText="Currently, label criteria can be IdP group or department."
            />
            is equal to
            <InputField
              error={formErrors.value}
              name="value"
              onChange={onInputChange}
              value={vitalValue}
              inputClassName={`${baseClass}__vital-value`}
              placeholder={
                vital === "end_user_idp_group" ? "IT admins" : "Engineering"
              }
              onBlur={onInputBlur}
            />
          </div>
        );

      case "manual":
        return (
          <TargetsInput
            label={LABEL_TARGET_HOSTS_INPUT_LABEL}
            placeholder={LABEL_TARGET_HOSTS_INPUT_PLACEHOLDER}
            searchText={targetsSearchQuery}
            searchResultsTableConfig={resultsTableConfig}
            selectedHostsTableConifg={selectedHostsTableConfig}
            isTargetsLoading={
              isLoadingTargetsSearchResults || isDebouncingTargetsSearch
            }
            hasFetchError={isErrorTargetsSearchResults}
            searchResults={targetsSearchResults ?? []}
            targetedHosts={targetedHosts}
            setSearchText={onChangeSearchQuery}
            handleRowSelect={onHostSelect}
          />
        );
      default:
        return null;
    }
  };

  const renderLabelForm = () => (
    <form className={`${baseClass}__label-form`} onSubmit={onSubmit}>
      <InputField
        error={formErrors.name}
        name="name"
        onChange={onInputChange}
        value={name}
        inputClassName={`${baseClass}__label-name`}
        label="Name"
        placeholder="Label name"
      />
      <InputField
        name="description"
        onChange={onInputChange}
        value={description}
        inputClassName={`${baseClass}__label-description`}
        label="Description"
        type="textarea"
        placeholder="Label description (optional)"
      />
      <div className="form-field type-field">
        <div className="form-field__label">Type</div>
        <Radio
          className={`${baseClass}__radio-input`}
          label="Dynamic"
          id="dynamic"
          checked={type === "dynamic"}
          value="dynamic"
          name="label-type"
          onChange={onTypeChange}
        />
        <Radio
          className={`${baseClass}__radio-input`}
          label="Host vitals"
          id="host-vitals"
          checked={type === "host-vitals"}
          value="host-vitals"
          name="label-type"
          onChange={onTypeChange}
        />
        <Radio
          className={`${baseClass}__radio-input`}
          label="Manual"
          id="manual"
          checked={type === "manual"}
          value="manual"
          name="label-type"
          onChange={onTypeChange}
        />
      </div>
      {renderVariableFields()}
      <div className="button-wrap">
        <Button
          onClick={() => {
            router.goBack();
          }}
          variant="inverse"
          disabled={isUpdating}
        >
          Cancel
        </Button>
        <Button type="submit" isLoading={isUpdating} disabled={isUpdating}>
          Save
        </Button>
      </div>
    </form>
  );

  return (
    <>
      <MainContent className={baseClass}>
        <h1>New label</h1>
        <p className={`${baseClass}__page-description`}>
          Create a new label for targeting and filtering hosts.
        </p>
        {renderLabelForm()}
      </MainContent>
      {type === "dynamic" && isSidePanelOpen && (
        <SidePanelContent>
          <QuerySidePanel
            key="query-side-panel"
            onOsqueryTableSelect={onOsqueryTableSelect}
            selectedOsqueryTable={selectedOsqueryTable}
            onClose={onCloseSidebar}
          />
        </SidePanelContent>
      )}
    </>
  );
};

export default NewLabelPage;

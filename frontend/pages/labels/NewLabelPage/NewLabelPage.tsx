import React, { useContext, useEffect, useState } from "react";

import { useQuery } from "react-query";
import { useDebouncedCallback } from "use-debounce";
import { IAceEditor } from "react-ace/lib/types";
import { Row } from "react-table";

import PATHS from "router/paths";

import targetsAPI, { ITargetsSearchResponse } from "services/entities/targets";
import idpAPI from "services/entities/idp";
import labelsAPI from "services/entities/labels";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
// TODO - move this table config near here once expanded this logic to encompass editing and
// therefore not longer needed anywhere else
import { generateTableHeaders } from "pages/labels/components/ManualLabelForm/LabelHostTargetTableConfig";

// @ts-ignore
import validateQuery from "components/forms/validators/validate_query";

import { QueryContext } from "context/query";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";

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
  labelQuery?: string | null;
  criteria?: string | null;
}

const validate = (newData: INewLabelFormData) => {
  const errors: INewLabelFormErrors = {};
  const { name, type, labelQuery, vitalValue } = newData;
  if (!name) {
    errors.name = "Label name must be present";
  }
  if (type === "dynamic") {
    if (!labelQuery) {
      errors.labelQuery = "Query text must be present";
    }
  } else if (type === "host_vitals") {
    if (!vitalValue) {
      errors.criteria = "Label criteria must be completed";
    }
  }
  return errors;
};

const DEFAULT_DYNAMIC_QUERY = "SELECT 1 FROM os_version WHERE major >= 13;";

const NewLabelPage = ({
  router,
  location,
}: RouteComponentProps<never, never>) => {
  // page-level state
  const { selectedOsqueryTable, setSelectedOsqueryTable } = useContext(
    QueryContext
  );
  const { isPremiumTier } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);

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
    labelQuery: DEFAULT_DYNAMIC_QUERY,
    platform: "",
    // host_vitals-specific
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

  const { data: scimIdPDetails } = useQuery(
    ["scim_details"],
    () => idpAPI.getSCIMDetails(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      enabled: isPremiumTier,
    }
  );
  const idpConfigured = !!scimIdPDetails?.last_request?.requested_at;

  let hostVitalsTooltipContent: React.ReactNode;
  if (!isPremiumTier) {
    hostVitalsTooltipContent = (
      <>
        Currently, host vitals labels are based on
        <br />
        identity provider (IdP) groups or departments.
        <br />
        IdP integration available in Fleet Premium.
      </>
    );
  } else if (!idpConfigured) {
    hostVitalsTooltipContent = (
      <>
        Currently, host vitals labels are based on
        <br />
        identity provider (IdP) groups or departments.
        <br />
        Configure IdP in{" "}
        <a href="/settings/integrations/identity-provider">
          integration settings
        </a>
        .
      </>
    );
  }

  useEffect(() => {
    if (location.pathname.includes("dynamic")) {
      router.replace(PATHS.NEW_LABEL);
    }
    if (location.pathname.includes("manual")) {
      setFormData((prevData) => ({
        ...prevData,
        type: "manual",
      }));

      router.replace(PATHS.NEW_LABEL);
    }
  }, [location.pathname, router]);

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
    const newFormData = {
      ...formData,
      type: value as LabelMembershipType, // reconcile type differences between form data and radio component handler
    };
    setFormData(newFormData);

    const newErrs = validate(newFormData);
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

  const onInputBlur = () => {
    setFormErrors(validate(formData));
  };

  const onSubmit = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const errs = validate(formData);
    if (Object.keys(errs).length > 0) {
      setFormErrors(errs);
      return;
    }
    setIsUpdating(true);
    try {
      const res = await labelsAPI.create(formData);
      router.push(PATHS.MANAGE_HOSTS_LABEL(res.label.id));
      renderFlash("success", "Label added successfully.");
    } catch {
      renderFlash("error", "Couldn't add label. Please try again.");
    }
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
              error={formErrors.labelQuery}
              name="query"
              onChange={onQueryChange}
              onBlur={onInputBlur}
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

      case "host_vitals":
        return (
          <div className={`${baseClass}__host_vitals-fields`}>
            <Dropdown
              label="Label criteria"
              name="vital"
              onChange={onInputChange}
              parseTarget
              value={vital}
              error={formErrors.criteria}
              options={availableCriteria}
              classname={`${baseClass}__criteria-dropdown`}
              wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--criteria`}
              helpText="Currently, label criteria can be IdP group or department."
            />
            <p>is equal to</p>
            <InputField
              error={formErrors.criteria}
              name="vitalValue"
              onChange={onInputChange}
              onBlur={onInputBlur}
              value={vitalValue}
              inputClassName={`${baseClass}__vital-value`}
              placeholder={
                vital === "end_user_idp_group" ? "IT admins" : "Engineering"
              }
              parseTarget
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
        onBlur={onInputBlur}
        value={name}
        inputClassName={`${baseClass}__label-name`}
        label="Name"
        placeholder="Label name"
        parseTarget
      />
      <InputField
        name="description"
        onChange={onInputChange}
        onBlur={onInputBlur}
        value={description}
        inputClassName={`${baseClass}__label-description`}
        label="Description"
        type="textarea"
        placeholder="Label description (optional)"
        parseTarget
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
          id="host_vitals"
          checked={type === "host_vitals"}
          value="host_vitals"
          name="label-type"
          onChange={onTypeChange}
          tooltip={hostVitalsTooltipContent}
          disabled={!!hostVitalsTooltipContent}
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
        <Button
          type="submit"
          isLoading={isUpdating}
          disabled={isUpdating || !!Object.entries(formErrors).length}
        >
          Save
        </Button>
      </div>
    </form>
  );

  return (
    <>
      <MainContent className={baseClass}>
        <div className={`${baseClass}__header`}>
          <h1>New label</h1>
          <p className={`${baseClass}__page-description`}>
            Create a new label for targeting and filtering hosts.
          </p>
        </div>
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

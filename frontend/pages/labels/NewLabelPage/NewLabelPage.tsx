import React, { useContext, useEffect, useState } from "react";

import { useQuery } from "react-query";
import { useDebouncedCallback } from "use-debounce";
import { Ace } from "ace-builds";
import { Row } from "react-table";

import PATHS from "router/paths";

import targetsAPI, { ITargetsSearchResponse } from "services/entities/targets";
import idpAPI from "services/entities/idp";
import labelsAPI from "services/entities/labels";
import customHostVitalsAPI, {
  IListCustomHostVitalsApiParams,
} from "services/entities/custom_host_vitals";

import {
  DEFAULT_USE_QUERY_OPTIONS,
  MAX_ENTITY_NAME_LENGTH,
} from "utilities/constants";
// TODO - move this table config near here once expanded this logic to encompass editing and
// therefore not longer needed anywhere else
import { generateTableHeaders } from "pages/labels/components/ManualLabelForm/LabelHostTargetTableConfig";

import { validateQuery } from "components/forms/validators/validate_query";

import { QueryContext } from "context/query";
import { AppContext } from "context/app";
import { notify } from "components/ToastNotification";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import useToggleSidePanel from "hooks/useToggleSidePanel";

import { RouteComponentProps } from "react-router";
import {
  CUSTOM_HOST_VITAL_CRITERION,
  LabelHostVitalsCriterion,
  LabelMembershipType,
} from "interfaces/label";
import { IHost } from "interfaces/host";
import { IInputFieldParseTarget } from "interfaces/form_field";
import { getErrorReason } from "interfaces/errors";

import SidePanelPage from "components/SidePanelPage";
import MainContent from "components/MainContent";
import SidePanelContent from "components/SidePanelContent";
import QuerySidePanel from "components/side_panels/QuerySidePanel";
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Button from "components/buttons/Button";
import SQLEditor from "components/SQLEditor";
import Icon from "components/Icon";
import TargetsInput from "components/TargetsInput";
import Radio from "components/forms/fields/Radio";
import PlatformField from "../components/PlatformField";
import {
  validateNewLabelFormData,
  INewLabelFormValidation,
  buildCriterionOptionValue,
  parseCriterionOptionValue,
  getVitalValuePlaceholder,
  getCriterionHelpText,
} from "./helpers";

interface ICriterionOption {
  label: string;
  // Dropdown value: an IdP criterion's stable enum value, or a synthetic
  // `custom_host_vital:<id>` value for a custom host vital (see
  // buildCriterionOptionValue / parseCriterionOptionValue).
  value: string;
}

const IDP_CRITERIA: ICriterionOption[] = [
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
  // Set only when `vital === CUSTOM_HOST_VITAL_CRITERION`; identifies the
  // selected custom host vital definition.
  customHostVitalId?: number;

  // manual
  targetedHosts: IHost[];
}

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
  const [formErrors, setFormErrors] = useState<INewLabelFormValidation>({
    isValid: true,
  });

  const {
    name,
    description,
    type,
    labelQuery,
    platform,
    vital,
    vitalValue,
    customHostVitalId,
    targetedHosts,
  } = formData;

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

  // Custom host vitals are a Fleet Free feature, so this query runs on all
  // tiers. We fetch the full list (no search/pagination) to both gate the
  // "Host vitals" label type and populate the criteria selector.
  const customHostVitalsParams: IListCustomHostVitalsApiParams = {};
  const { data: customHostVitalsData } = useQuery(
    ["custom_host_vitals", customHostVitalsParams],
    () => customHostVitalsAPI.getCustomHostVitals(customHostVitalsParams),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );
  const customHostVitals = customHostVitalsData?.custom_host_vitals ?? [];
  const hasCustomHostVitals = customHostVitals.length > 0;

  // Host vitals labels can be based on IdP groups/departments (Premium, once an
  // IdP is configured) OR custom host vitals (any tier). The type is disabled
  // only when neither source is available.
  let hostVitalsTooltipContent: React.ReactNode;
  if (!idpConfigured && !hasCustomHostVitals) {
    // IdP criteria are Premium-only, so a Free-tier admin can't "configure your
    // IdP" — point them at the custom host vital path instead.
    hostVitalsTooltipContent = isPremiumTier ? (
      <>
        To use host vitals labels, configure your IdP in integration settings or
        add a custom host vital.
      </>
    ) : (
      <>
        To use host vitals labels, add a custom host vital. Identity provider
        (IdP) group and department criteria are available in Fleet Premium.
      </>
    );
  }

  // Each custom host vital becomes its own selectable criterion. IdP criteria
  // only appear when an IdP is configured.
  const criterionOptions: ICriterionOption[] = [
    ...(idpConfigured ? IDP_CRITERIA : []),
    ...customHostVitals.map((customHostVital) => ({
      label: customHostVital.name,
      value: buildCriterionOptionValue(customHostVital.id),
    })),
  ];

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

  const onInputChange = ({
    name: fieldName,
    value,
  }: IInputFieldParseTarget) => {
    const newFormData = { ...formData, [fieldName]: value };
    setFormData(newFormData);

    const fullValidation = validateNewLabelFormData(newFormData);

    setFormErrors((prev) => {
      const next: INewLabelFormValidation = { ...prev, isValid: true };

      // start from previous errors
      if (prev.name) next.name = prev.name;
      if (prev.description) next.description = prev.description;
      if (prev.labelQuery) next.labelQuery = prev.labelQuery;
      if (prev.criteria) next.criteria = prev.criteria;

      // ONLY CLEAR existing error on this field if it is now valid.
      if (fieldName === "name") {
        if (prev.name && fullValidation.name?.isValid) {
          next.name = undefined;
        }
      } else if (fieldName === "description") {
        if (prev.description && fullValidation.description?.isValid) {
          next.description = undefined;
        }
      } else if (fieldName === "vitalValue") {
        if (prev.criteria && fullValidation.criteria?.isValid) {
          next.criteria = undefined;
        }
      }

      const fields = [
        next.name,
        next.description,
        next.labelQuery,
        next.criteria,
      ];
      next.isValid = fields.every((f) => !f || f.isValid);

      return next;
    });
  };

  // The criteria dropdown carries a synthetic value for custom host vitals, so
  // it can't reuse the generic `onInputChange` (which would set `vital` to the
  // encoded string). Decode it back into `vital` + `customHostVitalId`.
  const onCriterionChange = (optionValue: string): void => {
    const {
      vital: nextVital,
      customHostVitalId: nextId,
    } = parseCriterionOptionValue(optionValue);

    const newFormData: INewLabelFormData = {
      ...formData,
      vital: nextVital,
      customHostVitalId: nextId,
    };
    setFormData(newFormData);

    const fullValidation = validateNewLabelFormData(newFormData);
    setFormErrors((prev) => {
      const next: INewLabelFormValidation = { ...prev, isValid: true };

      if (prev.name) next.name = prev.name;
      if (prev.description) next.description = prev.description;
      if (prev.labelQuery) next.labelQuery = prev.labelQuery;
      if (prev.criteria && fullValidation.criteria?.isValid) {
        next.criteria = undefined;
      } else if (prev.criteria) {
        next.criteria = prev.criteria;
      }

      const fields = [
        next.name,
        next.description,
        next.labelQuery,
        next.criteria,
      ];
      next.isValid = fields.every((f) => !f || f.isValid);

      return next;
    });
  };

  const onTypeChange = (value: string): void => {
    const nextType = value as LabelMembershipType;
    const newFormData: INewLabelFormData = {
      ...formData,
      type: nextType,
    };

    // When switching to "host vitals", ensure the selected criterion is one the
    // dropdown actually offers: the default `end_user_idp_group` is invalid when
    // no IdP is configured (custom-host-vital-only case), so fall back to the
    // first custom host vital.
    if (nextType === "host_vitals" && !idpConfigured && hasCustomHostVitals) {
      newFormData.vital = CUSTOM_HOST_VITAL_CRITERION;
      newFormData.customHostVitalId = customHostVitals[0].id;
    }

    setFormData(newFormData);

    const fullValidation = validateNewLabelFormData(newFormData);

    setFormErrors((prev) => {
      const next: INewLabelFormValidation = { ...prev, isValid: true };

      if (prev.name) next.name = fullValidation.name ?? prev.name;
      if (prev.description)
        next.description = fullValidation.description ?? prev.description;
      if (prev.labelQuery)
        next.labelQuery = fullValidation.labelQuery ?? prev.labelQuery;
      if (prev.criteria)
        next.criteria = fullValidation.criteria ?? prev.criteria;

      const fields = [
        next.name,
        next.description,
        next.labelQuery,
        next.criteria,
      ];
      next.isValid = fields.every((f) => !f || f.isValid);

      return next;
    });
  };

  const onInputBlur = () => {
    setFormErrors(validateNewLabelFormData(formData));
  };

  const onSubmit = async (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const fullValidation = validateNewLabelFormData(formData);
    setFormErrors(fullValidation);
    if (!fullValidation.isValid) {
      return;
    }

    setIsUpdating(true);
    try {
      await labelsAPI.create(formData);
      notify.success("Label added successfully.");
      router.push(PATHS.MANAGE_LABELS);
    } catch (error) {
      const status = (error as { status: number }).status;
      let errorMessage = "Couldn't add label. Please try again.";
      if (status === 409) {
        errorMessage =
          "Couldn't add label: A label with this name already exists.";
      } else if (status === 422) {
        const reason = getErrorReason(error);
        if (reason) {
          errorMessage = `Couldn't add label: ${reason}. Please try again.`;
        }
      }
      notify.error(errorMessage, { response: error });
    }
    setIsUpdating(false);
  };

  const debounceValidateSQL = useDebouncedCallback((queryString: string) => {
    const { error } = validateQuery(queryString);
    return error || null;
  }, 500);

  const onQueryChange = (newQuery: string) => {
    const newFormData = { ...formData, labelQuery: newQuery };
    setFormData(newFormData);

    const fullValidation = validateNewLabelFormData(newFormData);

    setFormErrors((prev) => {
      const next: INewLabelFormValidation = { ...prev, isValid: true };

      if (prev.name) next.name = prev.name;
      if (prev.description) next.description = prev.description;
      if (prev.labelQuery) next.labelQuery = prev.labelQuery;
      if (prev.criteria) next.criteria = prev.criteria;

      if (prev.labelQuery && fullValidation.labelQuery?.isValid) {
        next.labelQuery = undefined;
      }

      const fields = [
        next.name,
        next.description,
        next.labelQuery,
        next.criteria,
      ];
      next.isValid = fields.every((f) => !f || f.isValid);

      return next;
    });

    debounceValidateSQL(newQuery);
  };

  // form rendering helpers
  const onLoadSQLEditor = (editor: Ace.Editor) => {
    editor.setOptions({
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
              error={formErrors.labelQuery?.message}
              name="query"
              onChange={onQueryChange}
              onBlur={onInputBlur}
              value={labelQuery}
              label="Query"
              labelActionComponent={
                showOpenSidebarButton ? (
                  <Button variant="subdued" onClick={onOpenSidebar}>
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

      case "host_vitals": {
        // The selected criterion is identified by the dropdown's string value:
        // IdP criteria use their stable enum value; each custom host vital uses
        // a synthetic `custom_host_vital:<id>` value so multiple custom vitals
        // are distinguishable in a single dropdown.
        const selectedCriterionValue =
          vital === CUSTOM_HOST_VITAL_CRITERION && customHostVitalId != null
            ? buildCriterionOptionValue(customHostVitalId)
            : vital;

        return (
          <div className={`${baseClass}__host_vitals-fields`}>
            <label className="form-field__label" htmlFor="criterion-and-value">
              Label criteria
            </label>
            <span id="criterion-and-value">
              <Dropdown
                name="vital"
                onChange={onCriterionChange}
                value={selectedCriterionValue}
                error={formErrors.criteria?.message}
                options={criterionOptions}
                classname={`${baseClass}__criteria-dropdown`}
                wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--criteria`}
              />
              <p>is equal to</p>
              <InputField
                error={formErrors.criteria?.message}
                name="vitalValue"
                onChange={onInputChange}
                onBlur={onInputBlur}
                value={vitalValue}
                inputClassName={`${baseClass}__vital-value`}
                placeholder={getVitalValuePlaceholder(vital)}
                parseTarget
              />
            </span>
            <span className="form-field__help-text">
              {getCriterionHelpText(vital)}
            </span>
          </div>
        );
      }

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
        error={formErrors.name?.message}
        name="name"
        onChange={onInputChange}
        onBlur={onInputBlur}
        value={name}
        inputClassName={`${baseClass}__label-name`}
        label="Name"
        placeholder="Label name"
        parseTarget
        inputOptions={{ maxLength: MAX_ENTITY_NAME_LENGTH }}
      />
      <InputField
        error={formErrors.description?.message}
        name="description"
        onChange={onInputChange}
        onBlur={onInputBlur}
        value={description}
        inputClassName={`${baseClass}__label-description`}
        label="Description"
        type="textarea"
        placeholder="Label description (optional)"
        parseTarget
        inputOptions={{ maxLength: MAX_ENTITY_NAME_LENGTH }}
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
        <GitOpsModeTooltipWrapper
          entityType="labels"
          renderChildren={(disableChildren) => (
            <Button
              type="submit"
              isLoading={isUpdating}
              disabled={disableChildren || isUpdating || !formErrors.isValid}
            >
              Save
            </Button>
          )}
        />
        <Button
          onClick={() => {
            router.goBack();
          }}
          variant="secondary"
          disabled={isUpdating}
        >
          Cancel
        </Button>
      </div>
    </form>
  );

  return (
    <SidePanelPage>
      <>
        <MainContent className={baseClass}>
          <div className={`${baseClass}__header`}>
            <h1 className="page-header">New label</h1>
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
    </SidePanelPage>
  );
};

export default NewLabelPage;

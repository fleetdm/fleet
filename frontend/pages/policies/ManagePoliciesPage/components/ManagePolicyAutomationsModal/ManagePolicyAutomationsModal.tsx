import React, { useState, useEffect } from "react";
import { Link } from "react-router";
import { isEmpty, noop, omit } from "lodash";

import { IAutomationsConfig, IWebhookSettings } from "interfaces/config";
import { IIntegration, IIntegrations } from "interfaces/integration";
import { IPolicy } from "interfaces/policy";
import { ITeamAutomationsConfig } from "interfaces/team";
import PATHS from "router/paths";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Slider from "components/forms/fields/Slider";
// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Radio from "components/forms/fields/Radio";
import validUrl from "components/forms/validators/valid_url";

import PreviewPayloadModal from "../PreviewPayloadModal";
import PreviewTicketModal from "../PreviewTicketModal";

interface IManagePolicyAutomationsModalProps {
  automationsConfig: IAutomationsConfig | ITeamAutomationsConfig;
  availableIntegrations: IIntegrations;
  availablePolicies: IPolicy[];
  isUpdatingAutomations: boolean;
  showPreviewPayloadModal: boolean;
  onExit: () => void;
  handleSubmit: (formData: {
    webhook_settings: Pick<IWebhookSettings, "failing_policies_webhook">;
    integrations: IIntegrations;
  }) => void;
  togglePreviewPayloadModal: () => void;
}

interface ICheckedPolicy {
  name?: string;
  id: number;
  isChecked: boolean;
}

const findEnabledIntegration = ({ jira, zendesk }: IIntegrations) => {
  return (
    jira?.find((j) => j.enable_failing_policies) ||
    zendesk?.find((z) => z.enable_failing_policies)
  );
};

const getIntegrationType = (integration?: IIntegration) => {
  return (
    (!!integration?.group_id && "zendesk") ||
    (!!integration?.project_key && "jira") ||
    undefined
  );
};

const useCheckboxListStateManagement = (
  allPolicies: IPolicy[],
  automatedPolicies: number[] | undefined
) => {
  const [policyItems, setPolicyItems] = useState<ICheckedPolicy[]>(() => {
    return allPolicies.map(({ name, id }) => ({
      name,
      id,
      isChecked: !!automatedPolicies?.includes(id),
    }));
  });

  const updatePolicyItems = (policyId: number) => {
    setPolicyItems((prevItems) =>
      prevItems.map((policy) =>
        policy.id !== policyId
          ? policy
          : { ...policy, isChecked: !policy.isChecked }
      )
    );
  };

  return { policyItems, updatePolicyItems };
};

const baseClass = "manage-policy-automations-modal";

const ManagePolicyAutomationsModal = ({
  automationsConfig,
  availableIntegrations,
  availablePolicies,
  isUpdatingAutomations,
  showPreviewPayloadModal,
  onExit,
  handleSubmit,
  togglePreviewPayloadModal: togglePreviewModal,
}: IManagePolicyAutomationsModalProps): JSX.Element => {
  const {
    webhook_settings: { failing_policies_webhook: webhook },
  } = automationsConfig;

  const { jira, zendesk } = availableIntegrations || {};
  const allIntegrations: IIntegration[] = [];
  jira && allIntegrations.push(...jira);
  zendesk && allIntegrations.push(...zendesk);

  const dropdownOptions = allIntegrations.map(
    ({ group_id, project_key, url }) => ({
      value: group_id || project_key,
      label: `${url} - ${group_id || project_key}`,
    })
  );

  const serverEnabledIntegration = findEnabledIntegration(
    automationsConfig.integrations
  );

  const [isPolicyAutomationsEnabled, setIsPolicyAutomationsEnabled] = useState(
    !!webhook.enable_failing_policies_webhook || !!serverEnabledIntegration
  );

  const [isWebhookEnabled, setIsWebhookEnabled] = useState(
    !isPolicyAutomationsEnabled || webhook.enable_failing_policies_webhook
  );

  const [destinationUrl, setDestinationUrl] = useState(
    webhook.destination_url || ""
  );

  const [selectedIntegration, setSelectedIntegration] = useState<
    IIntegration | undefined
  >(serverEnabledIntegration);

  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  const { policyItems, updatePolicyItems } = useCheckboxListStateManagement(
    availablePolicies,
    webhook?.policy_ids || []
  );

  const onChangeUrl = (value: string) => {
    setDestinationUrl(value);
    setErrors((errs) => omit(errs, "url"));
  };

  const onChangeRadio = (val: string) => {
    switch (val) {
      case "webhook":
        setIsWebhookEnabled(true);
        setSelectedIntegration(undefined);
        break;
      case "ticket":
        setIsWebhookEnabled(false);
        break;
      default:
        noop();
    }
  };

  const onSelectIntegration = (selected: string | number) => {
    setSelectedIntegration(
      allIntegrations.find(
        ({ group_id, project_key }) =>
          group_id === selected || project_key === selected
      )
    );
  };

  const onSubmit = (evt: React.MouseEvent<HTMLFormElement> | KeyboardEvent) => {
    evt.preventDefault();

    const newPolicyIds: number[] = [];
    policyItems?.forEach((p) => p.isChecked && newPolicyIds.push(p.id));

    const newErrors = { ...errors };

    if (
      isPolicyAutomationsEnabled &&
      newPolicyIds.length &&
      !isWebhookEnabled &&
      !selectedIntegration
    ) {
      newErrors.integration = "Please enable at least one integration:";
    } else {
      delete newErrors.integration;
    }

    if (isWebhookEnabled) {
      if (!destinationUrl) {
        newErrors.url = "Please add a destination URL";
      } else if (!validUrl({ url: destinationUrl })) {
        newErrors.url = `${destinationUrl} is not a valid URL`;
      } else {
        delete newErrors.url;
      }
    }

    if (!isEmpty(newErrors)) {
      setErrors(newErrors);
      return;
    }

    const newJira =
      availableIntegrations.jira?.map((j) => ({
        ...j,
        enable_failing_policies:
          isPolicyAutomationsEnabled &&
          !isWebhookEnabled &&
          j.project_key === selectedIntegration?.project_key,
      })) || null;

    const newZendesk =
      availableIntegrations.zendesk?.map((z) => ({
        ...z,
        enable_failing_policies:
          isPolicyAutomationsEnabled &&
          !isWebhookEnabled &&
          z.group_id === selectedIntegration?.group_id,
      })) || null;

    // if (
    //   !isPolicyAutomationsEnabled ||
    //   (!isWebhookEnabled && !selectedIntegration)
    // ) {
    //   newPolicyIds = [];
    // }

    const updatedEnabledPoliciesAcrossPages = () => {
      if (webhook.policy_ids) {
        // Array of policy ids on the page
        const availablePoliciesIds = availablePolicies.map(
          (policy) => policy.id
        );

        // Array of policy ids enabled NOT on the page
        const enabledPoliciesOnOtherPages = webhook.policy_ids.filter(
          (policyId) => !availablePoliciesIds.includes(policyId)
        );

        // Concatenate with array of policies enabled on the page
        const allEnabledPolicies = enabledPoliciesOnOtherPages.concat(
          newPolicyIds
        );

        return allEnabledPolicies;
      }

      return [];
    };

    // NOTE: backend uses webhook_settings to store automated policy ids for both webhooks and integrations
    const newWebhook = {
      failing_policies_webhook: {
        destination_url: destinationUrl,
        policy_ids: updatedEnabledPoliciesAcrossPages(),
        enable_failing_policies_webhook:
          isPolicyAutomationsEnabled && isWebhookEnabled,
      },
    };

    handleSubmit({
      webhook_settings: newWebhook,
      integrations: {
        jira: newJira,
        zendesk: newZendesk,
      },
    });

    setErrors(newErrors);
  };

  useEffect(() => {
    const listener = (event: KeyboardEvent) => {
      if (event.code === "Enter" || event.code === "NumpadEnter") {
        event.preventDefault();
        onSubmit(event);
      }
    };
    document.addEventListener("keydown", listener);
    return () => {
      document.removeEventListener("keydown", listener);
    };
  });

  const renderWebhook = () => {
    return (
      <>
        <InputField
          inputWrapperClass={`${baseClass}__url-input`}
          name="webhook-url"
          label="Destination URL"
          type="text"
          value={destinationUrl}
          onChange={onChangeUrl}
          error={errors.url}
          helpText='For each policy, Fleet will send a JSON payload to this URL with a list of the hosts that updated their answer to "No."'
          placeholder="https://server.com/example"
          tooltip="Provide a URL to deliver a webhook request to."
        />
        <Button type="button" variant="text-link" onClick={togglePreviewModal}>
          Preview payload
        </Button>
      </>
    );
  };

  const renderIntegrations = () => {
    return jira?.length || zendesk?.length ? (
      <div className={`${baseClass}__integrations`}>
        <Dropdown
          options={dropdownOptions}
          onChange={onSelectIntegration}
          placeholder="Select integration"
          value={
            selectedIntegration?.group_id || selectedIntegration?.project_key
          }
          label="Integration"
          error={errors.integration}
          wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--frequency`}
          hint={
            "For each policy, Fleet will create a ticket with a list of the failing hosts."
          }
        />
        <Button type="button" variant="text-link" onClick={togglePreviewModal}>
          Preview ticket
        </Button>
      </div>
    ) : (
      <div className={`form-field ${baseClass}__no-integrations`}>
        <div className="form-field__label">You have no integrations.</div>
        <Link
          to={PATHS.ADMIN_INTEGRATIONS}
          className={`${baseClass}__add-integration-link`}
        >
          Add integration
        </Link>
      </div>
    );
  };

  const renderPreview = () =>
    !isWebhookEnabled ? (
      <PreviewTicketModal
        integrationType={getIntegrationType(selectedIntegration)}
        onCancel={togglePreviewModal}
      />
    ) : (
      <PreviewPayloadModal onCancel={togglePreviewModal} />
    );

  return showPreviewPayloadModal ? (
    renderPreview()
  ) : (
    <Modal
      onExit={onExit}
      title="Manage automations"
      className={baseClass}
      width="large"
    >
      <div className={`${baseClass} form`}>
        <Slider
          value={isPolicyAutomationsEnabled}
          onChange={() => {
            setIsPolicyAutomationsEnabled(!isPolicyAutomationsEnabled);
            setErrors({});
          }}
          inactiveText="Policy automations disabled"
          activeText="Policy automations enabled"
        />
        <div
          className={`form ${baseClass}__policy-automations__${
            isPolicyAutomationsEnabled ? "enabled" : "disabled"
          }`}
        >
          <div className="form-field">
            {availablePolicies?.length ? (
              <>
                <div className="form-field__label">
                  Choose which policies you would like to listen to:
                </div>
                {policyItems &&
                  policyItems.map((policyItem) => {
                    const { isChecked, name, id } = policyItem;
                    return (
                      <div key={id}>
                        <Checkbox
                          value={isChecked}
                          name={name}
                          onChange={() => {
                            updatePolicyItems(policyItem.id);
                            !isChecked &&
                              setErrors((errs) => omit(errs, "policyItems"));
                          }}
                        >
                          {name}
                        </Checkbox>
                      </div>
                    );
                  })}
              </>
            ) : (
              <>
                <b>You have no policies.</b>
                <p>Add a policy to turn on automations.</p>
              </>
            )}
          </div>
          <div className={`form-field ${baseClass}__workflow`}>
            <div className="form-field__label">Workflow</div>
            <Radio
              className={`${baseClass}__radio-input`}
              label="Ticket"
              id="ticket-radio-btn"
              checked={!isWebhookEnabled}
              value="ticket"
              name="ticket"
              onChange={onChangeRadio}
            />
            <Radio
              className={`${baseClass}__radio-input`}
              label="Webhook"
              id="webhook-radio-btn"
              checked={isWebhookEnabled}
              value="webhook"
              name="webhook"
              onChange={onChangeRadio}
            />
          </div>
          {isWebhookEnabled ? renderWebhook() : renderIntegrations()}
        </div>
        <div className="modal-cta-wrap">
          <Button
            type="submit"
            variant="brand"
            onClick={onSubmit}
            className="save-loading"
            isLoading={isUpdatingAutomations}
          >
            Save
          </Button>
          <Button onClick={onExit} variant="inverse">
            Cancel
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ManagePolicyAutomationsModal;

import React, { useContext, useState, useRef } from "react";

import PATHS from "router/paths";
import { AppContext } from "context/app";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import BackButton from "components/BackButton";
import MainContent from "components/MainContent";
import CustomLink from "components/CustomLink/CustomLink";
import PageDescription from "components/PageDescription";
import EmptyTable from "components/EmptyTable";
import Card from "components/Card";
import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import UploadList from "components/UploadList";

import AddEntraTenantModal from "../components/AddEntraTenantModal";
import DeleteEntraTenantModal from "../components/DeleteEntraTenantModal";

import EntraTenantsListHeader from "./EntraTenantsListHeader";
import EntraTenantsListItem from "./EntraTenantsListItem";

const generateMdmTermsOfUseUrl = (domain: string) => {
  return `${domain}/api/mdm/microsoft/tos`;
};

const generateMdmDiscoveryUrl = (domain: string) => {
  return `${domain}/api/mdm/microsoft/discovery`;
};

const baseClass = "windows-automatic-enrollment-page";

const WindowsAutomaticEnrollmentPage = () => {
  const { config, setConfig } = useContext(AppContext);

  const deletingTenantId = useRef<null | string>(null);

  const [showAddTenantModal, setShowAddTenantModal] = useState(false);
  const [showDeleteTenantModal, setShowDeleteTenantModal] = useState(false);

  const onDeleteTenant = (tenantId: string) => {
    deletingTenantId.current = tenantId;
    setShowDeleteTenantModal(true);
  };

  const renderEntraTenants = () => {
    const tenants = config?.mdm.windows_entra_tenant_ids;

    if (!tenants || tenants.length === 0) {
      return (
        <Card paddingSize="xxlarge">
          <EmptyTable
            className={`${baseClass}__empty-tenant-message`}
            header="No tenants added"
            info="Add your Entra tenant ID to be able to enroll Windows hosts."
            primaryButton={
              <GitOpsModeTooltipWrapper
                renderChildren={(disable) => (
                  <Button
                    onClick={() => setShowAddTenantModal(true)}
                    disabled={disable}
                  >
                    Add
                  </Button>
                )}
              />
            }
          />
        </Card>
      );
    }

    return (
      <UploadList
        className={`${baseClass}__tenant-list`}
        listItems={tenants}
        HeadingComponent={() => (
          <EntraTenantsListHeader
            onClickAddTenant={() => setShowAddTenantModal(true)}
          />
        )}
        ListItemComponent={({ listItem }) => (
          <EntraTenantsListItem
            tenantId={listItem}
            onClickDelete={() => onDeleteTenant(listItem)}
          />
        )}
      />
    );
  };

  return (
    <MainContent className={baseClass}>
      <>
        <div className={`${baseClass}__header-links`}>
          <BackButton
            text="Back to MDM"
            path={PATHS.ADMIN_INTEGRATIONS_MDM}
            className={`${baseClass}__back-to-automatic-enrollment`}
          />
        </div>
        <h1>Microsoft Entra</h1>
        <div className={`${baseClass}__content-container`}>
          <PageDescription
            content={
              <>
                To connect Fleet to Microsoft Entra, follow the instructions in
                the{" "}
                <CustomLink
                  newTab
                  text="guide"
                  url="https://fleetdm.com/learn-more-about/connect-microsoft-entra"
                />
              </>
            }
          />
          <section className={`${baseClass}__mdm-urls-container`}>
            <h2>MDM URLs</h2>
            <div>
              <p>
                You will need to copy and paste these values to create the
                application in Microsoft Entra.
              </p>
              <div className={`${baseClass}__url-inputs-wrapper`}>
                <InputField
                  inputWrapperClass={`${baseClass}__url-input`}
                  label="MDM terms of use URL"
                  name="mdmTermsOfUseUrl"
                  tooltip="The terms of use URL is used to display the terms of service to end users
                  before turning on MDM for their host. The terms of use text informs users about
                  policies that will be enforced on the host."
                  value={generateMdmTermsOfUseUrl(
                    config?.server_settings.server_url || ""
                  )}
                  enableCopy
                />
                <InputField
                  inputWrapperClass={`${baseClass}__url-input`}
                  label="MDM discovery URL"
                  name="mdmDiscoveryUrl"
                  tooltip="The enrollment URL is used to connect hosts with the MDM service."
                  value={generateMdmDiscoveryUrl(
                    config?.server_settings.server_url || ""
                  )}
                  enableCopy
                />
              </div>
            </div>
          </section>
          <section className={`${baseClass}__tenants-container`}>
            <h2>Entra tenants</h2>
            <div>{renderEntraTenants()}</div>
          </section>
        </div>
        {showAddTenantModal && (
          <AddEntraTenantModal onExit={() => setShowAddTenantModal(false)} />
        )}
        {showDeleteTenantModal && deletingTenantId.current && (
          <DeleteEntraTenantModal
            tenantId={deletingTenantId.current}
            onExit={() => {
              deletingTenantId.current = null;
              setShowDeleteTenantModal(false);
            }}
          />
        )}
      </>
    </MainContent>
  );
};

export default WindowsAutomaticEnrollmentPage;

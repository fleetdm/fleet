import React, { useContext } from "react";

import { AppContext } from "context/app";
import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { ISoftwarePackage } from "interfaces/software";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Editor from "components/Editor";
import { createPackageYaml, renderYamlHelperText } from "./helpers";

const baseClass = "view-yaml-modal";

interface IViewYamlModalProps {
  softwareTitle: string;
  softwarePackage: ISoftwarePackage;
  onExit: () => void;
}

const ViewYamlModal = ({
  softwareTitle,
  softwarePackage,
  onExit,
}: IViewYamlModalProps) => {
  const { config } = useContext(AppContext);
  const repositoryUrl = config?.gitops?.repository_url;
  const packageYaml = createPackageYaml({
    softwareTitle,
    packageName: softwarePackage.name,
    version: softwarePackage.version,
    url: softwarePackage.url,
    sha256: softwarePackage.hash_sha256,
    includePreInstallQuery: !!softwarePackage.pre_install_query,
    includeInstallScript: !!softwarePackage.install_script,
    includePostInstallScript: !!softwarePackage.post_install_script,
    includeUninstallScript: !!softwarePackage.uninstall_script,
  });

  return (
    <Modal className={baseClass} title="YAML" onExit={onExit}>
      <>
        <InfoBanner className={`${baseClass}__info-banner`}>
          <p>
            To complete your GitOps configuration, follow the instructions
            below. If the YAML is not added, the package will be deleted on the
            next GitOps run.&nbsp;
            <CustomLink
              url={`${LEARN_MORE_ABOUT_BASE_LINK}/yaml-software`}
              text="How to use YAML"
              newTab
              multiline
            />
          </p>
        </InfoBanner>
        {repositoryUrl && (
          <p>
            First, create the YAML file below and save it to your{" "}
            <CustomLink url={repositoryUrl} text="repository" newTab />.
          </p>
        )}
        <p>Make sure you reference the package YAML from your team YAML.</p>
        <div className={`${baseClass}__form-fields`}>
          <InputField
            enableCopy
            readOnly
            inputWrapperClass
            name="filename"
            label="Filename"
            value={softwarePackage.name}
          />
          <Editor
            label="Contents"
            helpText={renderYamlHelperText(softwarePackage)}
            value={packageYaml}
          />
        </div>
        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default ViewYamlModal;

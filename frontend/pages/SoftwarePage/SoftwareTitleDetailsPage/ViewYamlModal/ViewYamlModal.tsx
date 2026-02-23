import React, { useContext } from "react";

import { AppContext } from "context/app";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { ISoftwarePackage } from "interfaces/software";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Editor from "components/Editor";

import { hyphenateString } from "utilities/strings/stringUtils";
import { createPackageYaml } from "./helpers";

const baseClass = "view-yaml-modal";

interface IViewYamlModalProps {
  softwareTitleName: string;
  softwareTitleId: number;
  teamId: number;
  iconUrl?: string | null;
  displayName?: string;
  softwarePackage: ISoftwarePackage;
  onExit: () => void;
  isScriptPackage?: boolean;
  isIosOrIpadosApp?: boolean;
}

const ViewYamlModal = ({
  softwareTitleName,
  iconUrl,
  displayName,
  softwarePackage,
  onExit,
  isScriptPackage = false,
}: IViewYamlModalProps) => {
  const { config } = useContext(AppContext);
  const repositoryUrl = config?.gitops?.repository_url;
  const { name, version, url, hash_sha256: sha256 } = softwarePackage;

  const packageYaml = createPackageYaml({
    softwareTitle: softwareTitleName,
    packageName: name,
    version,
    url,
    sha256,
    iconUrl: iconUrl || null,
    displayName,
    isScriptPackage,
  });

  const hyphenatedSoftwareTitle = hyphenateString(softwareTitleName);

  return (
    <Modal className={baseClass} title="YAML" onExit={onExit}>
      <>
        {repositoryUrl && (
          <p>
            Manage in{" "}
            <CustomLink url={repositoryUrl} text="YAML" newTab />.
          </p>
        )}
        <div className={`${baseClass}__form-fields`}>
          <InputField
            enableCopy
            readOnly
            name="filename"
            label="Filename"
            value={`${hyphenatedSoftwareTitle}.package.yml`}
          />
          <Editor
            label="Contents"
            value={packageYaml}
            enableCopy
            helpText={
              <>
                If you added advanced options, learn how to{" "}
                <CustomLink
                  url={`${LEARN_MORE_ABOUT_BASE_LINK}/yaml-packages`}
                  text="add them to your YAML"
                  newTab
                />
                .
              </>
            }
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

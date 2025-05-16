import React from "react";

import { LEARN_MORE_ABOUT_BASE_LINK } from "utilities/constants";
import { ISoftwarePackage } from "interfaces/software";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import InfoBanner from "components/InfoBanner";
import CustomLink from "components/CustomLink";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Editor from "components/Editor";

const baseClass = "view-yaml-modal";

interface IViewYamlModalProps {
  softwarePackage: ISoftwarePackage;
  onExit: () => void;
}

const ViewYamlModal = ({ softwarePackage, onExit }: IViewYamlModalProps) => {
  const renderHelpText = (): JSX.Element | undefined => {
    // Build the list of available items as buttons
    const items: JSX.Element[] = [];

    if (softwarePackage.pre_install_query) {
      items.push(
        <Button key="pre" variant="text-link">
          pre-install query
        </Button>
      );
    }
    if (softwarePackage.install_script) {
      items.push(
        <Button key="install" variant="text-link">
          install script
        </Button>
      );
    }
    if (softwarePackage.uninstall_script) {
      items.push(
        <Button key="uninstall" variant="text-link">
          uninstall script
        </Button>
      );
    }
    if (softwarePackage.post_install_script) {
      items.push(
        <Button key="post" variant="text-link">
          post-install script
        </Button>
      );
    }

    if (items.length === 0) return <></>;

    // Helper to join items with commas and "and"
    const joinWithCommasAnd = (elements: JSX.Element[]) => {
      return elements.map((el, idx) => {
        if (idx === 0) return el;
        if (idx === elements.length - 1) return <> and {el}</>;
        return <>, {el}</>;
      });
    };

    return (
      <>
        Next, download your {joinWithCommasAnd(items)} and add{" "}
        {items.length === 1 ? "it" : "them"} to your repository (see above for{" "}
        {items.length === 1 ? "path" : "paths"}).
      </>
    );
  };

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
        <p>
          First, create the YAML file below and save it to your{" "}
          <CustomLink
            url={`${LEARN_MORE_ABOUT_BASE_LINK}/yaml-software`} // TODO: Waiting on confirmation of link
            text="repository"
            newTab
          />
          .
        </p>
        <p>Make sure you reference the package YAML from your team YAML.</p>
        <InputField
          enableCopy
          readOnly
          inputWrapperClass
          name="filename"
          label="Filename"
          value={softwarePackage.name}
        />
        <Editor label="Contents" helpText={renderHelpText()} />

        <div className="modal-cta-wrap">
          <Button onClick={onExit}>Done</Button>
        </div>
      </>
    </Modal>
  );
};

export default ViewYamlModal;

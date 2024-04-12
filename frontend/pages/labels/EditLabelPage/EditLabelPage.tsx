import MainContent from "components/MainContent";
import React from "react";
import ManualLabelForm from "../components/ManualLabelForm";

const baseClass = "edit-label-page";

interface IEditLabelPageProps {}

const EditLabelPage = ({}: IEditLabelPageProps) => {
  // GET LABEL

  // GET HOSTS
  // host;
  return (
    <MainContent className={baseClass}>
      Edit label
      {/* <ManualLabelForm d={hosts} /> */}
    </MainContent>
  );
};

export default EditLabelPage;

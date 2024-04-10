import MainContent from "components/MainContent";
import React from "react";

const baseClass = "edit-label-page";

interface IEditLabelPageProps {}

const EditLabelPage = ({}: IEditLabelPageProps) => {
  return <MainContent className={baseClass}>Edit label</MainContent>;
};

export default EditLabelPage;

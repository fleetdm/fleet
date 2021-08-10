import { IQuery, IQueryFormFields } from "interfaces/query";

const formChanged = (fields: IQueryFormFields, query: IQuery) => {
  return (
    query.name !== fields.name.value ||
    query.description !== fields.description.value ||
    query.query !== fields.query.value ||
    // added 5/17
    query.observer_can_run !== fields.observer_can_run.value
  );
};

const canSaveAsNew = (fields: IQueryFormFields, query: IQuery) => {
  if (!fields.name.value && !fields.description.value) {
    return true;
  }

  if (fields.name.value !== query.name) {
    return true;
  }

  // added 5/17 canSaveAsNew returns true or false based on if observer can run has changed
  if (fields.observer_can_run.value !== query.observer_can_run) {
    return true;
  }

  return false;
};

const canSaveChanges = (fields: IQueryFormFields, query: IQuery) => {
  if (!query.name && !query.description) {
    return false;
  }

  if (formChanged(fields, query)) {
    return true;
  }

  return false;
};

const allPlatforms = { label: "All Platforms", value: "" };
const platformOptions = [
  allPlatforms,
  { label: "macOS", value: "darwin" },
  { label: "Windows", value: "windows" },
  { label: "Ubuntu", value: "ubuntu" },
  { label: "Centos", value: "centos" },
];

export default { allPlatforms, canSaveAsNew, canSaveChanges, platformOptions };

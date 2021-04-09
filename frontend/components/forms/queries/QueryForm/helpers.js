const formChanged = (fields, query) => {
  return (
    query.name !== fields.name.value ||
    query.description !== fields.description.value ||
    query.query !== fields.query.value
  );
};

const canSaveAsNew = (fields, query = {}) => {
  if (!fields.name.value && !fields.description.value) {
    return true;
  }

  if (fields.name.value !== query.name) {
    return true;
  }

  return false;
};

const canSaveChanges = (fields, query = {}) => {
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

import stateEntityGetter from "react-entity-getter";

const pathToEntities = (entityName) => {
  return `entities[${entityName}].data`;
};

export default stateEntityGetter(pathToEntities);
